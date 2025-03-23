package main

import (
	"time"

	"ragnaros/internal/system"
)

const topProcessCount = 5

// collector remembers the last reading: CPU usage is the difference
// between two readings.
type collector struct {
	previousCPU       cpuTimes
	previousProcesses map[processKey]uint64
	gpu               nvidiaGPU
}

func newCollector() (*collector, error) {
	cpu, err := readCPUTimes()
	if err != nil {
		return nil, err
	}
	processes, err := readProcesses()
	if err != nil {
		return nil, err
	}
	return &collector{
		previousCPU:       cpu,
		previousProcesses: ticksByProcess(processes),
		gpu:               findNvidiaGPU(pciDevicesPattern),
	}, nil
}

func (c *collector) collect() (sample, error) {
	cpu, err := readCPUTimes()
	if err != nil {
		return sample{}, err
	}
	memory, err := system.ReadMemory()
	if err != nil {
		return sample{}, err
	}
	load, err := system.ReadLoadAverage()
	if err != nil {
		return sample{}, err
	}
	processes, err := readProcesses()
	if err != nil {
		return sample{}, err
	}

	// /proc/stat adds up all cores. Divide by the number of cores to get
	// the time that passed for one core.
	elapsedTicks := float64(cpu.total-c.previousCPU.total) / float64(cpu.cores)

	frequencies := system.ReadCoreFrequencies(system.CPUFreqPattern)
	fastest, limit := system.FastestCore(frequencies)

	top := topProcesses(c.previousProcesses, processes, elapsedTicks, topProcessCount)
	// Only for the top 5: no need to read it for every process.
	for i := range top {
		top[i].Command = readCommand(top[i].PID)
		top[i].CoreMHz = frequencies[top[i].core].CurrentMHz
		top[i].CoreMaxMHz = frequencies[top[i].core].MaxMHz
	}

	gpu := c.gpu.read()

	s := sample{
		Time:            time.Now(),
		CPUPercent:      cpuUsage(c.previousCPU, cpu),
		LoadAverage:     load,
		FastestCoreMHz:  fastest,
		CoreLimitMHz:    limit,
		MemAvailableKB:  memory.AvailableKB,
		SwapUsedKB:      memory.SwapUsedKB,
		TempCelsius:     system.ReadCPUTemperature(system.HwmonPattern),
		GPUState:        gpu.state,
		GPUPercent:      gpu.usagePercent,
		GPUTempCelsius:  gpu.tempCelsius,
		GPUPowerWatts:   gpu.powerWatts,
		GPUMemoryMiB:    gpu.memoryMiB,
		GPUMemoryMaxMiB: gpu.memoryMaxMiB,
		TopProcesses:    top,
	}

	c.previousCPU = cpu
	c.previousProcesses = ticksByProcess(processes)
	return s, nil
}
