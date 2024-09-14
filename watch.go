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
)

func runWatch() error {
	collector, err := newCollector()
	if err != nil {
		return err
	}
	recent := newHistory(historySize)

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
			recent.add(s)
			printSample(os.Stdout, s)
		}
	}
}

func printSample(w io.Writer, s sample) {
	fmt.Fprintf(w, "\n%s  CPU %5.1f %%  available memory %s\n",
		s.Time.Format("15:04:05"), s.CPUPercent, formatKB(s.MemAvailableKB))
	for _, p := range s.TopProcesses {
		fmt.Fprintf(w, "  %7d  %-16s %6.1f %%\n", p.PID, p.Name, p.CPUPercent)
	}
}
