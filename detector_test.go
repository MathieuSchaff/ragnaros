package main

import "testing"

// countIncidents feeds simulated CPU values and returns the indexes that fired.
func countIncidents(d *detector, values []float64) []int {
	var fired []int
	for i, value := range values {
		if d.update(value) {
			fired = append(fired, i)
		}
	}
	return fired
}

func TestDetectorNeedsFiveConsecutiveSamples(t *testing.T) {
	d := newDetector(90, 5)
	fired := countIncidents(d, []float64{95, 95, 95, 95, 50, 95, 95, 95, 95})
	if len(fired) != 0 {
		t.Errorf("fired at %v, want no incident (spike interrupted)", fired)
	}
}

func TestDetectorFiresOncePerSpike(t *testing.T) {
	d := newDetector(90, 5)
	values := []float64{91, 92, 93, 94, 95, 99, 99, 99, 99, 99, 99}
	fired := countIncidents(d, values)
	if len(fired) != 1 || fired[0] != 4 {
		t.Errorf("fired at %v, want exactly once at index 4", fired)
	}
}

func TestDetectorRearmsAfterDrop(t *testing.T) {
	d := newDetector(90, 5)
	values := []float64{95, 95, 95, 95, 95, 95, 40, 95, 95, 95, 95, 95}
	fired := countIncidents(d, values)
	if len(fired) != 2 || fired[0] != 4 || fired[1] != 11 {
		t.Errorf("fired at %v, want [4 11]", fired)
	}
}

func TestDetectorThresholdIsStrict(t *testing.T) {
	d := newDetector(90, 5)
	fired := countIncidents(d, []float64{90, 90, 90, 90, 90, 90})
	if len(fired) != 0 {
		t.Errorf("fired at %v, want none: 90 is not above 90", fired)
	}
}
