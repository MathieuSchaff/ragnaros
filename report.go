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
	appearances int
	maxCPU      float64
}

type incidentReport struct {
	start, end   time.Time
	sampleCount  int
	peakCPU      float64
	peakCPUTime  time.Time
	minMemKB     uint64
	minMemTime   time.Time
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
		summary.appearances++
		if usage.CPUPercent > summary.maxCPU {
			summary.maxCPU = usage.CPUPercent
		}
	}
}

// rankProcesses sorts by number of appearances, then by peak usage.
func rankProcesses(summaries map[int]*processSummary, limit int) []processSummary {
	var ranked []processSummary
	for _, summary := range summaries {
		ranked = append(ranked, *summary)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].appearances != ranked[j].appearances {
			return ranked[i].appearances > ranked[j].appearances
		}
		return ranked[i].maxCPU > ranked[j].maxCPU
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}

func printReport(w io.Writer, r incidentReport) {
	const timeLayout = "2006-01-02 15:04:05"
	fmt.Fprintf(w, "Period:            %s -> %s (%s, %d samples)\n",
		r.start.Format(timeLayout), r.end.Format(timeLayout), r.end.Sub(r.start).Round(time.Second), r.sampleCount)
	fmt.Fprintf(w, "CPU peak:          %.1f %% at %s\n", r.peakCPU, r.peakCPUTime.Format("15:04:05"))
	fmt.Fprintf(w, "Lowest available memory: %s at %s\n", formatKB(r.minMemKB), r.minMemTime.Format("15:04:05"))

	fmt.Fprintf(w, "\nMost active processes (times seen in the top %d of each sample):\n", topProcessCount)
	fmt.Fprintf(w, "  %7s  %-16s %9s  %8s\n", "PID", "NAME", "SEEN", "MAX CPU")
	for _, p := range r.topProcesses {
		seen := fmt.Sprintf("%d/%d", p.appearances, r.sampleCount)
		fmt.Fprintf(w, "  %7d  %-16s %9s  %6.1f %%\n", p.pid, p.name, seen, p.maxCPU)
	}

	fmt.Fprintln(w, "\nProcess CPU: 100 % means one full core.")
	fmt.Fprintln(w, "These are observations of what was running before the spike, not a proof of its cause.")
}
