package nowplaying

import (
	"encoding/json"
	"testing"

	"sound-snoofer/snoofer"
)

func TestStatusAttentionTracksOnlyVisibleMediaFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*worker)
		want snoofer.AttentionLevel
	}{
		{"idle", func(*worker) {}, ""},
		{"windows", func(w *worker) { w.winErr = "Connected" }, snoofer.AttentionCritical},
		{"bridge", func(w *worker) { w.bridgeErr = "Running" }, snoofer.AttentionCritical},
		{"deck save", func(w *worker) { w.saveErr = "Off" }, ""},
		{"retired session", func(w *worker) { w.failure["gone"] = "Failed" }, ""},
		{"session", func(w *worker) {
			w.sessions = []session{{Key: "current", Title: "Player"}}
			w.failure["current"] = "Muted"
		}, snoofer.AttentionCritical},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, true)
			tc.set(r.w)
			r.w.publish(r.commands, r.now)
			var view statusView
			if err := json.Unmarshal(r.control("nowplaying.status").ViewData, &view); err != nil {
				t.Fatal(err)
			}
			if view.WindowsError != r.w.winErr {
				t.Fatalf("WindowsError = %q, want %q", view.WindowsError, r.w.winErr)
			}
			if got := r.control("nowplaying.status").Attention; got != tc.want {
				t.Fatalf("%q, want %q", got, tc.want)
			}
			r.w.winErr, r.w.bridgeErr, r.w.saveErr = "", "", ""
			clear(r.w.failure)
			r.w.publish(r.commands, r.now)
			if got := r.control("nowplaying.status").Attention; got != "" {
				t.Fatalf("recovery retained %q", got)
			}
			var recovered map[string]json.RawMessage
			if err := json.Unmarshal(r.control("nowplaying.status").ViewData, &recovered); err != nil {
				t.Fatal(err)
			}
			if _, ok := recovered["WindowsError"]; ok {
				t.Fatal("recovered view retained WindowsError")
			}
		})
	}
}

func TestPendingSessionHidesRememberedFailureAttention(t *testing.T) {
	r := newRig(t, true)
	r.w.sessions = []session{{Key: "current", Title: "Player"}}
	r.w.failure["current"] = "Muted"
	r.w.pending["current"] = pendingCommand{}
	r.w.publish(r.commands, r.now)
	if got := r.control("nowplaying.status").Attention; got != "" {
		t.Fatalf("pending session exposes remembered failure: %q", got)
	}
	delete(r.w.pending, "current")
	r.w.publish(r.commands, r.now)
	if got := r.control("nowplaying.status").Attention; got != snoofer.AttentionCritical {
		t.Fatalf("visible failure after pending cleared = %q", got)
	}
}
