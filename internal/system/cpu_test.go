package system

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
	if times.Total != 1000 {
		t.Errorf("total = %d, want 1000 (guest excluded)", times.Total)
	}
	if times.Idle != 850 {
		t.Errorf("idle = %d, want 850 (idle + iowait)", times.Idle)
	}
	if times.Cores != 2 {
		t.Errorf("cores = %d, want 2", times.Cores)
	}
}

func TestParseCPUTimesMissingLine(t *testing.T) {
	if _, err := parseCPUTimes("intr 1\n"); err == nil {
		t.Error("expected an error without a cpu line")
	}
}

func TestCPUUsage(t *testing.T) {
	previous := CPUTimes{Idle: 800, Total: 1000}
	tests := []struct {
		name    string
		current CPUTimes
		want    float64
	}{
		{"half busy", CPUTimes{Idle: 850, Total: 1100}, 50},
		{"idle", CPUTimes{Idle: 900, Total: 1100}, 0},
		{"full", CPUTimes{Idle: 800, Total: 1100}, 100},
		{"no time elapsed", CPUTimes{Idle: 800, Total: 1000}, 0},
		{"counters went back", CPUTimes{Idle: 10, Total: 20}, 0},
	}
	for _, test := range tests {
		got := CPUUsage(previous, test.current)
		if got != test.want {
			t.Errorf("%s: CPUUsage = %v, want %v", test.name, got, test.want)
		}
	}
}
