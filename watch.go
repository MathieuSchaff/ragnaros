package main

import (
	"errors"
	"flag"
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

type watchConfig struct {
	threshold    float64
	spikeSamples int
	incidentDir  string
}

func parseWatchFlags(args []string) (watchConfig, error) {
	var config watchConfig
	flags := flag.NewFlagSet("watch", flag.ContinueOnError)
	flags.Float64Var(&config.threshold, "threshold", 90, "global CPU percent above which a sample counts as high")
	flags.IntVar(&config.spikeSamples, "samples", 5, "consecutive high samples that make an incident")
	flags.StringVar(&config.incidentDir, "dir", "incidents", "folder where incident files are written")
	if err := flags.Parse(args); err != nil {
		return watchConfig{}, err
	}

	if config.threshold < 0 || config.threshold >= 100 {
		return watchConfig{}, fmt.Errorf("-threshold must be between 0 and 100 (100 excluded)")
	}
	if config.spikeSamples < 1 || config.spikeSamples > historySize {
		return watchConfig{}, fmt.Errorf("-samples must be between 1 and %d", historySize)
	}
	return config, nil
}

func runWatch(args []string) error {
	config, err := parseWatchFlags(args)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}

	collector, err := newCollector()
	if err != nil {
		return err
	}
	recent := newHistory(historySize)
	spikes := newDetector(config.threshold, config.spikeSamples)

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
			// The user may stop because the machine is slow: keep a spike
			// that started but is not long enough yet to be saved.
			if spikes.pendingSpike() {
				printSaveResult(saveIncident(config.incidentDir, recent.all(), time.Now()))
			}
			fmt.Println("\nStopped.")
			return nil
		case <-ticker.C:
			s, err := collector.collect()
			if err != nil {
				return err
			}
			printSample(os.Stdout, s)
			printSaveResult(recordSample(recent, spikes, s, config.incidentDir))
		}
	}
}

// printSaveResult only reports: a failed save must not stop the monitoring.
func printSaveResult(path string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ragnaros: cannot save incident:", err)
	} else if path != "" {
		fmt.Println("Incident saved:", path)
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
		fmt.Fprintf(w, "  %7d  %6.1f %%  %s\n", p.PID, p.CPUPercent, p.label())
	}
}
