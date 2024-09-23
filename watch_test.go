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
}

func TestParseWatchFlagsRejectsBadValues(t *testing.T) {
	for _, args := range [][]string{{"-threshold", "100"}, {"-samples", "0"}, {"-samples", "500"}} {
		if _, err := parseWatchFlags(args); err == nil {
			t.Errorf("%v: expected an error", args)
		}
	}
}
