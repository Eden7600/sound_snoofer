package soundboard

import (
	"time"

	"sound-snoofer/snoofer"
)

// playbackReport describes the native playback companion and its DirectSound
// renderer. The companion loads lazily on the first clip, so an unloaded but
// routable soundboard is Ready rather than disconnected.
func playbackReport(link *snoofer.ConnectionTracker, live bool, ready error, loaded bool, renderer, lastClip string, now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionReady, "Ready"
	routing := "Ready"
	switch {
	case !live:
		state, value = snoofer.ConnectionOff, "Preview"
	case ready != nil:
		state, value, routing = snoofer.ConnectionAttention, "Routing not ready", ready.Error()
	case loaded:
		state, value = snoofer.ConnectionConnected, "Loaded"
	}
	link.Observe(state, now)
	companion := "Not loaded (loads on first clip)"
	if loaded {
		companion = "Loaded"
	}
	details := []snoofer.ConnectionDetail{{Label: "Companion", Value: "snoofer-soundboard.dll · " + companion}, {Label: "Routing", Value: routing}}
	if lastClip != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Last clip", Value: lastClip})
	}
	return snoofer.Control{ID: "soundboard.app-playback", Label: "Soundboard playback", Group: "Soundboard", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: link.Report(renderer, details...)}
}
