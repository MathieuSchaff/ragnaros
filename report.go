package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

const reportProcessCount = 5

// processSummary sums up one process over the whole incident.
type processSummary struct {
	pid         int
	name        string
	label       string // command line if known, else name
	appearances int
	totalCPU    float64 // sum of the per-second percentages
	maxCPU      float64
	coreMHzSum  int // to get the average speed
	coreMHzSeen int
	coreMaxMHz  int
}

func (p processSummary) averageCoreMHz() int {
	if p.coreMHzSeen == 0 {
		return 0
	}
	return p.coreMHzSum / p.coreMHzSeen
}

type incidentReport struct {
	start, end         time.Time
	sampleCount        int
	peakCPU            float64
	peakCPUTime        time.Time
	peakLoad           float64
	peakLoadTime       time.Time
	minMemKB           uint64
	minMemTime         time.Time
	maxSwapKB          uint64
	maxSwapTime        time.Time
	peakTemp           float64 // 0 when the incident has no temperature
	peakTempTime       time.Time
	minFastestCoreMHz  int
	minFastestCoreTime time.Time
	coreLimitMHz       int
	gpuSamples         int // samples with an NVIDIA GPU state
	gpuAsleepSamples   int
	peakGPUPercent     float64
	peakGPUTime        time.Time
	maxGPUTempCelsius  float64
	maxGPUPowerWatts   float64
	topProcesses       []processSummary
}

func runReport(path string) error {
	samples, err := readIncident(path)
	if err != nil {
		return err
	}
	report, err := buildReport(samples)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	fmt.Println("Incident file:", path)
	printReport(os.Stdout, report)
	return nil
}

func buildReport(samples []sample) (incidentReport, error) {
	if len(samples) == 0 {
		return incidentReport{}, fmt.Errorf("no samples")
	}

	first := samples[0]
	report := incidentReport{
		start:        first.Time,
		end:          first.Time,
		sampleCount:  len(samples),
		peakCPU:      first.CPUPercent,
		peakCPUTime:  first.Time,
		peakLoad:     first.LoadAverage,
		peakLoadTime: first.Time,
		minMemKB:     first.MemAvailableKB,
		minMemTime:   first.Time,
	}

	summaries := make(map[int]*processSummary)
	for _, s := range samples {
		if s.Time.Before(report.start) {
			report.start = s.Time
		}
		if s.Time.After(report.end) {
			report.end = s.Time
		}
		if s.CPUPercent > report.peakCPU {
			report.peakCPU = s.CPUPercent
			report.peakCPUTime = s.Time
		}
		if s.LoadAverage > report.peakLoad {
			report.peakLoad = s.LoadAverage
			report.peakLoadTime = s.Time
		}
		if s.MemAvailableKB < report.minMemKB {
			report.minMemKB = s.MemAvailableKB
			report.minMemTime = s.Time
		}
		if s.SwapUsedKB > report.maxSwapKB {
			report.maxSwapKB = s.SwapUsedKB
			report.maxSwapTime = s.Time
		}
		if s.TempCelsius > report.peakTemp {
			report.peakTemp = s.TempCelsius
			report.peakTempTime = s.Time
		}
		if s.CoreLimitMHz > 0 && (report.minFastestCoreMHz == 0 || s.FastestCoreMHz < report.minFastestCoreMHz) {
			report.minFastestCoreMHz = s.FastestCoreMHz
			report.minFastestCoreTime = s.Time
			report.coreLimitMHz = s.CoreLimitMHz
		}
		addGPU(&report, s)
		addAppearances(summaries, s.TopProcesses)
	}

	report.topProcesses = rankProcesses(summaries, reportProcessCount)
	return report, nil
}

func addGPU(report *incidentReport, s sample) {
	if s.GPUState == "" {
		return
	}
	report.gpuSamples++
	if s.GPUState == "asleep" {
		report.gpuAsleepSamples++
	}
	if s.GPUPercent > report.peakGPUPercent || report.peakGPUTime.IsZero() {
		report.peakGPUPercent = s.GPUPercent
		report.peakGPUTime = s.Time
	}
	report.maxGPUTempCelsius = max(report.maxGPUTempCelsius, s.GPUTempCelsius)
	report.maxGPUPowerWatts = max(report.maxGPUPowerWatts, s.GPUPowerWatts)
}

func addAppearances(summaries map[int]*processSummary, usages []processUsage) {
	for _, usage := range usages {
		// Same PID but another name: a new program got this PID.
		summary, found := summaries[usage.PID]
		if !found || summary.name != usage.Name {
			summary = &processSummary{pid: usage.PID, name: usage.Name}
			summaries[usage.PID] = summary
		}
		summary.label = usage.label()
		summary.appearances++
		summary.totalCPU += usage.CPUPercent
		if usage.CPUPercent > summary.maxCPU {
			summary.maxCPU = usage.CPUPercent
		}
		if usage.CoreMaxMHz > 0 {
			summary.coreMHzSum += usage.CoreMHz
			summary.coreMHzSeen++
			summary.coreMaxMHz = max(summary.coreMaxMHz, usage.CoreMaxMHz)
		}
	}
}

// rankProcesses puts first the processes that used the most CPU in total.
func rankProcesses(summaries map[int]*processSummary, limit int) []processSummary {
	var ranked []processSummary
	for _, summary := range summaries {
		ranked = append(ranked, *summary)
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].totalCPU > ranked[j].totalCPU
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}

func printReport(w io.Writer, r incidentReport) {
	const timeLayout = "2006-01-02 15:04:05"
	fmt.Fprintf(w, "Period:             %s -> %s (%s, %d samples)\n",
		r.start.Format(timeLayout), r.end.Format(timeLayout), r.end.Sub(r.start).Round(time.Second), r.sampleCount)
	fmt.Fprintf(w, "CPU peak:           %.1f %% at %s\n", r.peakCPU, r.peakCPUTime.Format("15:04:05"))
	fmt.Fprintf(w, "Load peak:          %.2f at %s\n", r.peakLoad, r.peakLoadTime.Format("15:04:05"))
	fmt.Fprintf(w, "Lowest free memory: %s at %s\n", formatKB(r.minMemKB), r.minMemTime.Format("15:04:05"))
	if r.maxSwapKB > 0 {
		fmt.Fprintf(w, "Most swap used:     %s at %s\n", formatKB(r.maxSwapKB), r.maxSwapTime.Format("15:04:05"))
	} else {
		fmt.Fprintln(w, "Most swap used:     none")
	}
	if r.peakTemp > 0 {
		fmt.Fprintf(w, "Hottest CPU:        %.0f °C at %s\n", r.peakTemp, r.peakTempTime.Format("15:04:05"))
	} else {
		fmt.Fprintln(w, "Hottest CPU:        no sensor")
	}
	if r.coreLimitMHz > 0 {
		fmt.Fprintf(w, "Lowest CPU speed:   %d MHz (max %d) at %s\n",
			r.minFastestCoreMHz, r.coreLimitMHz, r.minFastestCoreTime.Format("15:04:05"))
	}
	if r.gpuSamples > 0 {
		fmt.Fprintf(w, "GPU peak:           %.0f %% at %s, up to %.0f °C and %.0f W (asleep %d/%d)\n",
			r.peakGPUPercent, r.peakGPUTime.Format("15:04:05"), r.maxGPUTempCelsius, r.maxGPUPowerWatts,
			r.gpuAsleepSamples, r.gpuSamples)
	}

	fmt.Fprintf(w, "\nBusiest processes (SEEN = times in the top %d):\n", topProcessCount)
	fmt.Fprintf(w, "  %7s  %6s  %8s  %14s  %s\n", "PID", "SEEN", "MAX CPU", "AVG SPEED", "COMMAND")
	for _, p := range r.topProcesses {
		seen := fmt.Sprintf("%d/%d", p.appearances, r.sampleCount)
		core := ""
		if p.coreMaxMHz > 0 {
			core = fmt.Sprintf("%d/%d MHz", p.averageCoreMHz(), p.coreMaxMHz)
		}
		fmt.Fprintf(w, "  %7d  %6s  %6.1f %%  %14s  %s\n", p.pid, seen, p.maxCPU, core, p.label)
	}

	fmt.Fprintln(w, "\n100 % = one full core. AVG SPEED = average speed of the core it ran on.")
	fmt.Fprintln(w, "This shows what was running, not proof of what caused the slowdown.")
}
