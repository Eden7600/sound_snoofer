package hue

import (
	"context"
	"errors"
	"time"

	"sound-snoofer/snoofer"
)

// motionLink tracks requested sensor states. The bridge owns the state:
// requests are never persisted or reapplied, and the control's value always
// comes from observed resources.
type motionLink struct {
	want    map[string]bool      // Requested enabled flag by motion service ID.
	sent    map[string]time.Time // When each request was accepted, for confirmation timeout.
	writing int                  // Requests in flight.
	err     string
}

// motionValue summarizes observed sensors as On, Off or Mixed. Sensors whose
// flag was never reported make the value unknown.
func motionValue(sensors []motionSensor) string {
	enabled := 0
	for _, s := range sensors {
		if !s.Known {
			return "N/A"
		}
		if s.Enabled {
			enabled++
		}
	}
	switch enabled {
	case len(sensors):
		return "On"
	case 0:
		return "Off"
	}
	return "Mixed"
}

func (w *worker) roomSensors() []motionSensor {
	if !w.connected || w.settings.Group == "" {
		return nil
	}
	return w.model.motionSensors(w.settings.Group)
}

// toggleMotion disables every sensor in the room when all are on, and
// otherwise enables every sensor.
type motionOperation struct {
	command   command
	remaining int
	errs      []error
}

func (w *worker) toggleMotion(ctx context.Context, c command) {
	finish := func(o snoofer.Outcome) {
		if c.complete != nil {
			c.complete(o)
		}
	}
	if !w.connected || len(c.sensors) == 0 {
		finish(rejected("Motion sensors unavailable"))
		return
	}
	if w.motionAction != nil {
		finish(rejected("Motion request already in progress"))
		return
	}
	w.motion.err = ""
	w.motion.want = map[string]bool{}
	w.motion.sent = map[string]time.Time{}
	op := &motionOperation{command: c}
	w.motionAction = op
	client := w.client
	for _, sensor := range c.sensors {
		current := w.model.resources[sensor.ID]
		if current.Enabled != nil && *current.Enabled == c.want {
			continue
		}
		id := sensor.ID
		w.motion.want[id] = c.want
		w.motion.writing++
		op.remaining++
		w.spawn(ctx, func(ctx context.Context) func(context.Context) {
			err := client.Put(ctx, "motion", id, map[string]bool{"enabled": c.want})
			return func(context.Context) {
				if w.motionAction != op {
					return
				}
				if c.target != w.capture(c.request.ID).target {
					w.resetMotion()
					return
				}
				op.remaining--
				w.motion.writing--
				if err != nil {
					op.errs = append(op.errs, err)
					delete(w.motion.want, id)
					w.motion.err = err.Error()
				} else {
					w.motion.sent[id] = time.Now()
					w.observeMotion()
				}
				if op.remaining == 0 {
					w.motionAction = nil
					finish(operationOutcome(errors.Join(op.errs...)))
				}
			}
		})
	}
	if op.remaining == 0 {
		w.motionAction = nil
		finish(processed("Sensors already set"))
	}
}

// observeMotion clears requests the bridge has reported as applied.
func (w *worker) observeMotion() {
	for id, want := range w.motion.want {
		r, ok := w.model.resources[id]
		if _, accepted := w.motion.sent[id]; accepted && ok && r.Enabled != nil && *r.Enabled == want {
			delete(w.motion.want, id)
			delete(w.motion.sent, id)
		}
	}
}

// expireMotion reports requests the bridge accepted but never reflected.
func (w *worker) expireMotion(now time.Time) bool {
	changed := false
	for id, sent := range w.motion.sent {
		if now.Sub(sent) > w.timing.confirm {
			delete(w.motion.want, id)
			delete(w.motion.sent, id)
			w.motion.err = "Not confirmed by bridge"
			changed = true
		}
	}
	return changed
}

// resetMotion drops requests from a previous connection.
func (w *worker) resetMotion() {
	if op := w.motionAction; op != nil {
		w.motionAction = nil
		if op.command.complete != nil {
			op.command.complete(snoofer.Outcome{Status: snoofer.Unknown, Detail: "Motion target changed"})
		}
	}
	w.motion = motionLink{}
}

func (w *worker) motionControl() snoofer.Control {
	sensors := w.roomSensors()
	value := "N/A"
	if len(sensors) > 0 {
		value = motionValue(sensors)
	}
	icon := "hue-motion"
	if value == "Off" {
		icon = "hue-motion-off"
	}
	status := w.motion.err
	if len(w.motion.want) > 0 {
		status = "Pending"
	}
	return snoofer.Control{ID: "hue.motion", Label: "Hue Motion Sensors", ShortLabel: "Motion", Group: "Hue", Kind: "toggle", Icon: icon,
		Value: value, Status: status, Operations: []string{"press"},
		Available: w.services.Live && len(sensors) > 0,
		// Blank on the deck while connected to a room without sensors; N/A while disconnected.
		Hidden: w.connected && len(sensors) == 0}
}
