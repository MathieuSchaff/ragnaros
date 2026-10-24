// Package system reads the state of the machine from /proc, /sys and nvidia-smi.
package system

import "time"

// Sample is one measure of the machine. It is also one line of an incident file.
type Sample struct {
	Time            time.Time      `json:"time"`
	CPUPercent      float64        `json:"cpu_percent"`
	LoadAverage     float64        `json:"load_1m"`
	FastestCoreMHz  int            `json:"fastest_core_mhz,omitempty"`
	CoreLimitMHz    int            `json:"core_limit_mhz,omitempty"`
	MemAvailableKB  uint64         `json:"mem_available_kb"`
	SwapUsedKB      uint64         `json:"swap_used_kb"`
	TempCelsius     float64        `json:"temp_celsius,omitempty"` // 0 when no sensor
	GPUState        string         `json:"gpu_state,omitempty"`    // "active", "asleep" or "no data", "" without GPU
	GPUPercent      float64        `json:"gpu_percent,omitempty"`
	GPUTempCelsius  float64        `json:"gpu_temp_celsius,omitempty"`
	GPUPowerWatts   float64        `json:"gpu_power_watts,omitempty"`
	GPUMemoryMiB    float64        `json:"gpu_memory_mib,omitempty"`
	GPUMemoryMaxMiB float64        `json:"gpu_memory_max_mib,omitempty"`
	TopProcesses    []ProcessUsage `json:"top_processes"`
	Alert           string         `json:"alert,omitempty"` // why the sample is high, "" if normal
}
