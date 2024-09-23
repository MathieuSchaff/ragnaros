package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadMaxTemperature(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"zone0": "54000\n", "zone1": "71500\n", "zone2": "garbage\n"}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if got := readMaxTemperature(filepath.Join(dir, "zone*")); got != 71.5 {
		t.Errorf("got %v, want 71.5", got)
	}
}

func TestReadMaxTemperatureNoSensor(t *testing.T) {
	if got := readMaxTemperature(filepath.Join(t.TempDir(), "zone*")); got != 0 {
		t.Errorf("got %v, want 0 without sensor", got)
	}
}
