package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readMemAvailableKB() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	return parseMemAvailableKB(string(data))
}

// MemAvailable is the kernel estimate of memory usable without swapping,
// more meaningful than MemFree which ignores reclaimable cache.
func parseMemAvailableKB(content string) (uint64, error) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "MemAvailable:" {
			continue
		}
		return strconv.ParseUint(fields[1], 10, 64)
	}
	return 0, fmt.Errorf("no MemAvailable line in /proc/meminfo")
}

func formatKB(kb uint64) string {
	mib := float64(kb) / 1024
	if mib < 1024 {
		return fmt.Sprintf("%.0f MiB", mib)
	}
	return fmt.Sprintf("%.1f GiB", mib/1024)
}
