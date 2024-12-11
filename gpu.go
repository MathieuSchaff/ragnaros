package main

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
	pciDevicesPattern = "/sys/bus/pci/devices/*"
	nvidiaVendorID    = "0x10de"
	nvidiaSmiTimeout  = time.Second
)

type gpuReading struct {
	state        string // "active", "asleep" or "no data", "" without GPU
	usagePercent float64
	tempCelsius  float64
	powerWatts   float64
	memoryMiB    float64 // video memory used
	memoryMaxMiB float64
}

// nvidiaGPU is found once at start. smiPath is "" when nvidia-smi is not installed.
type nvidiaGPU struct {
	pciDir  string
	smiPath string
}

func findNvidiaGPU(pattern string) nvidiaGPU {
	dirs, _ := filepath.Glob(pattern)
	for _, dir := range dirs {
		vendor := readTrimmed(filepath.Join(dir, "vendor"))
		class := readTrimmed(filepath.Join(dir, "class"))
		// Class 0x03... means a graphics card.
		if vendor == nvidiaVendorID && strings.HasPrefix(class, "0x03") {
			smiPath, _ := exec.LookPath("nvidia-smi")
			return nvidiaGPU{pciDir: dir, smiPath: smiPath}
		}
	}
	return nvidiaGPU{}
}

func (gpu nvidiaGPU) read() gpuReading {
	if gpu.pciDir == "" {
		return gpuReading{}
	}

	// Calling nvidia-smi wakes up a sleeping GPU, and the awake GPU heats
	// the CPU (they share the same heatpipe). So check first if it sleeps.
	state := readTrimmed(filepath.Join(gpu.pciDir, "power", "runtime_status"))
	if state == "suspended" {
		return gpuReading{state: "asleep"}
	}
	if gpu.smiPath == "" {
		return gpuReading{state: "no data"}
	}

	// Give up after 1 second if the driver is stuck.
	ctx, cancel := context.WithTimeout(context.Background(), nvidiaSmiTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, gpu.smiPath,
		"--query-gpu=utilization.gpu,temperature.gpu,power.draw,memory.used,memory.total",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return gpuReading{state: "no data"}
	}
	reading := parseNvidiaSmi(string(output))
	reading.state = "active"
	return reading
}

// parseNvidiaSmi reads "20, 54, 16.66, 43, 6144" (usage, temperature, power,
// memory used, memory total).
// A value shown as "[N/A]" stays 0.
func parseNvidiaSmi(output string) gpuReading {
	firstLine, _, _ := strings.Cut(output, "\n")
	values := strings.Split(firstLine, ",")
	for len(values) < 5 {
		values = append(values, "")
	}
	return gpuReading{
		usagePercent: parseOptionalFloat(values[0]),
		tempCelsius:  parseOptionalFloat(values[1]),
		powerWatts:   parseOptionalFloat(values[2]),
		memoryMiB:    parseOptionalFloat(values[3]),
		memoryMaxMiB: parseOptionalFloat(values[4]),
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
