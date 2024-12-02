package main

import (
	"strings"
	"testing"
	"time"
)

func TestBuildReport(t *testing.T) {
	start := time.Date(2024, 9, 6, 17, 0, 0, 0, time.UTC)
	samples := []sample{
		{Time: start, CPUPercent: 40, LoadAverage: 12.5, MemAvailableKB: 5000, TopProcesses: []processUsage{
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
	if report.peakLoad != 12.5 || !report.peakLoadTime.Equal(start) {
		t.Errorf("peakLoad = %v at %v", report.peakLoad, report.peakLoadTime)
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
	if !strings.Contains(output.String(), "not proof") {
		t.Errorf("report should warn it shows observations only:\n%s", output.String())
	}
}

func TestBuildReportCoreFrequency(t *testing.T) {
	game := func(coreMHz int) processUsage {
		return processUsage{PID: 6, Name: "game", CPUPercent: 100, CoreMHz: coreMHz, CoreMaxMHz: 4700}
	}
	samples := []sample{
		{FastestCoreMHz: 4600, CoreLimitMHz: 4700, TopProcesses: []processUsage{game(4600)}},
		{FastestCoreMHz: 2697, CoreLimitMHz: 4700, TopProcesses: []processUsage{game(2697)}},
		{FastestCoreMHz: 3000, CoreLimitMHz: 4700, TopProcesses: []processUsage{game(960)}},
	}
	report, _ := buildReport(samples)
	if report.minFastestCoreMHz != 2697 || report.coreLimitMHz != 4700 {
		t.Errorf("slowest = %d/%d, want 2697/4700", report.minFastestCoreMHz, report.coreLimitMHz)
	}
	if got := report.topProcesses[0].averageCoreMHz(); got != 2752 {
		t.Errorf("average core = %d, want 2752", got)
	}
}

func TestBuildReportWithoutFrequency(t *testing.T) {
	report, _ := buildReport([]sample{{CPUPercent: 95}})
	if report.coreLimitMHz != 0 {
		t.Errorf("coreLimitMHz = %d, want 0 without cpufreq", report.coreLimitMHz)
	}
}

func TestBuildReportGPU(t *testing.T) {
	start := time.Date(2024, 9, 6, 17, 0, 0, 0, time.UTC)
	samples := []sample{
		{Time: start, GPUState: "asleep"},
		{Time: start.Add(time.Second), GPUState: "active", GPUPercent: 14, GPUTempCelsius: 83, GPUPowerWatts: 24},
		{Time: start.Add(2 * time.Second), GPUState: "active", GPUPercent: 9, GPUTempCelsius: 80, GPUPowerWatts: 30},
	}
	report, _ := buildReport(samples)
	if report.gpuSamples != 3 || report.gpuAsleepSamples != 1 {
		t.Errorf("gpu samples %d, asleep %d, want 3 and 1", report.gpuSamples, report.gpuAsleepSamples)
	}
	if report.peakGPUPercent != 14 || !report.peakGPUTime.Equal(start.Add(time.Second)) {
		t.Errorf("GPU peak %v at %v", report.peakGPUPercent, report.peakGPUTime)
	}
	if report.maxGPUTempCelsius != 83 || report.maxGPUPowerWatts != 30 {
		t.Errorf("max GPU %v °C %v W", report.maxGPUTempCelsius, report.maxGPUPowerWatts)
	}
}

func TestBuildReportKeepsLastAlert(t *testing.T) {
	samples := []sample{{Alert: "CPU at 91 °C"}, {}, {Alert: "game uses 99 % of a core"}, {}}
	report, _ := buildReport(samples)
	if report.alert != "game uses 99 % of a core" {
		t.Errorf("alert = %q, want the last one", report.alert)
	}
}
