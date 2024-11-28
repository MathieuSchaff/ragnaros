package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const cpuFreqPattern = "/sys/devices/system/cpu/cpu[0-9]*/cpufreq"

// A busy core far below maxMHz is usually slowed down by heat.
type coreFrequency struct {
	currentMHz int
	maxMHz     int
}

// readCoreFrequencies returns the speed of each core, by core number.
func readCoreFrequencies(pattern string) map[int]coreFrequency {
	frequencies := make(map[int]coreFrequency)
	dirs, _ := filepath.Glob(pattern)
	for _, dir := range dirs {
		// dir looks like ".../cpu12/cpufreq": take 12 from the parent folder.
		core, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(filepath.Dir(dir)), "cpu"))
		if err != nil {
			continue
		}
		current := readKHzAsMHz(filepath.Join(dir, "scaling_cur_freq"))
		maximum := readKHzAsMHz(filepath.Join(dir, "cpuinfo_max_freq"))
		if current == 0 || maximum == 0 {
			continue
		}
		frequencies[core] = coreFrequency{currentMHz: current, maxMHz: maximum}
	}
	return frequencies
}

// readKHzAsMHz returns 0 when the file is missing or invalid.
func readKHzAsMHz(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	kHz, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return kHz / 1000
}

// fastestCore returns the speed of the fastest core, and the top speed
// that any core can reach.
func fastestCore(frequencies map[int]coreFrequency) (int, int) {
	var fastest, limit int
	for _, frequency := range frequencies {
		fastest = max(fastest, frequency.currentMHz)
		limit = max(limit, frequency.maxMHz)
	}
	return fastest, limit
}
