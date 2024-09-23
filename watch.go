package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	sampleInterval = time.Second
	historySize    = 120 // two minutes at one sample per second
	cpuThreshold   = 90.0
	spikeSamples   = 5
	incidentDir    = "incidents"
)

func runWatch() error {
	collector, err := newCollector()
	if err != nil {
		return err
	}
	recent := newHistory(historySize)
	spikes := newDetector(cpuThreshold, spikeSamples)

	// Ctrl+C (SIGINT) or SIGTERM is delivered on this channel instead of killing the program.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	fmt.Println("Ragnaros is watching. Press Ctrl+C to stop.")
	for {
		// Wait for whichever comes first: a stop signal or the next tick.
		select {
		case <-stop:
			fmt.Println("\nStopped.")
			return nil
		case <-ticker.C:
			s, err := collector.collect()
			if err != nil {
				return err
			}
			printSample(os.Stdout, s)
			path, err := recordSample(recent, spikes, s, incidentDir)
			if err != nil {
				// Keep watching: a failed save must not stop the monitoring.
				fmt.Fprintln(os.Stderr, "ragnaros: cannot save incident:", err)
			} else if path != "" {
				fmt.Println("Incident saved:", path)
			}
		}
	}
}

func printSample(w io.Writer, s sample) {
	fmt.Fprintf(w, "\n%s  CPU %5.1f %%  available memory %s",
		s.Time.Format("15:04:05"), s.CPUPercent, formatKB(s.MemAvailableKB))
	if s.TempCelsius > 0 {
		fmt.Fprintf(w, "  max temperature %.0f °C", s.TempCelsius)
	}
	fmt.Fprintln(w)
	for _, p := range s.TopProcesses {
		fmt.Fprintf(w, "  %7d  %-16s %6.1f %%\n", p.PID, p.Name, p.CPUPercent)
	}
}
