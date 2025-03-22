package system

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

func TestReadCPUTemperaturePicksCPUSensor(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{
		dir + "/hwmon0/name":        "nvme\n",
		dir + "/hwmon0/temp1_input": "70000\n", // hotter, but not the CPU
		dir + "/hwmon1/name":        "coretemp\n",
		dir + "/hwmon1/temp1_input": "55000\n",
		dir + "/hwmon1/temp2_input": "61000\n",
	})
	if got := ReadCPUTemperature(dir + "/hwmon*"); got != 61 {
		t.Errorf("got %v, want 61", got)
	}
}

func TestReadCPUTemperatureWithoutCPUSensor(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{
		dir + "/hwmon0/name":        "nvme\n",
		dir + "/hwmon0/temp1_input": "70000\n",
	})
	if got := ReadCPUTemperature(dir + "/hwmon*"); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}
