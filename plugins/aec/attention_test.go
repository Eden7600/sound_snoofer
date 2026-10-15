package aec

import (
	"testing"

	engine "sound-snoofer/internal/aec"
	"sound-snoofer/snoofer"
)

func TestStatusAttentionUsesCurrentFailureState(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*worker)
		want snoofer.AttentionLevel
	}{
		{"off", func(w *worker) { w.settings.Mode = "off" }, ""},
		{"inactive", func(w *worker) { w.reason = "Microphone muted" }, ""},
		{"loading", func(w *worker) { w.loading = true }, ""},
		{"active", func(w *worker) {
			w.active = true
			w.stats = engine.Stats{Active: true, SampleRate: 48000}
		}, ""},
		{"engine", func(w *worker) { w.engineErr = "Running" }, snoofer.AttentionCritical},
		{"release", func(w *worker) { w.releaseErr = "Off" }, snoofer.AttentionCritical},
		{"retry", func(w *worker) { w.retryErr = "Saved" }, snoofer.AttentionCritical},
		{"statistics", func(w *worker) { w.statsErr = "Muted" }, snoofer.AttentionCritical},
		{"failed", func(w *worker) { w.stats.Failed = true }, snoofer.AttentionCritical},
		{"unsupported rate", func(w *worker) {
			w.active = true
			w.stats.SampleRate = 44100
		}, snoofer.AttentionWarning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorker(Settings{}, &fakeMixer{}, &fakeEngine{})
			tc.set(w)
			if got := control(t, w, "aec.status").Attention; got != tc.want {
				t.Fatalf("attention %q, want %q", got, tc.want)
			}
			w.loading, w.active = false, false
			w.releaseErr, w.retryErr, w.engineErr, w.statsErr = "", "", "", ""
			w.stats = engine.Stats{}
			if got := control(t, w, "aec.status").Attention; got != "" {
				t.Fatalf("recovered status retained %q", got)
			}
		})
	}
}
