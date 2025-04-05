package incident

import (
	"os"
	"testing"
	"time"

	"ragnaros/internal/system"
)

func TestRecordSampleSavesOneIncidentPerSpike(t *testing.T) {
	dir := t.TempDir()
	recent := NewHistory(120)
	alerts := NewDetector(5)
	start := time.Date(2024, 9, 6, 17, 0, 0, 0, time.UTC)

	// 3 calm samples, then a 10 second spike: only one file expected.
	cpuValues := []float64{20, 30, 40, 95, 96, 97, 98, 99, 99, 99, 99, 99, 99}
	var saved []string
	for i, cpu := range cpuValues {
		s := system.Sample{Time: start.Add(time.Duration(i) * time.Second), CPUPercent: cpu, MemAvailableKB: 1000}
		if cpu > 90 {
			s.Alert = "CPU high"
		}
		path, err := RecordSample(recent, alerts, s, dir)
		if err != nil {
			t.Fatal(err)
		}
		if path != "" {
			saved = append(saved, path)
		}
	}

	if len(saved) != 1 {
		t.Fatalf("saved %d incidents, want 1", len(saved))
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("%d files in dir, want 1", len(entries))
	}

	samples, err := Read(saved[0])
	if err != nil {
		t.Fatal(err)
	}
	// The spike is detected on the 5th high sample, index 7: 8 samples saved.
	if len(samples) != 8 {
		t.Errorf("incident holds %d samples, want 8", len(samples))
	}
	if samples[7].Alert != "CPU high" {
		t.Errorf("alert not saved in the file: %+v", samples[7])
	}
	if !samples[0].Time.Equal(start) || samples[7].CPUPercent != 99 {
		t.Errorf("unexpected content: first %v, last %+v", samples[0].Time, samples[7])
	}
}

func TestReadIncidentRejectsBadLine(t *testing.T) {
	path := t.TempDir() + "/bad.jsonl"
	if err := os.WriteFile(path, []byte("{\"cpu_percent\": 10}\nnot json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Error("expected an error for an invalid line")
	}
}
