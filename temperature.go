package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	hwmonPattern       = "/sys/class/hwmon/hwmon*"
	thermalZonePattern = "/sys/class/thermal/thermal_zone*/temp"
)

// Drivers that expose the CPU temperature: Intel, then AMD.
var cpuSensorNames = []string{"coretemp", "k10temp", "zenpower"}

// readTemperature prefers the CPU sensor and falls back to the hottest
// thermal zone, which may be another component (Wi-Fi, battery...).
func readTemperature() float64 {
	if celsius := readCPUTemperature(hwmonPattern); celsius > 0 {
		return celsius
	}
	return readMaxTemperature(thermalZonePattern)
}

// readCPUTemperature returns the hottest value of the first CPU hwmon sensor, or 0.
func readCPUTemperature(pattern string) float64 {
	dirs, _ := filepath.Glob(pattern)
	for _, dir := range dirs {
		name, err := os.ReadFile(filepath.Join(dir, "name"))
		if err != nil || !slices.Contains(cpuSensorNames, strings.TrimSpace(string(name))) {
			continue
		}
		return readMaxTemperature(filepath.Join(dir, "temp*_input"))
	}
	return 0
}

// readMaxTemperature returns the hottest of the files matching pattern, in °C,
// or 0 when none is readable. Sensors vary between machines, so a missing one
// is not an error. Values are in millidegrees, both in hwmon and thermal zones.
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
