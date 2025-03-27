package system

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	PCIDevicesPattern = "/sys/bus/pci/devices/*"
	nvidiaVendorID    = "0x10de"
	nvidiaSmiTimeout  = time.Second
)

type GPUReading struct {
	State        string // "active", "asleep" or "no data", "" without GPU
	UsagePercent float64
	TempCelsius  float64
	PowerWatts   float64
	MemoryMiB    float64 // video memory used
	MemoryMaxMiB float64
}

// NvidiaGPU is found once at start. smiPath is "" when nvidia-smi is not installed.
type NvidiaGPU struct {
	pciDir  string
	smiPath string
}

func FindNvidiaGPU(pattern string) NvidiaGPU {
	dirs, _ := filepath.Glob(pattern)
	for _, dir := range dirs {
		vendor := readTrimmed(filepath.Join(dir, "vendor"))
		class := readTrimmed(filepath.Join(dir, "class"))
		// Class 0x03... means a graphics card.
		if vendor == nvidiaVendorID && strings.HasPrefix(class, "0x03") {
			smiPath, _ := exec.LookPath("nvidia-smi")
			return NvidiaGPU{pciDir: dir, smiPath: smiPath}
		}
	}
	return NvidiaGPU{}
}

func (gpu NvidiaGPU) Read() GPUReading {
	if gpu.pciDir == "" {
		return GPUReading{}
	}

	// Calling nvidia-smi wakes up a sleeping GPU, and the awake GPU heats
	// the CPU (they share the same heatpipe). So check first if it sleeps.
	state := readTrimmed(filepath.Join(gpu.pciDir, "power", "runtime_status"))
	if state == "suspended" {
		return GPUReading{State: "asleep"}
	}
	if gpu.smiPath == "" {
		return GPUReading{State: "no data"}
	}

	// Give up after 1 second if the driver is stuck.
	ctx, cancel := context.WithTimeout(context.Background(), nvidiaSmiTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, gpu.smiPath,
		"--query-gpu=utilization.gpu,temperature.gpu,power.draw,memory.used,memory.total",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return GPUReading{State: "no data"}
	}
	reading := parseNvidiaSmi(string(output))
	reading.State = "active"
	return reading
}

// parseNvidiaSmi reads "20, 54, 16.66, 43, 6144": usage, temperature, power,
// memory used and memory total. A value shown as "[N/A]" stays 0.
func parseNvidiaSmi(output string) GPUReading {
	firstLine, _, _ := strings.Cut(output, "\n")
	values := strings.Split(firstLine, ",")
	for len(values) < 5 {
		values = append(values, "")
	}
	return GPUReading{
		UsagePercent: parseOptionalFloat(values[0]),
		TempCelsius:  parseOptionalFloat(values[1]),
		PowerWatts:   parseOptionalFloat(values[2]),
		MemoryMiB:    parseOptionalFloat(values[3]),
		MemoryMaxMiB: parseOptionalFloat(values[4]),
	}
}

func parseOptionalFloat(text string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return 0
	}
	return value
}

// readTrimmed returns "" when the file is missing.
func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
