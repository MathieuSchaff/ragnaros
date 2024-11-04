package main

import "testing"

func TestParseLoadAverage(t *testing.T) {
	got, err := parseLoadAverage("3.25 2.10 1.05 4/1500 98765\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != 3.25 {
		t.Errorf("got %v, want 3.25", got)
	}
}

func TestParseLoadAverageInvalid(t *testing.T) {
	for _, content := range []string{"", "abc 1 2"} {
		if _, err := parseLoadAverage(content); err == nil {
			t.Errorf("%q: expected an error", content)
		}
	}
}
