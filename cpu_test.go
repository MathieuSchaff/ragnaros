package main

import "testing"

const fakeProcStat = `cpu  100 0 50 800 50 0 0 0 10 0
cpu0 50 0 25 400 25 0 0 0 5 0
cpu1 50 0 25 400 25 0 0 0 5 0
intr 12345
ctxt 6789
`

func TestParseCPUTimes(t *testing.T) {
	times, err := parseCPUTimes(fakeProcStat)
	if err != nil {
		t.Fatal(err)
	}
	if times.total != 1000 {
		t.Errorf("total = %d, want 1000 (guest excluded)", times.total)
	}
	if times.idle != 850 {
		t.Errorf("idle = %d, want 850 (idle + iowait)", times.idle)
	}
	if times.cores != 2 {
		t.Errorf("cores = %d, want 2", times.cores)
	}
}

func TestParseCPUTimesMissingLine(t *testing.T) {
	if _, err := parseCPUTimes("intr 1\n"); err == nil {
		t.Error("expected an error without a cpu line")
	}
}

func TestCPUUsage(t *testing.T) {
	previous := cpuTimes{idle: 800, total: 1000}
	tests := []struct {
		name    string
		current cpuTimes
		want    float64
	}{
		{"half busy", cpuTimes{idle: 850, total: 1100}, 50},
		{"idle", cpuTimes{idle: 900, total: 1100}, 0},
		{"full", cpuTimes{idle: 800, total: 1100}, 100},
		{"no time elapsed", cpuTimes{idle: 800, total: 1000}, 0},
		{"counters went back", cpuTimes{idle: 10, total: 20}, 0},
	}
	for _, test := range tests {
		got := cpuUsage(previous, test.current)
		if got != test.want {
			t.Errorf("%s: cpuUsage = %v, want %v", test.name, got, test.want)
		}
	}
}
