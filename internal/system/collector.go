package system

import "time"

const TopProcessCount = 5

// Collector remembers the last reading: CPU usage is the difference
// between two readings.
type Collector struct {
	previousCPU       CPUTimes
	previousProcesses map[processKey]uint64
	gpu               NvidiaGPU
}

func NewCollector() (*Collector, error) {
	cpu, err := ReadCPUTimes()
	if err != nil {
		return nil, err
	}
	processes, err := readProcesses()
	if err != nil {
		return nil, err
	}
	return &Collector{
		previousCPU:       cpu,
		previousProcesses: ticksByProcess(processes),
		gpu:               FindNvidiaGPU(PCIDevicesPattern),
	}, nil
}

func (c *Collector) Collect() (Sample, error) {
	cpu, err := ReadCPUTimes()
	if err != nil {
		return Sample{}, err
	}
	memory, err := ReadMemory()
	if err != nil {
		return Sample{}, err
	}
	load, err := ReadLoadAverage()
	if err != nil {
		return Sample{}, err
	}
	processes, err := readProcesses()
	if err != nil {
		return Sample{}, err
	}

	// /proc/stat adds up all cores. Divide by the number of cores to get
	// the time that passed for one core.
	elapsedTicks := float64(cpu.Total-c.previousCPU.Total) / float64(cpu.Cores)

	frequencies := ReadCoreFrequencies(CPUFreqPattern)
	fastest, limit := FastestCore(frequencies)

	top := topProcesses(c.previousProcesses, processes, elapsedTicks, TopProcessCount)
	// Only for the top 5: no need to read it for every process.
	for i := range top {
		top[i].Command = readCommand(top[i].PID)
		top[i].CoreMHz = frequencies[top[i].core].CurrentMHz
		top[i].CoreMaxMHz = frequencies[top[i].core].MaxMHz
	}

	gpu := c.gpu.Read()

	s := Sample{
		Time:            time.Now(),
		CPUPercent:      CPUUsage(c.previousCPU, cpu),
		LoadAverage:     load,
		FastestCoreMHz:  fastest,
		CoreLimitMHz:    limit,
		MemAvailableKB:  memory.AvailableKB,
		SwapUsedKB:      memory.SwapUsedKB,
		TempCelsius:     ReadCPUTemperature(HwmonPattern),
		GPUState:        gpu.State,
		GPUPercent:      gpu.UsagePercent,
		GPUTempCelsius:  gpu.TempCelsius,
		GPUPowerWatts:   gpu.PowerWatts,
		GPUMemoryMiB:    gpu.MemoryMiB,
		GPUMemoryMaxMiB: gpu.MemoryMaxMiB,
		TopProcesses:    top,
	}

	c.previousCPU = cpu
	c.previousProcesses = ticksByProcess(processes)
	return s, nil
}
