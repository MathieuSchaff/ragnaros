package main

import "time"

const topProcessCount = 5

// sample is one measure of the machine. It is also one line of an incident file.
type sample struct {
	Time           time.Time      `json:"time"`
	CPUPercent     float64        `json:"cpu_percent"`
	MemAvailableKB uint64         `json:"mem_available_kb"`
	SwapUsedKB     uint64         `json:"swap_used_kb"`
	TempCelsius    float64        `json:"temp_celsius,omitempty"` // 0 when no sensor
	TopProcesses   []processUsage `json:"top_processes"`
}

// collector keeps the previous counters: CPU usages are computed from the
// difference between two successive readings.
type collector struct {
	previousCPU       cpuTimes
	previousProcesses map[processKey]uint64
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
	return &collector{previousCPU: cpu, previousProcesses: ticksByProcess(processes)}, nil
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
	processes, err := readProcesses()
	if err != nil {
		return sample{}, err
	}

	// /proc/stat counts ticks for all cores together; dividing by the number
	// of cores gives the elapsed time of one core, in the same unit as process times.
	elapsedTicks := float64(cpu.total-c.previousCPU.total) / float64(cpu.cores)

	top := topProcesses(c.previousProcesses, processes, elapsedTicks, cpu.cores, topProcessCount)
	// Read the command line only for the few top processes: cheaper than for all.
	for i := range top {
		top[i].Command = readCommand(top[i].PID)
	}

	s := sample{
		Time:           time.Now(),
		CPUPercent:     cpuUsage(c.previousCPU, cpu),
		MemAvailableKB: memory.availableKB,
		SwapUsedKB:     memory.swapUsedKB,
		TempCelsius:    readTemperature(),
		TopProcesses:   top,
	}

	c.previousCPU = cpu
	c.previousProcesses = ticksByProcess(processes)
	return s, nil
}
