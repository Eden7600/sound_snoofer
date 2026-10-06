package media

import (
	"strings"
	"time"

	"sound-snoofer/snoofer"
)

// mediaKeys tracks the fire-and-forget SendInput integration: it has no peer
// connection, only the outcome of the last command.
type mediaKeys struct {
	link    snoofer.ConnectionTracker
	failing bool
	last    string
	lastAt  time.Time
}

func (m *mediaKeys) sent(id string, err error, now time.Time) {
	m.last, m.lastAt = strings.TrimPrefix(id, "media."), now
	m.failing = err != nil
	if err != nil {
		m.link.Fail(err.Error(), now)
		return
	}
	m.link.Activity(now)
}

func (m *mediaKeys) report(live bool, now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionReady, "Ready"
	switch {
	case !live:
		state, value = snoofer.ConnectionOff, "Preview"
	case m.failing:
		state, value = snoofer.ConnectionError, "Error"
	}
	m.link.Observe(state, now)
	var details []snoofer.ConnectionDetail
	if m.last != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Last command", Value: m.last})
	}
	details = append(details, snoofer.ConnectionDetail{Label: "Player feedback", Value: "None (keys are fire-and-forget)"})
	return snoofer.Control{ID: "media.app-keys", Label: "Windows media keys", Group: "Windows media", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: m.link.Report("user32 SendInput", details...)}
}
