package incident

import "ragnaros/internal/system"

// History keeps only the most recent samples, up to maxSize.
type History struct {
	samples []system.Sample
	maxSize int
}

func NewHistory(maxSize int) *History {
	return &History{maxSize: maxSize}
}

func (h *History) Add(s system.Sample) {
	h.samples = append(h.samples, s)
	if len(h.samples) > h.maxSize {
		// Drop the oldest sample.
		h.samples = h.samples[1:]
	}
}

// All returns a copy, safe to keep while new samples arrive.
func (h *History) All() []system.Sample {
	result := make([]system.Sample, len(h.samples))
	copy(result, h.samples)
	return result
}
