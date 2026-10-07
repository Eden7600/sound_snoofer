package insta360

import (
	"context"
	"fmt"
	"time"

	"sound-snoofer/snoofer"
)

var (
	trackingOptions = []string{"off", "single", "group"}
	trackingLabels  = map[string]string{"off": "Off", "single": "Single", "group": "Group"}
	framingOptions  = []string{"head", "half", "full"}
	framingLabels   = map[string]string{"head": "Head", "half": "Half body", "full": "Full body"}
)

// unavailable is why camera controls cannot be used, or "".
func (w *worker) unavailable() string {
	switch {
	case w.dev == nil && w.absent:
		return "No camera"
	case w.dev == nil:
		return "Unavailable"
	case !w.services.Live:
		return "Preview"
	}
	return ""
}

// controlStatus is a control's status word: pending, a recorded outcome or a
// reason it is unavailable.
func (w *worker) controlStatus(id string, blocked string) string {
	if _, ok := w.pending[id]; ok {
		return "Pending"
	}
	if note := w.notes[id]; note != "" {
		return note
	}
	return blocked
}

func (w *worker) controls(now time.Time) []snoofer.Control {
	reason := w.unavailable()
	// Privacy hides the lens; the camera ignores tracking and framing until it ends.
	gated := reason
	if gated == "" && w.status.Privacy {
		gated = "Privacy"
	}
	privacyIcon := "camera-privacy-off"
	if w.status.Privacy {
		privacyIcon = "camera-privacy"
	}
	tracking := w.observed("insta360.tracking")
	framing := w.observed("insta360.framing")
	return []snoofer.Control{
		{ID: "insta360.privacy", Label: "Camera privacy", ShortLabel: "Privacy", Group: "Camera", Kind: "toggle", Icon: privacyIcon,
			Value: w.observed("insta360.privacy"), Status: w.controlStatus("insta360.privacy", reason), Operations: []string{"press"}, Available: reason == "" && w.status.PrivacyKnown},
		{ID: "insta360.tracking", Label: "Camera tracking", ShortLabel: "Tracking", Group: "Camera", Kind: "selection", Icon: "tracking",
			Value: tracking, Options: trackingOptions, OptionLabels: trackingLabels, Status: w.controlStatus("insta360.tracking", gated), Operations: []string{"set"}, Available: gated == "" && w.status.ModeKnown},
		{ID: "insta360.framing", Label: "Camera framing", ShortLabel: "Framing", Group: "Camera", Kind: "selection", Icon: "framing",
			Value: framing, Options: framingOptions, OptionLabels: framingLabels, Status: w.controlStatus("insta360.framing", gated), Operations: []string{"set"}, Available: gated == "" && w.framingKnown},
		{ID: "insta360.reset", Label: "Reset camera position", ShortLabel: "Reset", Group: "Camera", Kind: "command", Icon: "camera-reset",
			Status: w.controlStatus("insta360.reset", gated), Operations: []string{"press"}, Available: gated == ""},
		{ID: "insta360.state", Label: "Camera tracking state", ShortLabel: "State", Group: "Camera", Kind: "status", Icon: "tracking",
			Value: w.stateValue(), Status: reason, Available: true},
		w.report(now),
	}
}

func (w *worker) stateValue() string {
	if w.dev == nil {
		return "N/A"
	}
	if w.status.Privacy {
		return "Privacy"
	}
	return stateWord(w.status)
}

func (w *worker) report(now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionConnected, "Connected"
	switch {
	case w.dev == nil && w.absent:
		state, value = snoofer.ConnectionDisconnected, "No camera"
	case w.dev == nil && w.lastOpen.IsZero():
		state, value = snoofer.ConnectionConnecting, "Searching"
	case w.dev == nil:
		state, value = snoofer.ConnectionError, "Error"
	}
	w.link.Observe(state, now)
	details := []snoofer.ConnectionDetail{{Label: "Model", Value: "Link 2"}, {Label: "Interval", Value: readInterval.String()}}
	if w.dev != nil {
		details = append(details, snoofer.ConnectionDetail{Label: "Tracking", Value: w.stateValue()})
	}
	return snoofer.Control{ID: "insta360.app-camera", Label: "Insta360 Link 2", Group: "Insta360", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: w.link.Report("USB "+devicePath, details...)}
}

func (w *worker) publish(now time.Time) {
	requests := w.requests
	_ = w.services.Controls.Publish("insta360", w.controls(now), func(ctx context.Context, r snoofer.Request) error {
		select {
		case requests <- r:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("camera queue full")
		}
	})
}
