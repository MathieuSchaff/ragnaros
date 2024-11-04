package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readLoadAverage returns the 1 minute load average: the mean number of tasks
// running or waiting, for CPU or for disk I/O. Compare it to the number of cores.
func readLoadAverage() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	return parseLoadAverage(string(data))
}

// Format: "0.52 0.41 0.30 2/1234 5678", the first value is the last minute.
func parseLoadAverage(content string) (float64, error) {
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty /proc/loadavg")
	}
	return strconv.ParseFloat(fields[0], 64)
}
