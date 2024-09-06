package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// cpuTimes holds the cumulative CPU counters read from /proc/stat, in clock ticks.
type cpuTimes struct {
	idle  uint64
	total uint64
	cores int
}

func readCPUTimes() (cpuTimes, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	return parseCPUTimes(string(data))
}

func parseCPUTimes(content string) (cpuTimes, error) {
	var times cpuTimes
	found := false

	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "cpu" {
			idle, total, err := sumCPUFields(fields[1:])
			if err != nil {
				return cpuTimes{}, err
			}
			times.idle = idle
			times.total = total
			found = true
		} else if strings.HasPrefix(fields[0], "cpu") {
			// Lines "cpu0", "cpu1", ... describe one core each.
			times.cores++
		}
	}

	if !found {
		return cpuTimes{}, fmt.Errorf("no global cpu line in /proc/stat")
	}
	if times.cores == 0 {
		times.cores = 1
	}
	return times, nil
}

// Fields order: user nice system idle iowait irq softirq steal guest guest_nice.
// Only the first 8 are summed: guest time is already counted in user and nice.
func sumCPUFields(fields []string) (uint64, uint64, error) {
	var idle, total uint64
	for i, field := range fields {
		if i >= 8 {
			break
		}
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("bad cpu field %q: %w", field, err)
		}
		total += value
		if i == 3 || i == 4 {
			idle += value
		}
	}
	return idle, total, nil
}

// cpuUsage returns the global CPU usage (0 to 100) between two measures.
func cpuUsage(previous, current cpuTimes) float64 {
	if current.total <= previous.total || current.idle < previous.idle {
		return 0
	}
	totalDelta := current.total - previous.total
	idleDelta := current.idle - previous.idle
	if idleDelta > totalDelta {
		return 0
	}
	return float64(totalDelta-idleDelta) / float64(totalDelta) * 100
}
