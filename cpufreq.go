package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const cpuFreqPattern = "/sys/devices/system/cpu/cpu[0-9]*/cpufreq"

// coreFrequency of one core. A busy core far below its maxMHz is often
// throttled by temperature or by the power profile.
type coreFrequency struct {
	currentMHz int
	maxMHz     int
}

// readCoreFrequencies returns the frequencies by core number. It returns an
// empty map when cpufreq is missing, for example in many virtual machines.
func readCoreFrequencies(pattern string) map[int]coreFrequency {
	frequencies := make(map[int]coreFrequency)
	dirs, _ := filepath.Glob(pattern)
	for _, dir := range dirs {
		// dir looks like ".../cpu12/cpufreq": the core number is in the parent name.
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

// fastestCore returns the highest current frequency and the highest possible one.
// Under load, a fastest core well below the limit means the whole CPU is held back.
func fastestCore(frequencies map[int]coreFrequency) (int, int) {
	var fastest, limit int
	for _, frequency := range frequencies {
		fastest = max(fastest, frequency.currentMHz)
		limit = max(limit, frequency.maxMHz)
	}
	return fastest, limit
}
