package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const thermalZonePattern = "/sys/class/thermal/thermal_zone*/temp"

// readMaxTemperature returns the hottest sensor in °C, or 0 when no sensor
// is readable. Sensors vary between machines, so a missing one is not an error.
func readMaxTemperature(pattern string) float64 {
	paths, _ := filepath.Glob(pattern)
	var hottest float64
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			// Some zones refuse reads, for example when the device sleeps.
			continue
		}
		milliCelsius, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			continue
		}
		celsius := float64(milliCelsius) / 1000
		if celsius > hottest {
			hottest = celsius
		}
	}
	return hottest
}
