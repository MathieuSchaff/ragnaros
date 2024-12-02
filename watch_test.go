package main

import "testing"

func TestParseWatchFlagsDefaults(t *testing.T) {
	config, err := parseWatchFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	if config.threshold != 90 || config.coreThreshold != 95 || config.tempThreshold != 90 || config.spikeSamples != 5 || config.incidentDir != "incidents" {
		t.Errorf("got %+v", config)
	}
	if config.historySize() != 120 {
		t.Errorf("historySize = %d, want 120 (two minutes at 1s)", config.historySize())
	}
}

func TestHistorySizeFollowsInterval(t *testing.T) {
	config, err := parseWatchFlags([]string{"-interval", "500ms"})
	if err != nil {
		t.Fatal(err)
	}
	if config.historySize() != 240 {
		t.Errorf("historySize = %d, want 240", config.historySize())
	}
}

func TestParseWatchFlagsRejectsBadValues(t *testing.T) {
	for _, args := range [][]string{
		{"-threshold", "100"},
		{"-samples", "0"},
		{"-samples", "500"},
		{"-interval", "10ms"},
		{"-core-threshold", "-1"},
		{"-temp-threshold", "-5"},
		{"-interval", "30s", "-samples", "5"}, // only 4 samples fit in two minutes
	} {
		if _, err := parseWatchFlags(args); err == nil {
			t.Errorf("%v: expected an error", args)
		}
	}
}

func TestSpikeReason(t *testing.T) {
	config := watchConfig{threshold: 90, coreThreshold: 95, tempThreshold: 90}
	game := []processUsage{{Name: "game", CPUPercent: 99}}
	tests := []struct {
		name   string
		sample sample
		want   string
	}{
		{"calm", sample{CPUPercent: 20}, ""},
		{"global CPU", sample{CPUPercent: 95}, "CPU at 95 %"},
		{"one saturated core", sample{CPUPercent: 6, TopProcesses: game}, "game uses 99 % of a core"},
		{"hot CPU", sample{CPUPercent: 30, TempCelsius: 91}, "CPU at 91 °C"},
		{"warm but fine", sample{CPUPercent: 30, TempCelsius: 90}, ""},
	}
	for _, test := range tests {
		if got := spikeReason(test.sample, config); got != test.want {
			t.Errorf("%s: got %q, want %q", test.name, got, test.want)
		}
	}

	config.coreThreshold = 0
	if got := spikeReason(sample{CPUPercent: 6, TopProcesses: game}, config); got != "" {
		t.Errorf("disabled core trigger: got %q", got)
	}
}
