package main

// detector reports a CPU spike once the usage stays above threshold for
// required consecutive samples. It fires once per spike and rearms when
// the usage goes back to or below the threshold.
type detector struct {
	threshold   float64
	required    int
	consecutive int
	triggered   bool
}

func newDetector(threshold float64, required int) *detector {
	return &detector{threshold: threshold, required: required}
}

// update returns true only on the sample that starts an incident.
func (d *detector) update(cpuPercent float64) bool {
	if cpuPercent <= d.threshold {
		d.consecutive = 0
		d.triggered = false
		return false
	}

	d.consecutive++
	if d.consecutive >= d.required && !d.triggered {
		d.triggered = true
		return true
	}
	return false
}
