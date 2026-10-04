package controller

import (
	"strings"
	"time"

	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

// ParameterBackend omits enumeration only inside a numeric-only transaction.
type ParameterBackend interface {
	ParameterSnapshot() (model.Snapshot, error)
}

func (c *Controller) observe(fast bool) (model.Snapshot, error) {
	read := c.Backend.Snapshot
	if b, ok := c.Backend.(ParameterBackend); fast && ok {
		read = b.ParameterSnapshot
	}
	s, err := read()
	if err == nil && c.Config.Studio != nil && c.Config.Studio.Recording != nil {
		r := c.readRecorder()
		s.Recorder = &r
	}
	return s, err
}
func operationValue(s model.Snapshot, param string) (float32, bool) {
	if strings.HasPrefix(param, "Recorder.") {
		if s.Recorder == nil || s.Recorder.Error != "" {
			return 0, false
		}
		v, ok := s.Recorder.Values[param]
		return v, ok
	}
	v, ok := s.Numbers[param]
	return v, ok
}
func verificationInterval(op routing.Operation) time.Duration {
	if op.Device != nil {
		return 100 * time.Millisecond
	}
	return 5 * time.Millisecond
}
