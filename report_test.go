package main

import (
	"strings"
	"testing"
	"time"
)

func TestBuildReport(t *testing.T) {
	start := time.Date(2024, 9, 6, 17, 0, 0, 0, time.UTC)
	samples := []sample{
		{Time: start, CPUPercent: 40, MemAvailableKB: 5000, TopProcesses: []processUsage{
			{PID: 10, Name: "backup", CPUPercent: 80},
			{PID: 20, Name: "browser", CPUPercent: 30},
		}},
		{Time: start.Add(time.Second), CPUPercent: 97, MemAvailableKB: 2000, TempCelsius: 81, TopProcesses: []processUsage{
			{PID: 10, Name: "backup", CPUPercent: 350},
		}},
		{Time: start.Add(2 * time.Second), CPUPercent: 92, MemAvailableKB: 3000, SwapUsedKB: 700, TopProcesses: []processUsage{
			{PID: 10, Name: "backup", CPUPercent: 300},
			{PID: 20, Name: "browser", CPUPercent: 400},
		}},
	}

	report, err := buildReport(samples)
	if err != nil {
		t.Fatal(err)
	}
	if report.peakCPU != 97 || !report.peakCPUTime.Equal(start.Add(time.Second)) {
		t.Errorf("peak = %v at %v", report.peakCPU, report.peakCPUTime)
	}
	if report.peakTemp != 81 {
		t.Errorf("peakTemp = %v, want 81", report.peakTemp)
	}
	if report.maxSwapKB != 700 || !report.maxSwapTime.Equal(start.Add(2*time.Second)) {
		t.Errorf("maxSwap = %d at %v", report.maxSwapKB, report.maxSwapTime)
	}
	if report.minMemKB != 2000 {
		t.Errorf("minMemKB = %d, want 2000", report.minMemKB)
	}
	if report.end.Sub(report.start) != 2*time.Second || report.sampleCount != 3 {
		t.Errorf("period %v, %d samples", report.end.Sub(report.start), report.sampleCount)
	}

	if len(report.topProcesses) != 2 {
		t.Fatalf("got %d processes, want 2", len(report.topProcesses))
	}
	// backup used 730 % in total, browser 430 %.
	first, second := report.topProcesses[0], report.topProcesses[1]
	if first.name != "backup" || first.appearances != 3 || first.maxCPU != 350 {
		t.Errorf("first = %+v", first)
	}
	if second.name != "browser" || second.appearances != 2 || second.maxCPU != 400 {
		t.Errorf("second = %+v", second)
	}
}

func TestBuildReportRanksHeavyProcessFirst(t *testing.T) {
	samples := []sample{
		{TopProcesses: []processUsage{{PID: 1, Name: "light", CPUPercent: 10}}},
		{TopProcesses: []processUsage{{PID: 1, Name: "light", CPUPercent: 10}, {PID: 2, Name: "heavy", CPUPercent: 900}}},
	}
	report, _ := buildReport(samples)
	if report.topProcesses[0].name != "heavy" {
		t.Errorf("first = %+v, want heavy despite fewer appearances", report.topProcesses[0])
	}
}

func TestBuildReportEmpty(t *testing.T) {
	if _, err := buildReport(nil); err == nil {
		t.Error("expected an error for an empty incident")
	}
}

func TestPrintReportMentionsObservations(t *testing.T) {
	report, _ := buildReport([]sample{{Time: time.Now(), CPUPercent: 95}})
	var output strings.Builder
	printReport(&output, report)
	if !strings.Contains(output.String(), "not a proof") {
		t.Errorf("report should warn it shows observations only:\n%s", output.String())
	}
}
