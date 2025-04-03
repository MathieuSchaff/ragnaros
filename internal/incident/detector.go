package incident

// Detector raises an alert after `required` high samples in a row.
// Only one alert per spike: it resets on the first normal sample.
type Detector struct {
	required    int
	consecutive int
	triggered   bool
}

func NewDetector(required int) *Detector {
	return &Detector{required: required}
}

// Update returns true only on the sample that starts an incident.
func (d *Detector) Update(high bool) bool {
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

// PendingAlert is true when samples are high but no incident was saved yet.
func (d *Detector) PendingAlert() bool {
	return d.consecutive > 0 && !d.triggered
}
