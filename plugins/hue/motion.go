package hue

import (
	"context"
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
func (w *worker) toggleMotion(ctx context.Context) {
	sensors := w.roomSensors()
	if len(sensors) == 0 || w.motion.writing > 0 {
		return
	}
	enable := motionValue(sensors) != "On"
	w.motion.err = ""
	if w.motion.want == nil {
		w.motion.want = map[string]bool{}
		w.motion.sent = map[string]time.Time{}
	}
	client, generation := w.client, w.generation
	for _, sensor := range sensors {
		if sensor.Known && sensor.Enabled == enable {
			continue
		}
		id := sensor.ID
		w.motion.want[id] = enable
		w.motion.writing++
		w.spawn(ctx, func(ctx context.Context) func(context.Context) {
			err := client.Put(ctx, "motion", id, map[string]bool{"enabled": enable})
			return func(context.Context) { w.motionWritten(generation, id, err) }
		})
	}
}

func (w *worker) motionWritten(generation int, id string, err error) {
	if generation != w.generation {
		return
	}
	w.motion.writing--
	if err != nil {
		delete(w.motion.want, id)
		w.motion.err = err.Error()
		return
	}
	w.motion.sent[id] = time.Now()
	w.observeMotion()
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
	return snoofer.Control{ID: "hue.motion", Label: "Hue motion sensors", ShortLabel: "Motion", Group: "Hue", Kind: "toggle", Icon: icon,
		Value: value, Status: status, Operations: []string{"press"},
		Available: w.services.Live && len(sensors) > 0,
		// Blank on the deck while connected to a room without sensors; N/A while disconnected.
		Hidden: w.connected && len(sensors) == 0}
}
