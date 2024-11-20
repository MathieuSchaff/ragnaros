package main

import (
	"strings"
	"testing"
)

func TestParseProcessStat(t *testing.T) {
	// Name with a space and a ')' to check the parsing of the name.
	line := "4242 (my (app) x) S 1 4242 4242 0 -1 4194560 100 0 0 0 30 12 0 0 20 0 1 0 98765 1000 50 0" +
		" 0 0 0 0 0 0 0 0 0 0 0 0 0 6 0 0 0 0 0 0 0 0 0 0 0 0 0\n" // field 39 (core) = 6
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
	if stat.core != 6 {
		t.Errorf("core = %d, want 6", stat.core)
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
		{pid: 4, name: "fresh", startTime: 40, ticks: 300}, // started since last reading
	}

	got := topProcesses(previous, current, 100, 8, 5)

	// The reused PID 3 is counted from zero, not from the old process's 100 ticks.
	want := []processUsage{
		{PID: 3, Name: "new", CPUPercent: 500},
		{PID: 4, Name: "fresh", CPUPercent: 300},
		{PID: 1, Name: "busy", CPUPercent: 150},
		{PID: 2, Name: "calm", CPUPercent: 10},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rank %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestTopProcessesLimit(t *testing.T) {
	previous := map[processKey]uint64{}
	var current []processStat
	for pid := 1; pid <= 8; pid++ {
		previous[processKey{pid, 0}] = 0
		current = append(current, processStat{pid: pid, ticks: uint64(pid)})
	}
	got := topProcesses(previous, current, 100, 8, 5)
	if len(got) != 5 || got[0].PID != 8 {
		t.Errorf("got %+v, want 5 processes starting with PID 8", got)
	}
}

func TestFormatCommand(t *testing.T) {
	if got := formatCommand("python3\x00backup.py\x00--all\x00"); got != "python3 backup.py --all" {
		t.Errorf("got %q", got)
	}
	long := formatCommand("/usr/bin/" + strings.Repeat("é", 100))
	if len([]rune(long)) != maxCommandLength || !strings.HasSuffix(long, "...") {
		t.Errorf("long command not cut to %d characters: %q", maxCommandLength, long)
	}
}

func TestLabelFallsBackToName(t *testing.T) {
	if got := (processUsage{Name: "kworker/0:1"}).label(); got != "kworker/0:1" {
		t.Errorf("got %q", got)
	}
}

func TestTopProcessesSkipsImpossibleUsage(t *testing.T) {
	// Missed by the previous reading: its 2000 lifetime ticks would mean 20 cores out of 8.
	current := []processStat{{pid: 7, name: "old", startTime: 1, ticks: 2000}}
	if got := topProcesses(map[processKey]uint64{}, current, 100, 8, 5); len(got) != 0 {
		t.Errorf("got %+v, want nothing", got)
	}
}
