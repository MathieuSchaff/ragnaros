package system

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
	gpu := FindNvidiaGPU(dir + "/*")
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

	got := NvidiaGPU{pciDir: dir, smiPath: smi}.Read()

	if got != (GPUReading{State: "asleep"}) {
		t.Errorf("got %+v, want asleep only", got)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("nvidia-smi was called and would have woken the GPU")
	}
}

func TestReadActiveGPU(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{dir + "/power/runtime_status": "active\n"})
	smi := fakeNvidiaSmi(t, "echo '20, 54, 16.66, 43, 6144'")

	got := NvidiaGPU{pciDir: dir, smiPath: smi}.Read()

	want := GPUReading{State: "active", UsagePercent: 20, TempCelsius: 54, PowerWatts: 16.66, MemoryMiB: 43, MemoryMaxMiB: 6144}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseNvidiaSmiMissingValues(t *testing.T) {
	got := parseNvidiaSmi("35, 61, [N/A]\n12, 40, 5\n")
	if got.UsagePercent != 35 || got.TempCelsius != 61 || got.PowerWatts != 0 {
		t.Errorf("got %+v, want first GPU with power 0", got)
	}
}

func TestReadGPUWhenNvidiaSmiFails(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{dir + "/power/runtime_status": "active\n"})
	smi := fakeNvidiaSmi(t, "exit 1")

	got := NvidiaGPU{pciDir: dir, smiPath: smi}.Read()

	if got != (GPUReading{State: "no data"}) {
		t.Errorf("got %+v, want no data instead of zeros", got)
	}
}
