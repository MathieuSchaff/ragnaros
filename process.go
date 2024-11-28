package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// processStat is one reading of /proc/<pid>/stat.
type processStat struct {
	pid       int
	name      string
	startTime uint64 // when it started, to spot a reused PID
	ticks     uint64 // CPU time used since it started
	core      int    // core the process last ran on
}

// processKey tells processes apart even when Linux reuses a PID:
// the new process has another start time.
type processKey struct {
	pid       int
	startTime uint64
}

// processUsage is the CPU usage of one process between two samples.
// 100 % = one full core, so it can go above 100.
type processUsage struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	Command    string  `json:"command,omitempty"`
	CPUPercent float64 `json:"cpu_percent"`
	CoreMHz    int     `json:"core_mhz,omitempty"`     // frequency of the core it last ran on
	CoreMaxMHz int     `json:"core_max_mhz,omitempty"` // highest frequency of that core
	core       int     // not saved: only used to find CoreMHz
}

func readProcesses() ([]processStat, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var stats []processStat
	for _, entry := range entries {
		// Only numeric directories are processes.
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if err != nil {
			// The process ended while we were reading.
			continue
		}
		stat, err := parseProcessStat(string(data))
		if err != nil {
			continue
		}
		stats = append(stats, stat)
	}
	return stats, nil
}

// Format: "pid (name) state ppid ...". The name may contain spaces and ')',
// so it ends at the last ')' of the line.
func parseProcessStat(content string) (processStat, error) {
	nameStart := strings.Index(content, "(")
	nameEnd := strings.LastIndex(content, ")")
	if nameStart < 0 || nameEnd < nameStart {
		return processStat{}, fmt.Errorf("bad stat line")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(content[:nameStart]))
	if err != nil {
		return processStat{}, err
	}

	// fields[0] is field 3 in "man proc", so field N is fields[N-3].
	fields := strings.Fields(content[nameEnd+1:])
	if len(fields) < 37 {
		return processStat{}, fmt.Errorf("stat line too short")
	}
	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return processStat{}, err
	}
	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return processStat{}, err
	}
	startTime, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return processStat{}, err
	}
	core, err := strconv.Atoi(fields[36]) // field 39, "processor"
	if err != nil {
		return processStat{}, err
	}

	return processStat{
		pid:       pid,
		name:      content[nameStart+1 : nameEnd],
		startTime: startTime,
		ticks:     utime + stime,
		core:      core,
	}, nil
}

const maxCommandLength = 60

// readCommand returns the full command, like "python3 backup.py".
// It returns "" for kernel threads and for processes that ended.
func readCommand(pid int) string {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return ""
	}
	return formatCommand(string(data))
}

// Arguments in /proc/<pid>/cmdline are separated by NUL bytes.
func formatCommand(cmdline string) string {
	command := strings.TrimSpace(strings.ReplaceAll(cmdline, "\x00", " "))
	runes := []rune(command) // cut on letters, not bytes, so accents stay intact
	if len(runes) > maxCommandLength {
		return string(runes[:maxCommandLength-3]) + "..."
	}
	return command
}

// label is what we print: the command line when known, else the short name.
func (p processUsage) label() string {
	if p.Command != "" {
		return p.Command
	}
	return p.Name
}

func ticksByProcess(stats []processStat) map[processKey]uint64 {
	ticks := make(map[processKey]uint64, len(stats))
	for _, stat := range stats {
		ticks[processKey{stat.pid, stat.startTime}] = stat.ticks
	}
	return ticks
}

// topProcesses returns the processes that used the most CPU since the previous
// sample. elapsedTicks is the time that passed, in ticks.
func topProcesses(previous map[processKey]uint64, current []processStat, elapsedTicks float64, limit int) []processUsage {
	if elapsedTicks <= 0 {
		return nil
	}

	var usages []processUsage
	for _, stat := range current {
		// A process not seen last time is new: count all its CPU time.
		before := previous[processKey{stat.pid, stat.startTime}]
		if stat.ticks <= before {
			continue
		}
		percent := float64(stat.ticks-before) / elapsedTicks * 100
		usages = append(usages, processUsage{PID: stat.pid, Name: stat.name, CPUPercent: percent, core: stat.core})
	}

	sort.Slice(usages, func(i, j int) bool {
		return usages[i].CPUPercent > usages[j].CPUPercent
	})
	if len(usages) > limit {
		usages = usages[:limit]
	}
	return usages
}
