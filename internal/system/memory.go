package system

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MemoryInfo struct {
	AvailableKB uint64
	SwapUsedKB  uint64
}

func ReadMemory() (MemoryInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemoryInfo{}, err
	}
	return parseMemory(string(data))
}

// MemAvailable is the memory programs can still use. MemFree is lower
// because it ignores the cache, which Linux frees when needed.
func parseMemory(content string) (MemoryInfo, error) {
	available, err := meminfoValueKB(content, "MemAvailable")
	if err != nil {
		return MemoryInfo{}, err
	}
	swapTotal, err := meminfoValueKB(content, "SwapTotal")
	if err != nil {
		return MemoryInfo{}, err
	}
	swapFree, err := meminfoValueKB(content, "SwapFree")
	if err != nil {
		return MemoryInfo{}, err
	}
	if swapFree > swapTotal {
		swapFree = swapTotal
	}
	return MemoryInfo{AvailableKB: available, SwapUsedKB: swapTotal - swapFree}, nil
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

func FormatKB(kb uint64) string {
	mib := float64(kb) / 1024
	if mib < 1024 {
		return fmt.Sprintf("%.0f MiB", mib)
	}
	return fmt.Sprintf("%.1f GiB", mib/1024)
}
