package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type memoryInfo struct {
	availableKB uint64
	swapUsedKB  uint64
}

func readMemory() (memoryInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return memoryInfo{}, err
	}
	return parseMemory(string(data))
}

// MemAvailable is the kernel estimate of memory usable without swapping,
// more meaningful than MemFree which ignores reclaimable cache.
func parseMemory(content string) (memoryInfo, error) {
	available, err := meminfoValueKB(content, "MemAvailable")
	if err != nil {
		return memoryInfo{}, err
	}
	swapTotal, err := meminfoValueKB(content, "SwapTotal")
	if err != nil {
		return memoryInfo{}, err
	}
	swapFree, err := meminfoValueKB(content, "SwapFree")
	if err != nil {
		return memoryInfo{}, err
	}
	if swapFree > swapTotal {
		swapFree = swapTotal
	}
	return memoryInfo{availableKB: available, swapUsedKB: swapTotal - swapFree}, nil
}

func meminfoValueKB(content, key string) (uint64, error) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != key+":" {
			continue
		}
		return strconv.ParseUint(fields[1], 10, 64)
	}
	return 0, fmt.Errorf("no %s line in /proc/meminfo", key)
}

func formatKB(kb uint64) string {
	mib := float64(kb) / 1024
	if mib < 1024 {
		return fmt.Sprintf("%.0f MiB", mib)
	}
	return fmt.Sprintf("%.1f GiB", mib/1024)
}
