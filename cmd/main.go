package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"net/http"

	"github.com/project-flotta/powertop_container/pkg/collectors"
	"github.com/project-flotta/powertop_container/pkg/stats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

var (
	address = flag.String(
		"address",
		"0.0.0.0:8887",
		"bind address",
	)
	metricsPath = flag.String(
		"metrics-path",
		"/metrics",
		"metrics path",
	)
	enableComparison = flag.Bool(
		"enable-comparison",
		false,
		"Enable both powertop and RPI metrics on Raspberry Pi systems",
	)
	sysInfo       stats.SysInfo
	data          [][]string
	baseLinePower float64
	tunNum        uint32
)

func isRaspberryPi() bool {
	if _, err := exec.LookPath("vcgencmd"); err == nil {
		if data, err := os.ReadFile("/proc/device-tree/model"); err == nil {
			return strings.Contains(strings.ToLower(string(data)), "raspberry pi")
		}
	}
	return false
}

func main() {
	flag.Parse()

	isRPi := isRaspberryPi()
	log.Printf("System is Raspberry Pi: %v", isRPi)

	//register the version collector
	err := prometheus.Register(version.NewCollector("power_metrics_exporter"))
	if err != nil {
		log.Fatalf("failed to register version collector: %v", err)
	}

	// Always register RPI collector if on Raspberry Pi
	if isRPi {
		rpiCollector, err := collectors.NewRPiPowerCollector()
		if err != nil {
			log.Fatalf("failed to create RPI collector: %v", err)
		}
		err = prometheus.Register(rpiCollector)
		if err != nil {
			log.Fatalf("failed to register RPI collector: %v", err)
		}
		log.Printf("Registered RPI power metrics (prefix: rpi_)")
	}

	// Register powertop collector if appropriate
	if !isRPi || (isRPi && *enableComparison) {
		// Original powertop metrics with renamed metrics for comparison
		ptTuCount := promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "powertop_tunables_count",
				Help: "counts the number of tuning available by powertop",
			},
		)
		ptWakeupCount := promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "powertop_wakeup_count",
				Help: "counts the wake up calls per second available by powertop",
			},
		)
		ptCpuUsageCount := promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "powertop_cpu_usage_count",
				Help: "counts the cpu usage in % by powertop",
			},
		)
		ptBaselinePowerCount := promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "powertop_baseline_power_count",
				Help: "counts the baseline power used available by powertop",
			},
		)

		ticker := time.NewTicker(2 * time.Second)
		done := make(chan bool)
		go powerTopStart(
			done,
			ticker,
			ptWakeupCount,
			ptCpuUsageCount,
			ptBaselinePowerCount,
			ptTuCount,
		)
		log.Printf("Registered powertop metrics (prefix: powertop_)")
	}

	//prometheus http handler
	http.Handle(
		*metricsPath,
		promhttp.Handler(),
	)
	http.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			systemInfo := "Standard System"
			if isRPi {
				systemInfo = "Raspberry Pi"
				if *enableComparison {
					systemInfo += " (with comparison metrics)"
				}
			}

			_, err = w.Write(
				[]byte(`<html>
				<head><title>Power Metrics Exporter</title></head>
				<body>
				<h1>Power Metrics Exporter</h1>
				<p>System Type: ` + systemInfo + `</p>
				<p><a href="` + *metricsPath + `">Metrics</a></p>
				<h2>Available Metrics:</h2>
				<ul>` +
					func() string {
						var metrics string
						if isRPi {
							metrics += "<li>RPI metrics (rpi_*)</li>"
						}
						if !isRPi || *enableComparison {
							metrics += "<li>Powertop metrics (powertop_*)</li>"
						}
						return metrics
					}() +
					`</ul>
				</body>
				</html>`),
			)
			if err != nil {
				log.Fatalf("failed to write response: %v", err)
			}
		},
	)

	log.Printf("Starting power metrics exporter on %s", *address)
	err = http.ListenAndServe(*address, nil)
	if err != nil {
		log.Fatalf("failed to bind on %s: %v", *address, err)
	}
}

func powerTopStart(done chan bool, ticker *time.Ticker, ptWakeupCount prometheus.Gauge, ptCpuUsageCount prometheus.Gauge, ptBaselinePowerCount prometheus.Gauge, ptTuCount prometheus.Gauge) {
	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			fmt.Println(
				"Tick at",
				t,
			)
			fmt.Println("command started")
			file, err := tempPowerTopCsvFile()
			defer func(name string) {
				err := os.Remove(name)
				if err != nil {

				}
			}((*file).Name())
			//lock.Lock()
			fmt.Println(file.Name())
			cmd := exec.Command(
				"powertop",
				"--debug",
				"--csv="+file.Name(),
				"--time=1",
			)
			out, err := cmd.Output()
			fmt.Println("to")
			if err != nil {
				log.Printf(
					"%v",
					err,
				)
			}
			fmt.Println("****************")
			fmt.Printf(
				"%s",
				out,
			)
			fmt.Println("****************")
			fmt.Println("opening file")
			data, err := stats.ReadCSV(file.Name())
			fmt.Println("opened")
			if err != nil {
				log.Printf(
					"error in opening the csv file %v",
					err,
				)
			}

			// parse_csv_and_publish(path)
			sysInfo, baseLinePower, tunNum := ParseData(data)

			//Fetch wakeup data
			if sysInfo.Wakeups != 0 {
				ptWakeupCount.Set(sysInfo.Wakeups)
			}

			if sysInfo.CpuUsage != 0 {
				ptCpuUsageCount.Set(sysInfo.CpuUsage)
			}
			if baseLinePower != 0 {
				ptBaselinePowerCount.Set(baseLinePower)
			}

			if baseLinePower != 0 {
				ptBaselinePowerCount.Set(baseLinePower)
			}

			//Fetch no of tunables
			ptTuCount.Set(float64(tunNum))
		}
	}
}

func tempPowerTopCsvFile() (*os.File, error) {
	file, err := ioutil.TempFile(
		"/var/tmp",
		"powertop_report.csv",
	)
	if err != nil {
		fmt.Println("error")
		log.Fatal(err)
	}
	return file, err
}

func ParseData(data [][]string) (stats.SysInfo, float64, uint32) {
	//parsing data
	sysInfo = sysInfo.ParseSysInfo(data)
	baseLineData := stats.ParseBaseLinePower(data)
	parsedTuned := stats.ParseTunables(data)
	tunNum = stats.GeNumOfTunables(parsedTuned)
	if baseLineData != "" {
		baseLinePower = stats.GetBaseLinePower(baseLineData)
	}

	fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
	fmt.Printf(
		"%v",
		sysInfo,
	)
	fmt.Println(baseLinePower)
	fmt.Println(tunNum)
	fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
	return sysInfo, baseLinePower, tunNum
}
