package main

import "testing"

func TestParseMemAvailableKB(t *testing.T) {
	content := "MemTotal:       16000000 kB\nMemFree:         1000000 kB\nMemAvailable:    8000000 kB\n"
	got, err := parseMemAvailableKB(content)
	if err != nil {
		t.Fatal(err)
	}
	if got != 8000000 {
		t.Errorf("got %d, want 8000000", got)
	}
}

func TestParseMemAvailableKBMissing(t *testing.T) {
	if _, err := parseMemAvailableKB("MemTotal: 1 kB\n"); err == nil {
		t.Error("expected an error without MemAvailable")
	}
}

func TestFormatKB(t *testing.T) {
	if got := formatKB(512 * 1024); got != "512 MiB" {
		t.Errorf("got %q", got)
	}
	if got := formatKB(3 * 1024 * 1024); got != "3.0 GiB" {
		t.Errorf("got %q", got)
	}
}
