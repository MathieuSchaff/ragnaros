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
	startTime uint64 // ticks since boot, used to detect PID reuse
	ticks     uint64 // user + system CPU time, cumulative
}

// processKey identifies a process even if its PID is later reused:
// a new process with the same PID has a different start time.
type processKey struct {
	pid       int
	startTime uint64
}

// processUsage is the CPU usage of one process between two samples.
// 100 % means one full core, so the value can exceed 100 on multi-core machines.
type processUsage struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	Command    string  `json:"command,omitempty"`
	CPUPercent float64 `json:"cpu_percent"`
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
			// The process exited between ReadDir and ReadFile.
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

	// fields[0] is field 3 of the man page (state), so field N is fields[N-3].
	fields := strings.Fields(content[nameEnd+1:])
	if len(fields) < 20 {
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

	return processStat{
		pid:       pid,
		name:      content[nameStart+1 : nameEnd],
		startTime: startTime,
		ticks:     utime + stime,
	}, nil
}

const maxCommandLength = 60

// readCommand returns the full command line, which tells apart processes that
// share a short name (for example several "python3"). It returns "" for kernel
// threads, which have no command line, and for processes that already exited.
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
	runes := []rune(command) // cut on characters, not bytes, to keep UTF-8 valid
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
// sample. elapsedTicks is the time elapsed for one core, in clock ticks.
func topProcesses(previous map[processKey]uint64, current []processStat, elapsedTicks float64, cores int, limit int) []processUsage {
	if elapsedTicks <= 0 {
		return nil
	}

	var usages []processUsage
	for _, stat := range current {
		// A process absent from the previous reading (new, or reused PID)
		// started since then, so all its CPU time belongs to this interval.
		before := previous[processKey{stat.pid, stat.startTime}]
		if stat.ticks <= before {
			continue
		}
		percent := float64(stat.ticks-before) / elapsedTicks * 100
		// More than all cores together is impossible: the process existed before
		// but its previous reading failed, so its whole lifetime was counted.
		if percent > float64(cores)*100 {
			continue
		}
		usages = append(usages, processUsage{PID: stat.pid, Name: stat.name, CPUPercent: percent})
	}

	sort.Slice(usages, func(i, j int) bool {
		return usages[i].CPUPercent > usages[j].CPUPercent
	})
	if len(usages) > limit {
		usages = usages[:limit]
	}
	return usages
}
