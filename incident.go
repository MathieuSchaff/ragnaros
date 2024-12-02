package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// saveIncident writes the samples as JSONL: one JSON object per line.
func saveIncident(dir string, samples []sample, at time.Time) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "incident-"+at.Format("20060102-150405")+".jsonl")

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer) // Encode adds a newline after each object
	for _, s := range samples {
		if err := encoder.Encode(s); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func readIncident(path string) ([]sample, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var samples []sample
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var s sample
		if err := decoder.Decode(&s); err != nil {
			return nil, fmt.Errorf("%s: sample %d: %w", path, len(samples)+1, err)
		}
		samples = append(samples, s)
	}
	return samples, nil
}

// recordSample adds a sample to the history and saves the history when a spike starts.
// It returns the incident file path, or "" when nothing was saved.
func recordSample(recent *history, spikes *detector, s sample, dir string) (string, error) {
	recent.add(s)
	if !spikes.update(s.Alert != "") {
		return "", nil
	}
	return saveIncident(dir, recent.all(), s.Time)
}
