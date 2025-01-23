package collectors

import (
	"os/exec"
	"testing"
)

func TestRPiPowerCollector(t *testing.T) {
	// Skip if not on RPi
	if _, err := exec.LookPath("vcgencmd"); err != nil {
		t.Skip("Skipping test on non-RPi system")
	}

	collector, err := NewRPiPowerCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Test power calculation
	power := collector.calculatePowerUsage(50.0, 1200.0)
	if power <= 0 {
		t.Error("Power calculation returned invalid value")
	}

	// Test temperature reading
	temp := collector.getTemperature()
	if temp < 0 || temp > 100 {
		t.Error("Temperature reading out of expected range")
	}
}
