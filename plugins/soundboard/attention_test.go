package soundboard

import (
	"errors"
	"testing"

	"sound-snoofer/snoofer"
)

func TestLibraryAttentionMatchesVisibleDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name           string
		live           bool
		diagnostic     string
		scanErr, ready error
		clips          []clip
		note           string
		want           snoofer.AttentionLevel
	}{
		{name: "empty", live: true},
		{name: "scan", live: true, scanErr: errors.New("Running"), note: "Running", want: snoofer.AttentionCritical},
		{name: "playback", live: true, diagnostic: "Off", note: "Off", want: snoofer.AttentionCritical},
		{name: "artwork", live: true, clips: []clip{{ArtworkError: "Saved"}}, note: "Saved", want: snoofer.AttentionWarning},
		{name: "readiness uses connection", live: true, diagnostic: "hidden error", ready: errors.New("Routing not ready"), note: "Routing not ready"},
		{name: "scan precedence", live: true, scanErr: errors.New("scan"), ready: errors.New("routing"), note: "scan", want: snoofer.AttentionCritical},
		{name: "preview", diagnostic: "hidden error", scanErr: errors.New("scan"), ready: errors.New("routing"), clips: []clip{{ArtworkError: "art"}}, note: "Preview"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			note, attention := libraryStatus(tc.live, tc.diagnostic, tc.scanErr, tc.ready, tc.clips)
			if note != tc.note || attention != tc.want {
				t.Fatalf("%q/%q, want %q/%q", note, attention, tc.note, tc.want)
			}
		})
	}
}
