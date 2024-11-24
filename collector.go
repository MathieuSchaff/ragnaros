package main

import "time"

const topProcessCount = 5

// sample is one measure of the machine. It is also one line of an incident file.
type sample struct {
	Time           time.Time      `json:"time"`
	CPUPercent     float64        `json:"cpu_percent"`
	LoadAverage    float64        `json:"load_1m"`
	FastestCoreMHz int            `json:"fastest_core_mhz,omitempty"` // 0 without cpufreq
	CoreLimitMHz   int            `json:"core_limit_mhz,omitempty"`
	MemAvailableKB uint64         `json:"mem_available_kb"`
	SwapUsedKB     uint64         `json:"swap_used_kb"`
	TempCelsius    float64        `json:"temp_celsius,omitempty"` // 0 when no sensor
	GPUState       string         `json:"gpu_state,omitempty"`    // "" without NVIDIA GPU
	GPUPercent     float64        `json:"gpu_percent,omitempty"`
	GPUTempCelsius float64        `json:"gpu_temp_celsius,omitempty"`
	GPUPowerWatts  float64        `json:"gpu_power_watts,omitempty"`
	TopProcesses   []processUsage `json:"top_processes"`
}

// collector keeps the previous counters: CPU usages are computed from the
// difference between two successive readings.
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
	memory, err := readMemory()
	if err != nil {
		return sample{}, err
	}
	load, err := readLoadAverage()
	if err != nil {
		return sample{}, err
	}
	processes, err := readProcesses()
	if err != nil {
		return sample{}, err
	}

	// /proc/stat counts ticks for all cores together; dividing by the number
	// of cores gives the elapsed time of one core, in the same unit as process times.
	elapsedTicks := float64(cpu.total-c.previousCPU.total) / float64(cpu.cores)

	frequencies := readCoreFrequencies(cpuFreqPattern)
	fastest, limit := fastestCore(frequencies)

	top := topProcesses(c.previousProcesses, processes, elapsedTicks, cpu.cores, topProcessCount)
	// Read the command line only for the few top processes: cheaper than for all.
	for i := range top {
		top[i].Command = readCommand(top[i].PID)
		// A single-threaded program is limited by the speed of its own core.
		top[i].CoreMHz = frequencies[top[i].core].currentMHz
		top[i].CoreMaxMHz = frequencies[top[i].core].maxMHz
	}

	gpu := c.gpu.read()

	s := sample{
		Time:           time.Now(),
		CPUPercent:     cpuUsage(c.previousCPU, cpu),
		LoadAverage:    load,
		FastestCoreMHz: fastest,
		CoreLimitMHz:   limit,
		MemAvailableKB: memory.availableKB,
		SwapUsedKB:     memory.swapUsedKB,
		TempCelsius:    readCPUTemperature(hwmonPattern),
		GPUState:       gpu.state,
		GPUPercent:     gpu.usagePercent,
		GPUTempCelsius: gpu.tempCelsius,
		GPUPowerWatts:  gpu.powerWatts,
		TopProcesses:   top,
	}

	c.previousCPU = cpu
	c.previousProcesses = ticksByProcess(processes)
	return s, nil
}
