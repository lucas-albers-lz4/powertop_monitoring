package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/project-flotta/powertop_container/pkg/stats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	//"log"
	"net/http"
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
	sysInfo stats.SysInfo
	data          [][]string
	baseLinePower float64
	tunNum        uint32
)

func main() {

	flag.Parse()

	//register the collector
	err := prometheus.Register(version.NewCollector("powertop_tunable_exporter"))
	if err != nil {
		log.Fatalf(
			"failed to register : %v",
			err,
		)
	}

	if err != nil {
		log.Fatalf(
			"failed to create collector: %v",
			err,
		)
	}

	//prometheus http handler
	go func() {
		http.Handle(
			*metricsPath,
			promhttp.Handler(),
		)
		http.HandleFunc(
			"/",
			func(w http.ResponseWriter, r *http.Request) {
				_, err = w.Write(
					[]byte(`<html>}
	fmt.Println("exporter call over")
}
			<head><title>PowerTop Tunable Exporter</title></head>
			<body>
			<h1>Tunable Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`),
				)
				if err != nil {
					log.Fatalf(
						"failed to write response: %v",
						err,
					)
				}
			},
		)

		err = http.ListenAndServe(
			*address,
			nil,
		)
		if err != nil {
			log.Fatalf(
				"failed to bind on %s: %v",
				*address,
				err,
			)
		}
		fmt.Println("exporter call over")
	}()

	//Prometheus Metrics using Gauge
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
	for {
		go powerTopStart(
			done,
			ticker,
			ptWakeupCount,
			ptCpuUsageCount,
			ptBaselinePowerCount,
			ptTuCount,
		)
		time.Sleep(2 * time.Second)
		done <- true
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
