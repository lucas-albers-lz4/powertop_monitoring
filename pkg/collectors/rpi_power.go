package collectors

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Export the type by capitalizing it
type RPiPowerCollector struct {
	mutex sync.Mutex

	// Power metrics
	watts       *prometheus.Desc
	kilowatts   *prometheus.Desc
	kwhPerDay   *prometheus.Desc
	kwhPerMonth *prometheus.Desc

	// Voltage metrics
	coreVoltage   *prometheus.Desc
	sdramVoltages *prometheus.Desc

	// CPU metrics
	cpuFrequency   *prometheus.Desc
	cpuTemperature *prometheus.Desc

	// Clock frequencies
	clockFrequencies *prometheus.Desc

	// Power states
	powerStates *prometheus.Desc

	// Memory split
	memorySplit *prometheus.Desc

	// Cache values
	model        string
	powerValues  map[string]map[string]float64
	lastCheck    time.Time
	cachedValues map[string]float64
}

// Export the constructor by capitalizing it
func NewRPiPowerCollector() (*RPiPowerCollector, error) {
	c := &RPiPowerCollector{
		watts: prometheus.NewDesc(
			"rpi_power_watts",
			"Current power usage in watts",
			nil, nil,
		),
		kilowatts: prometheus.NewDesc(
			"rpi_power_kilowatts",
			"Current power usage in kilowatts",
			nil, nil,
		),
		kwhPerDay: prometheus.NewDesc(
			"rpi_power_kwh_per_day",
			"Estimated kilowatt hours per day",
			nil, nil,
		),
		kwhPerMonth: prometheus.NewDesc(
			"rpi_power_kwh_per_month",
			"Estimated kilowatt hours per month",
			nil, nil,
		),
		coreVoltage: prometheus.NewDesc(
			"rpi_voltage_core",
			"Core voltage in volts",
			nil, nil,
		),
		sdramVoltages: prometheus.NewDesc(
			"rpi_voltage_sdram",
			"SDRAM voltage readings",
			[]string{"type"}, nil,
		),
		cpuFrequency: prometheus.NewDesc(
			"rpi_cpu_frequency_mhz",
			"CPU frequency in MHz",
			nil, nil,
		),
		cpuTemperature: prometheus.NewDesc(
			"rpi_cpu_temperature_celsius",
			"CPU temperature in Celsius",
			nil, nil,
		),
		clockFrequencies: prometheus.NewDesc(
			"rpi_clock_frequency_hz",
			"Clock frequencies for different components",
			[]string{"component"}, nil,
		),
		powerStates: prometheus.NewDesc(
			"rpi_power_state",
			"Power state indicators (1 for true, 0 for false)",
			[]string{"state"}, nil,
		),
		memorySplit: prometheus.NewDesc(
			"rpi_memory_split_mb",
			"Memory split between ARM and GPU in MB",
			[]string{"type"}, nil,
		),
		powerValues: map[string]map[string]float64{
			"Raspberry Pi 5 Model B": {"idle_power": 2.7, "max_power": 7.5, "max_power_nvme": 15.0},
			"Raspberry Pi 4 Model B": {"idle_power": 3.0, "max_power": 6.0},
			// ... add other models
		},
	}

	return c, nil
}

func (c *RPiPowerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.watts
	ch <- c.kilowatts
	ch <- c.kwhPerDay
	ch <- c.kwhPerMonth
	ch <- c.coreVoltage
	ch <- c.sdramVoltages
	ch <- c.cpuFrequency
	ch <- c.cpuTemperature
	ch <- c.clockFrequencies
	ch <- c.powerStates
	ch <- c.memorySplit
}

func (c *RPiPowerCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Get power metrics
	watts := c.calculatePowerUsage()
	ch <- prometheus.MustNewConstMetric(c.watts, prometheus.GaugeValue, watts)
	ch <- prometheus.MustNewConstMetric(c.kilowatts, prometheus.GaugeValue, watts/1000)
	ch <- prometheus.MustNewConstMetric(c.kwhPerDay, prometheus.GaugeValue, (watts/1000)*24)
	ch <- prometheus.MustNewConstMetric(c.kwhPerMonth, prometheus.GaugeValue, (watts/1000)*24*30)

	// Get voltages
	if voltage, err := c.getVoltage("core"); err == nil {
		ch <- prometheus.MustNewConstMetric(c.coreVoltage, prometheus.GaugeValue, voltage)
	}

	for _, sdramType := range []string{"c", "i", "p"} {
		if voltage, err := c.getVoltage("sdram_" + sdramType); err == nil {
			ch <- prometheus.MustNewConstMetric(c.sdramVoltages, prometheus.GaugeValue, voltage, sdramType)
		}
	}

	// Get CPU metrics
	if freq, err := c.getCPUFrequency(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.cpuFrequency, prometheus.GaugeValue, freq)
	}
	if temp, err := c.getCPUTemperature(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.cpuTemperature, prometheus.GaugeValue, temp)
	}

	// Get clock frequencies
	freqs := c.getClockFrequencies()
	for component, freq := range freqs {
		ch <- prometheus.MustNewConstMetric(c.clockFrequencies, prometheus.GaugeValue, freq, component)
	}

	// Get power states
	states := c.getPowerStates()
	for state, value := range states {
		ch <- prometheus.MustNewConstMetric(c.powerStates, prometheus.GaugeValue, boolToFloat64(value), state)
	}

	// Get memory split
	memSplit := c.getMemorySplit()
	for memType, size := range memSplit {
		ch <- prometheus.MustNewConstMetric(c.memorySplit, prometheus.GaugeValue, float64(size), memType)
	}
}

// Helper functions for RPiPowerCollector

func (c *RPiPowerCollector) runVcgenCmd(args ...string) (string, error) {
	cmd := exec.Command("vcgencmd", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *RPiPowerCollector) calculatePowerUsage() float64 {
	// Get core voltage and CPU frequency as they affect power usage
	voltage, err := c.getVoltage("core")
	if err != nil {
		voltage = 1.0 // fallback
	}

	freq, err := c.getCPUFrequency()
	if err != nil {
		freq = 600.0 // fallback to minimum
	}

	// Basic power calculation (P = V² * f)
	// This is a simplified model, actual power usage would need calibration
	power := voltage * voltage * (freq / 1000.0)
	return power
}

func (c *RPiPowerCollector) getVoltage(voltageType string) (float64, error) {
	output, err := c.runVcgenCmd("measure_volts", voltageType)
	if err != nil {
		return 0, err
	}

	// Remove 'V' suffix and convert to float
	voltage := strings.TrimSuffix(strings.Split(output, "=")[1], "V")
	return strconv.ParseFloat(voltage, 64)
}

func (c *RPiPowerCollector) getCPUFrequency() (float64, error) {
	output, err := c.runVcgenCmd("measure_clock", "arm")
	if err != nil {
		return 0, err
	}

	freqStr := strings.Split(output, "=")[1]
	freq, err := strconv.ParseFloat(freqStr, 64)
	if err != nil {
		return 0, err
	}

	return freq / 1000000.0, nil // Convert Hz to MHz
}

func (c *RPiPowerCollector) getCPUTemperature() (float64, error) {
	output, err := c.runVcgenCmd("measure_temp")
	if err != nil {
		return 0, err
	}

	// Remove 'temp=' prefix and '\'C' suffix
	temp := strings.TrimSuffix(strings.Split(output, "=")[1], "'C")
	return strconv.ParseFloat(temp, 64)
}

func (c *RPiPowerCollector) getClockFrequencies() map[string]float64 {
	frequencies := make(map[string]float64)
	components := []string{"core", "arm", "v3d", "uart", "emmc", "pixel", "hdmi"}

	for _, component := range components {
		output, err := c.runVcgenCmd("measure_clock", component)
		if err != nil {
			continue
		}

		freqStr := strings.Split(output, "=")[1]
		freq, err := strconv.ParseFloat(freqStr, 64)
		if err != nil {
			continue
		}

		frequencies[component] = freq
	}

	return frequencies
}

func (c *RPiPowerCollector) getPowerStates() map[string]bool {
	states := make(map[string]bool)
	output, err := c.runVcgenCmd("get_throttled")
	if err != nil {
		return states
	}

	// Convert hex string to int
	hexStr := strings.Split(output, "=")[1]
	val, err := strconv.ParseInt(strings.TrimPrefix(hexStr, "0x"), 16, 32)
	if err != nil {
		return states
	}

	// Map throttled bits to states
	states["under_voltage_detected"] = val&0x1 != 0
	states["freq_capped"] = val&0x2 != 0
	states["throttling_active"] = val&0x4 != 0
	states["soft_temp_limit"] = val&0x8 != 0
	states["under_voltage_occurred"] = val&0x10000 != 0
	states["freq_capped_occurred"] = val&0x20000 != 0
	states["throttling_occurred"] = val&0x40000 != 0
	states["soft_temp_limit_occurred"] = val&0x80000 != 0

	return states
}

func (c *RPiPowerCollector) getMemorySplit() map[string]int64 {
	split := make(map[string]int64)

	// Get GPU memory
	output, err := c.runVcgenCmd("get_mem", "gpu")
	if err == nil {
		memStr := strings.Split(output, "=")[1]
		memStr = strings.TrimSuffix(memStr, "M")
		if gpu, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			split["gpu"] = gpu
		}
	}

	// Get ARM memory
	output, err = c.runVcgenCmd("get_mem", "arm")
	if err == nil {
		memStr := strings.Split(output, "=")[1]
		memStr = strings.TrimSuffix(memStr, "M")
		if arm, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			split["arm"] = arm
		}
	}

	return split
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
