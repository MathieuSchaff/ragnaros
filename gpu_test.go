package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindNvidiaGPU(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{
		dir + "/0000:00:02.0/vendor": "0x8086\n", // Intel GPU
		dir + "/0000:00:02.0/class":  "0x030000\n",
		dir + "/0000:01:00.1/vendor": "0x10de\n", // NVIDIA audio function
		dir + "/0000:01:00.1/class":  "0x040300\n",
		dir + "/0000:01:00.0/vendor": "0x10de\n",
		dir + "/0000:01:00.0/class":  "0x030000\n",
	})
	gpu := findNvidiaGPU(dir + "/*")
	if filepath.Base(gpu.pciDir) != "0000:01:00.0" {
		t.Errorf("pciDir = %q, want the NVIDIA display controller", gpu.pciDir)
	}
}

func TestReadAsleepGPUDoesNotRunNvidiaSmi(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{dir + "/power/runtime_status": "suspended\n"})
	// A fake nvidia-smi that would fail the test if it were called.
	marker := filepath.Join(dir, "called")
	smi := fakeNvidiaSmi(t, "touch "+marker+"\necho '99, 99, 99'")

	got := nvidiaGPU{pciDir: dir, smiPath: smi}.read()

	if got != (gpuReading{state: "asleep"}) {
		t.Errorf("got %+v, want asleep only", got)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("nvidia-smi was called and would have woken the GPU")
	}
}

func TestReadActiveGPU(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{dir + "/power/runtime_status": "active\n"})
	smi := fakeNvidiaSmi(t, "echo '20, 54, 16.66'")

	got := nvidiaGPU{pciDir: dir, smiPath: smi}.read()

	want := gpuReading{state: "active", usagePercent: 20, tempCelsius: 54, powerWatts: 16.66}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseNvidiaSmiMissingValues(t *testing.T) {
	got := parseNvidiaSmi("35, 61, [N/A]\n12, 40, 5\n")
	if got.usagePercent != 35 || got.tempCelsius != 61 || got.powerWatts != 0 {
		t.Errorf("got %+v, want first GPU with power 0", got)
	}
}

func fakeNvidiaSmi(t *testing.T, body string) string {
	path := filepath.Join(t.TempDir(), "nvidia-smi")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
