package main

import "time"

const topProcessCount = 5

// sample is one measure of the machine. It is also one line of an incident file.
type sample struct {
	Time           time.Time      `json:"time"`
	CPUPercent     float64        `json:"cpu_percent"`
	LoadAverage    float64        `json:"load_1m"`
	FastestCoreMHz int            `json:"fastest_core_mhz,omitempty"`
	CoreLimitMHz   int            `json:"core_limit_mhz,omitempty"`
	MemAvailableKB uint64         `json:"mem_available_kb"`
	SwapUsedKB     uint64         `json:"swap_used_kb"`
	TempCelsius    float64        `json:"temp_celsius,omitempty"` // 0 when no sensor
	GPUState       string         `json:"gpu_state,omitempty"`    // "active", "asleep" or "no data", "" without GPU
	GPUPercent     float64        `json:"gpu_percent,omitempty"`
	GPUTempCelsius float64        `json:"gpu_temp_celsius,omitempty"`
	GPUPowerWatts  float64        `json:"gpu_power_watts,omitempty"`
	GPUMemoryMiB   float64        `json:"gpu_memory_mib,omitempty"`
	GPUMemoryMax   float64        `json:"gpu_memory_max_mib,omitempty"`
	TopProcesses   []processUsage `json:"top_processes"`
	Alert          string         `json:"alert,omitempty"` // why the sample is high, "" if normal
}

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

	// /proc/stat adds up all cores. Divide by the number of cores to get
	// the time that passed for one core.
	elapsedTicks := float64(cpu.total-c.previousCPU.total) / float64(cpu.cores)

	frequencies := readCoreFrequencies(cpuFreqPattern)
	fastest, limit := fastestCore(frequencies)

	top := topProcesses(c.previousProcesses, processes, elapsedTicks, topProcessCount)
	// Only for the top 5: no need to read it for every process.
	for i := range top {
		top[i].Command = readCommand(top[i].PID)
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
		GPUMemoryMiB:   gpu.memoryMiB,
		GPUMemoryMax:   gpu.memoryMaxMiB,
		TopProcesses:   top,
	}

	c.previousCPU = cpu
	c.previousProcesses = ticksByProcess(processes)
	return s, nil
}
