package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

const reportProcessCount = 5

// processSummary tells how often a process appeared in the per-second top list.
type processSummary struct {
	pid         int
	name        string
	label       string // command line if known, else name
	appearances int
	totalCPU    float64 // sum of the per-second percentages
	maxCPU      float64
}

type incidentReport struct {
	start, end   time.Time
	sampleCount  int
	peakCPU      float64
	peakCPUTime  time.Time
	minMemKB     uint64
	minMemTime   time.Time
	peakTemp     float64 // 0 when the incident has no temperature
	peakTempTime time.Time
	topProcesses []processSummary
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
		start:       first.Time,
		end:         first.Time,
		sampleCount: len(samples),
		peakCPU:     first.CPUPercent,
		peakCPUTime: first.Time,
		minMemKB:    first.MemAvailableKB,
		minMemTime:  first.Time,
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
		if s.MemAvailableKB < report.minMemKB {
			report.minMemKB = s.MemAvailableKB
			report.minMemTime = s.Time
		}
		if s.TempCelsius > report.peakTemp {
			report.peakTemp = s.TempCelsius
			report.peakTempTime = s.Time
		}
		addAppearances(summaries, s.TopProcesses)
	}

	report.topProcesses = rankProcesses(summaries, reportProcessCount)
	return report, nil
}

func addAppearances(summaries map[int]*processSummary, usages []processUsage) {
	for _, usage := range usages {
		// The incident file has no start time, so a PID reused by another
		// program is told apart by its name only.
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
	}
}

// rankProcesses sorts by CPU used over the whole period, so a short but heavy
// process ranks above a light one that is always present.
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
	fmt.Fprintf(w, "Period:           %s -> %s (%s, %d samples)\n",
		r.start.Format(timeLayout), r.end.Format(timeLayout), r.end.Sub(r.start).Round(time.Second), r.sampleCount)
	fmt.Fprintf(w, "CPU peak:         %.1f %% at %s\n", r.peakCPU, r.peakCPUTime.Format("15:04:05"))
	fmt.Fprintf(w, "Lowest memory:    %s available at %s\n", formatKB(r.minMemKB), r.minMemTime.Format("15:04:05"))
	if r.peakTemp > 0 {
		fmt.Fprintf(w, "Max temperature:  %.0f °C at %s\n", r.peakTemp, r.peakTempTime.Format("15:04:05"))
	} else {
		fmt.Fprintln(w, "Max temperature:  no sensor data")
	}

	fmt.Fprintf(w, "\nMost active processes (ranked by CPU used, SEEN = times in the top %d):\n", topProcessCount)
	fmt.Fprintf(w, "  %7s  %6s  %8s  %s\n", "PID", "SEEN", "MAX CPU", "COMMAND")
	for _, p := range r.topProcesses {
		seen := fmt.Sprintf("%d/%d", p.appearances, r.sampleCount)
		fmt.Fprintf(w, "  %7d  %6s  %6.1f %%  %s\n", p.pid, seen, p.maxCPU, p.label)
	}

	fmt.Fprintln(w, "\nProcess CPU: 100 % means one full core.")
	fmt.Fprintln(w, "These are observations of what was running before the spike, not a proof of its cause.")
}
