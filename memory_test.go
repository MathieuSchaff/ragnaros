package main

import "testing"

func TestParseMemory(t *testing.T) {
	content := "MemTotal:       16000000 kB\nMemFree:         1000000 kB\nMemAvailable:    8000000 kB\n" +
		"SwapTotal:       4000000 kB\nSwapFree:        3000000 kB\n"
	got, err := parseMemory(content)
	if err != nil {
		t.Fatal(err)
	}
	if got.availableKB != 8000000 || got.swapUsedKB != 1000000 {
		t.Errorf("got %+v, want 8000000 available and 1000000 swap used", got)
	}
}

func TestParseMemoryWithoutSwap(t *testing.T) {
	got, err := parseMemory("MemAvailable: 500 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.swapUsedKB != 0 {
		t.Errorf("swapUsedKB = %d, want 0", got.swapUsedKB)
	}
}

func TestParseMemoryMissing(t *testing.T) {
	if _, err := parseMemory("MemTotal: 1 kB\n"); err == nil {
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
