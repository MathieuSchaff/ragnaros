package main

// detector reports a spike once required consecutive samples are high.
// It fires once per spike and rearms on the first sample that is not high.
// Deciding what "high" means is left to the caller.
type detector struct {
	required    int
	consecutive int
	triggered   bool
}

func newDetector(required int) *detector {
	return &detector{required: required}
}

// update returns true only on the sample that starts an incident.
func (d *detector) update(high bool) bool {
	if !high {
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

// pendingSpike is true when samples are high but no incident was saved yet.
func (d *detector) pendingSpike() bool {
	return d.consecutive > 0 && !d.triggered
}
