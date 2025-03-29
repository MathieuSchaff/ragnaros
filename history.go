package main

import "ragnaros/internal/system"

// history keeps only the most recent samples, up to maxSize.
type history struct {
	samples []system.Sample
	maxSize int
}

func newHistory(maxSize int) *history {
	return &history{maxSize: maxSize}
}

func (h *history) add(s system.Sample) {
	h.samples = append(h.samples, s)
	if len(h.samples) > h.maxSize {
		// Drop the oldest sample.
		h.samples = h.samples[1:]
	}
}

// all returns a copy, safe to keep while new samples arrive.
func (h *history) all() []system.Sample {
	result := make([]system.Sample, len(h.samples))
	copy(result, h.samples)
	return result
}
