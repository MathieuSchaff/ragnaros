package system

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const CPUFreqPattern = "/sys/devices/system/cpu/cpu[0-9]*/cpufreq"

// A busy core far below MaxMHz is usually slowed down by heat.
type CoreFrequency struct {
	CurrentMHz int
	MaxMHz     int
}

// ReadCoreFrequencies returns the speed of each core, by core number.
func ReadCoreFrequencies(pattern string) map[int]CoreFrequency {
	frequencies := make(map[int]CoreFrequency)
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
		frequencies[core] = CoreFrequency{CurrentMHz: current, MaxMHz: maximum}
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

// FastestCore returns the speed of the fastest core, and the top speed
// that any core can reach.
func FastestCore(frequencies map[int]CoreFrequency) (int, int) {
	var fastest, limit int
	for _, frequency := range frequencies {
		fastest = max(fastest, frequency.CurrentMHz)
		limit = max(limit, frequency.MaxMHz)
	}
	return fastest, limit
}
