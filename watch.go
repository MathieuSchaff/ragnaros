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

const historyDuration = 2 * time.Minute

type watchConfig struct {
	interval      time.Duration
	threshold     float64
	coreThreshold float64 // 0 disables the single process trigger
	spikeSamples  int
	incidentDir   string
}

// historySize is the number of samples that cover historyDuration.
func (config watchConfig) historySize() int {
	return int(historyDuration / config.interval)
}

func parseWatchFlags(args []string) (watchConfig, error) {
	var config watchConfig
	flags := flag.NewFlagSet("watch", flag.ContinueOnError)
	flags.DurationVar(&config.interval, "interval", time.Second, "time between two samples, for example 500ms or 2s")
	flags.Float64Var(&config.threshold, "threshold", 90, "global CPU percent above which a sample counts as high")
	flags.Float64Var(&config.coreThreshold, "core-threshold", 95, "process CPU percent (100 = one core) that counts as high, 0 to disable")
	flags.IntVar(&config.spikeSamples, "samples", 5, "consecutive high samples that make an incident")
	flags.StringVar(&config.incidentDir, "dir", "incidents", "folder where incident files are written")
	if err := flags.Parse(args); err != nil {
		return watchConfig{}, err
	}

	if config.interval < 100*time.Millisecond || config.interval > time.Minute {
		return watchConfig{}, fmt.Errorf("-interval must be between 100ms and 1m")
	}
	if config.threshold < 0 || config.threshold >= 100 {
		return watchConfig{}, fmt.Errorf("-threshold must be between 0 and 100 (100 excluded)")
	}
	if config.coreThreshold < 0 {
		return watchConfig{}, fmt.Errorf("-core-threshold must be 0 or more")
	}
	if config.spikeSamples < 1 || config.spikeSamples > config.historySize() {
		return watchConfig{}, fmt.Errorf("-samples must be between 1 and %d for this interval", config.historySize())
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
	recent := newHistory(config.historySize())
	spikes := newDetector(config.spikeSamples)

	// Ctrl+C (SIGINT) or SIGTERM is delivered on this channel instead of killing the program.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(config.interval)
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
			reason := spikeReason(s, config)
			path, err := recordSample(recent, spikes, s, reason != "", config.incidentDir)
			if path != "" {
				fmt.Println("Spike:", reason)
			}
			printSaveResult(path, err)
		}
	}
}

// spikeReason tells why a sample is high, or returns "" when it is not.
// A single-threaded program, like many games, can saturate one core and
// slow down while the global CPU stays low: hence the per-process check.
func spikeReason(s sample, config watchConfig) string {
	if s.CPUPercent > config.threshold {
		return fmt.Sprintf("global CPU at %.0f %%", s.CPUPercent)
	}
	// TopProcesses is sorted, so the first one is the busiest.
	if config.coreThreshold > 0 && len(s.TopProcesses) > 0 && s.TopProcesses[0].CPUPercent > config.coreThreshold {
		busiest := s.TopProcesses[0]
		return fmt.Sprintf("%s at %.0f %% of a core", busiest.Name, busiest.CPUPercent)
	}
	return ""
}

// printSaveResult only reports: a failed save must not stop the monitoring.
func printSaveResult(path string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ragnaros: cannot save incident:", err)
	} else if path != "" {
		fmt.Println("Incident saved:", path)
	}
}

// formatCoreFrequency keeps a fixed width so the commands stay aligned.
func formatCoreFrequency(p processUsage) string {
	if p.CoreMaxMHz == 0 {
		return fmt.Sprintf("%14s", "")
	}
	return fmt.Sprintf("%5d/%4d MHz", p.CoreMHz, p.CoreMaxMHz)
}

func printSample(w io.Writer, s sample) {
	fmt.Fprintf(w, "\n%s  CPU %5.1f %%  load %.2f  available memory %s  swap used %s",
		s.Time.Format("15:04:05"), s.CPUPercent, s.LoadAverage, formatKB(s.MemAvailableKB), formatKB(s.SwapUsedKB))
	if s.TempCelsius > 0 {
		fmt.Fprintf(w, "  temperature %.0f °C", s.TempCelsius)
	}
	if s.CoreLimitMHz > 0 {
		fmt.Fprintf(w, "  fastest core %d/%d MHz", s.FastestCoreMHz, s.CoreLimitMHz)
	}
	fmt.Fprintln(w)
	for _, p := range s.TopProcesses {
		fmt.Fprintf(w, "  %7d  %6.1f %%  %s  %s\n", p.PID, p.CPUPercent, formatCoreFrequency(p), p.label())
	}
}
