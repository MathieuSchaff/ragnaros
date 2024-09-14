package main

import "testing"

func TestHistoryKeepsLatestSamples(t *testing.T) {
	h := newHistory(3)
	for i := 1; i <= 5; i++ {
		h.add(sample{CPUPercent: float64(i)})
	}

	got := h.all()
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, want := range []float64{3, 4, 5} {
		if got[i].CPUPercent != want {
			t.Errorf("sample %d = %v, want %v", i, got[i].CPUPercent, want)
		}
	}
}
