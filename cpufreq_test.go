package main

import (
	"path/filepath"
	"testing"
)

func TestReadCoreFrequencies(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{
		dir + "/cpu0/cpufreq/scaling_cur_freq":  "960000\n",
		dir + "/cpu0/cpufreq/cpuinfo_max_freq":  "4700000\n",
		dir + "/cpu12/cpufreq/scaling_cur_freq": "3500000\n",
		dir + "/cpu12/cpufreq/cpuinfo_max_freq": "3500000\n",
		dir + "/cpu3/cpufreq/scaling_cur_freq":  "broken\n",
		dir + "/cpu3/cpufreq/cpuinfo_max_freq":  "4700000\n",
	})

	got := readCoreFrequencies(filepath.Join(dir, "cpu[0-9]*", "cpufreq"))

	if len(got) != 2 {
		t.Fatalf("got %+v, want cores 0 and 12 only", got)
	}
	if got[0] != (coreFrequency{currentMHz: 960, maxMHz: 4700}) {
		t.Errorf("core 0 = %+v", got[0])
	}
	if got[12] != (coreFrequency{currentMHz: 3500, maxMHz: 3500}) {
		t.Errorf("core 12 = %+v", got[12])
	}
}

func TestFastestCore(t *testing.T) {
	frequencies := map[int]coreFrequency{
		0: {currentMHz: 960, maxMHz: 4700},
		1: {currentMHz: 2697, maxMHz: 4700},
		2: {currentMHz: 2000, maxMHz: 3500},
	}
	fastest, limit := fastestCore(frequencies)
	if fastest != 2697 || limit != 4700 {
		t.Errorf("got %d / %d, want 2697 / 4700", fastest, limit)
	}
}

func TestFastestCoreWithoutCpufreq(t *testing.T) {
	if fastest, limit := fastestCore(nil); fastest != 0 || limit != 0 {
		t.Errorf("got %d / %d, want 0 / 0", fastest, limit)
	}
}
