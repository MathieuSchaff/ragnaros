package main

import "testing"

func TestParseProcessStat(t *testing.T) {
	// Name with a space and a ')' to check the parsing of the name.
	line := "4242 (my (app) x) S 1 4242 4242 0 -1 4194560 100 0 0 0 30 12 0 0 20 0 1 0 98765 1000 50 0\n"
	stat, err := parseProcessStat(line)
	if err != nil {
		t.Fatal(err)
	}
	if stat.pid != 4242 || stat.name != "my (app) x" {
		t.Errorf("got pid %d name %q", stat.pid, stat.name)
	}
	if stat.ticks != 42 {
		t.Errorf("ticks = %d, want 42 (30 user + 12 system)", stat.ticks)
	}
	if stat.startTime != 98765 {
		t.Errorf("startTime = %d, want 98765", stat.startTime)
	}
}

func TestParseProcessStatInvalid(t *testing.T) {
	if _, err := parseProcessStat("12 (short) S 1"); err == nil {
		t.Error("expected an error for a truncated line")
	}
}

func TestTopProcesses(t *testing.T) {
	previous := map[processKey]uint64{
		{pid: 1, startTime: 10}: 100,
		{pid: 2, startTime: 20}: 100,
		{pid: 3, startTime: 30}: 100, // PID 3 is reused below
		{pid: 9, startTime: 90}: 100, // PID 9 has disappeared
	}
	current := []processStat{
		{pid: 1, name: "busy", startTime: 10, ticks: 250},  // 150 ticks
		{pid: 2, name: "calm", startTime: 20, ticks: 110},  // 10 ticks
		{pid: 3, name: "new", startTime: 31, ticks: 500},   // new process, same PID
		{pid: 4, name: "fresh", startTime: 40, ticks: 300}, // never seen
	}

	got := topProcesses(previous, current, 100, 5)

	if len(got) != 2 {
		t.Fatalf("got %d processes, want 2: %+v", len(got), got)
	}
	if got[0].Name != "busy" || got[0].CPUPercent != 150 {
		t.Errorf("first = %+v, want busy at 150 %% (1.5 cores)", got[0])
	}
	if got[1].Name != "calm" || got[1].CPUPercent != 10 {
		t.Errorf("second = %+v, want calm at 10 %%", got[1])
	}
}

func TestTopProcessesLimit(t *testing.T) {
	previous := map[processKey]uint64{}
	var current []processStat
	for pid := 1; pid <= 8; pid++ {
		previous[processKey{pid, 0}] = 0
		current = append(current, processStat{pid: pid, ticks: uint64(pid)})
	}
	got := topProcesses(previous, current, 100, 5)
	if len(got) != 5 || got[0].PID != 8 {
		t.Errorf("got %+v, want 5 processes starting with PID 8", got)
	}
}
