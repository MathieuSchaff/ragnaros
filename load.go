package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readLoadAverage returns the load of the last minute: about how many
// tasks were running or waiting (for the CPU or the disk).
func readLoadAverage() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	return parseLoadAverage(string(data))
}

// Format: "0.52 0.41 0.30 2/1234 5678". The first value is the last minute.
func parseLoadAverage(content string) (float64, error) {
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty /proc/loadavg")
	}
	return strconv.ParseFloat(fields[0], 64)
}
