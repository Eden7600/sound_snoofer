package audio

import (
	"testing"

	"sound-snoofer/internal/control"
	"sound-snoofer/snoofer"
)

func TestHealthAttentionUsesErrorStateAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state control.State
		want  snoofer.AttentionLevel
	}{
		{"ordinary health text", control.State{Health: "Error"}, ""},
		{"no state", control.State{}, ""},
		{"error", control.State{Error: "Running"}, snoofer.AttentionCritical},
		{"state error", control.State{StateError: "Off"}, snoofer.AttentionCritical},
		{"stall", control.State{Stalled: true}, snoofer.AttentionCritical},
		{"error notice", control.State{Notice: "Saved", NoticeKind: control.NoticeError}, snoofer.AttentionCritical},
		{"success notice", control.State{Notice: "error in track title", NoticeKind: control.NoticeSuccess}, ""},
		{"ordinary pending", control.State{Notice: "Pending", NoticeKind: control.NoticePending}, ""},
		{"recovery", control.State{RecoveryPending: true}, snoofer.AttentionWarning},
		{"failure takes precedence", control.State{RecoveryPending: true, Stalled: true}, snoofer.AttentionCritical},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := controlByID(t, controls(tc.state), "audio.health").Attention; got != tc.want {
				t.Fatalf("%q, want %q", got, tc.want)
			}
		})
	}
	if got := controlByID(t, controls(control.State{Connected: true, Live: true}), "audio.health").Attention; got != "" {
		t.Fatalf("recovery retained %q", got)
	}
}
