package system

import (
	"os"
	"testing"
	"time"
)

// Reads the real /proc twice, 100 ms apart, without creating any load.
func TestCollectOnRealProc(t *testing.T) {
	if _, err := os.Stat("/proc/stat"); err != nil {
		t.Skip("no /proc on this system")
	}
	c, err := NewCollector()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	s, err := c.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if s.CPUPercent < 0 || s.CPUPercent > 100 {
		t.Errorf("CPUPercent = %v, want 0 to 100", s.CPUPercent)
	}
	if s.MemAvailableKB == 0 {
		t.Error("MemAvailableKB is 0")
	}
	if len(s.TopProcesses) > TopProcessCount {
		t.Errorf("%d top processes, want at most %d", len(s.TopProcesses), TopProcessCount)
	}
}
