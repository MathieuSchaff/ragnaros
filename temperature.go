package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const hwmonPattern = "/sys/class/hwmon/hwmon*"

// readCPUTemperature returns the hottest value of the Intel CPU sensor
// ("coretemp"), in °C. It returns 0 if the sensor is not found.
func readCPUTemperature(pattern string) float64 {
	dirs, _ := filepath.Glob(pattern)
	for _, dir := range dirs {
		name, err := os.ReadFile(filepath.Join(dir, "name"))
		if err != nil || strings.TrimSpace(string(name)) != "coretemp" {
			continue
		}
		return readMaxTemperature(filepath.Join(dir, "temp*_input"))
	}
	return 0
}

// readMaxTemperature returns the hottest of the files that match pattern.
// The files hold millidegrees: 54000 means 54 °C.
func readMaxTemperature(pattern string) float64 {
	paths, _ := filepath.Glob(pattern)
	var hottest float64
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		milliCelsius, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			continue
		}
		hottest = max(hottest, float64(milliCelsius)/1000)
	}
	return hottest
}
