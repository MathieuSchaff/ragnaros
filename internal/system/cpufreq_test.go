package system

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

	got := ReadCoreFrequencies(filepath.Join(dir, "cpu[0-9]*", "cpufreq"))

	if len(got) != 2 {
		t.Fatalf("got %+v, want cores 0 and 12 only", got)
	}
	if got[0] != (CoreFrequency{CurrentMHz: 960, MaxMHz: 4700}) {
		t.Errorf("core 0 = %+v", got[0])
	}
	if got[12] != (CoreFrequency{CurrentMHz: 3500, MaxMHz: 3500}) {
		t.Errorf("core 12 = %+v", got[12])
	}
}

func TestFastestCore(t *testing.T) {
	frequencies := map[int]CoreFrequency{
		0: {CurrentMHz: 960, MaxMHz: 4700},
		1: {CurrentMHz: 2697, MaxMHz: 4700},
		2: {CurrentMHz: 2000, MaxMHz: 3500},
	}
	fastest, limit := FastestCore(frequencies)
	if fastest != 2697 || limit != 4700 {
		t.Errorf("got %d / %d, want 2697 / 4700", fastest, limit)
	}
}

func TestFastestCoreWithoutCpufreq(t *testing.T) {
	if fastest, limit := FastestCore(nil); fastest != 0 || limit != 0 {
		t.Errorf("got %d / %d, want 0 / 0", fastest, limit)
	}
}
