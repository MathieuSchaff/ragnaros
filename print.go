package main

import (
	"fmt"
	"io"
	"os"

	"ragnaros/internal/system"
)

// printSaveResult prints the result. A failed save does not stop watch.
func printSaveResult(path string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ragnaros: cannot save incident:", err)
	} else if path != "" {
		fmt.Println("Incident saved:", path)
	}
}

// formatCoreFrequency keeps the same width so the commands line up.
func formatCoreFrequency(p system.ProcessUsage) string {
	if p.CoreMaxMHz == 0 {
		return fmt.Sprintf("%14s", "")
	}
	return fmt.Sprintf("%5d/%4d MHz", p.CoreMHz, p.CoreMaxMHz)
}

func printSample(w io.Writer, s system.Sample) {
	fmt.Fprintf(w, "\n%s  CPU %5.1f %%  load %.2f  free memory %s  swap %s",
		s.Time.Format("15:04:05"), s.CPUPercent, s.LoadAverage, system.FormatKB(s.MemAvailableKB), system.FormatKB(s.SwapUsedKB))
	if s.TempCelsius > 0 {
		fmt.Fprintf(w, "  %.0f °C", s.TempCelsius)
	}
	if s.CoreLimitMHz > 0 {
		fmt.Fprintf(w, "  speed %d/%d MHz", s.FastestCoreMHz, s.CoreLimitMHz)
	}
	if s.GPUState == "active" {
		fmt.Fprintf(w, "  GPU %.0f %% %.0f °C %.0f W %.0f/%.0f MiB",
			s.GPUPercent, s.GPUTempCelsius, s.GPUPowerWatts, s.GPUMemoryMiB, s.GPUMemoryMaxMiB)
	} else if s.GPUState != "" {
		fmt.Fprintf(w, "  GPU %s", s.GPUState)
	}
	fmt.Fprintln(w)
	for _, p := range s.TopProcesses {
		fmt.Fprintf(w, "  %7d  %6.1f %%  %s  %s\n", p.PID, p.CPUPercent, formatCoreFrequency(p), p.Label())
	}
}
