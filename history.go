package main

// history keeps only the most recent samples, up to maxSize.
type history struct {
	samples []sample
	maxSize int
}

func newHistory(maxSize int) *history {
	return &history{maxSize: maxSize}
}

func (h *history) add(s sample) {
	h.samples = append(h.samples, s)
	if len(h.samples) > h.maxSize {
		// Drop the oldest sample. append later copies the remaining ones
		// into a new array, so memory stays bounded.
		h.samples = h.samples[1:]
	}
}

// all returns a copy, so the caller can keep it while the history changes.
func (h *history) all() []sample {
	result := make([]sample, len(h.samples))
	copy(result, h.samples)
	return result
}
