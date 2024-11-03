package main

import "testing"

func TestParseWatchFlagsDefaults(t *testing.T) {
	config, err := parseWatchFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	if config.threshold != 90 || config.spikeSamples != 5 || config.incidentDir != "incidents" {
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
		{"-interval", "30s", "-samples", "5"}, // only 4 samples fit in two minutes
	} {
		if _, err := parseWatchFlags(args); err == nil {
			t.Errorf("%v: expected an error", args)
		}
	}
}
