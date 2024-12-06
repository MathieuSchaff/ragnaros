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
	coreThreshold float64 // 0 = off
	tempThreshold float64 // 0 = off
	spikeSamples  int
	incidentDir   string
}

// historySize is how many samples fit in two minutes.
func (config watchConfig) historySize() int {
	return int(historyDuration / config.interval)
}

func parseWatchFlags(args []string) (watchConfig, error) {
	var config watchConfig
	flags := flag.NewFlagSet("watch", flag.ContinueOnError)
	flags.DurationVar(&config.interval, "interval", time.Second, "time between two samples, for example 500ms or 2s")
	flags.Float64Var(&config.threshold, "threshold", 90, "global CPU percent above which a sample counts as high")
	flags.Float64Var(&config.coreThreshold, "core-threshold", 95, "process CPU percent (100 = one core) that counts as high, 0 to disable")
	flags.Float64Var(&config.tempThreshold, "temp-threshold", 90, "temperature in °C that counts as high, 0 to disable")
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
	if config.tempThreshold < 0 {
		return watchConfig{}, fmt.Errorf("-temp-threshold must be 0 or more")
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

	// Send Ctrl+C to this channel instead of killing the program.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(config.interval)
	defer ticker.Stop()

	fmt.Println("Ragnaros is watching. Press Ctrl+C to stop.")
	for {
		// Wait for Ctrl+C or the next tick, whichever comes first.
		select {
		case <-stop:
			// If the machine is slow right now, save what we have before quitting.
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
			s.Alert = spikeReason(s, config)
			printSample(os.Stdout, s)
			path, err := recordSample(recent, spikes, s, config.incidentDir)
			if path != "" {
				fmt.Println("Alert:", s.Alert)
			}
			printSaveResult(path, err)
		}
	}
}

// spikeReason says why a sample counts as high, or returns "".
// A program that uses only one core can lag while the total CPU stays low.
func spikeReason(s sample, config watchConfig) string {
	if s.CPUPercent > config.threshold {
		return fmt.Sprintf("CPU at %.0f %%", s.CPUPercent)
	}
	// TopProcesses is sorted, so the first one is the busiest.
	if config.coreThreshold > 0 && len(s.TopProcesses) > 0 && s.TopProcesses[0].CPUPercent > config.coreThreshold {
		busiest := s.TopProcesses[0]
		return fmt.Sprintf("%s uses %.0f %% of a core", busiest.Name, busiest.CPUPercent)
	}
	// A hot CPU slows itself down.
	if config.tempThreshold > 0 && s.TempCelsius > config.tempThreshold {
		return fmt.Sprintf("CPU at %.0f °C", s.TempCelsius)
	}
	return ""
}

// printSaveResult prints the result. A failed save does not stop watch.
func printSaveResult(path string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ragnaros: cannot save incident:", err)
	} else if path != "" {
		fmt.Println("Incident saved:", path)
	}
}

// formatCoreFrequency keeps the same width so the commands line up.
func formatCoreFrequency(p processUsage) string {
	if p.CoreMaxMHz == 0 {
		return fmt.Sprintf("%14s", "")
	}
	return fmt.Sprintf("%5d/%4d MHz", p.CoreMHz, p.CoreMaxMHz)
}

func printSample(w io.Writer, s sample) {
	fmt.Fprintf(w, "\n%s  CPU %5.1f %%  load %.2f  free memory %s  swap %s",
		s.Time.Format("15:04:05"), s.CPUPercent, s.LoadAverage, formatKB(s.MemAvailableKB), formatKB(s.SwapUsedKB))
	if s.TempCelsius > 0 {
		fmt.Fprintf(w, "  %.0f °C", s.TempCelsius)
	}
	if s.CoreLimitMHz > 0 {
		fmt.Fprintf(w, "  speed %d/%d MHz", s.FastestCoreMHz, s.CoreLimitMHz)
	}
	if s.GPUState == "active" {
		fmt.Fprintf(w, "  GPU %.0f %% %.0f °C %.0f W %.0f/%.0f MiB",
			s.GPUPercent, s.GPUTempCelsius, s.GPUPowerWatts, s.GPUMemoryMiB, s.GPUMemoryMax)
	} else if s.GPUState != "" {
		fmt.Fprintf(w, "  GPU %s", s.GPUState)
	}
	fmt.Fprintln(w)
	for _, p := range s.TopProcesses {
		fmt.Fprintf(w, "  %7d  %6.1f %%  %s  %s\n", p.PID, p.CPUPercent, formatCoreFrequency(p), p.label())
	}
}
