package audio

import (
	"context"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/snoofer"
)

func TestMuteRegistryRevisionSurvivesOtherPreferenceEdit(t *testing.T) {
	state := control.State{Revision: 10, MicMuteGeneration: 1, PlaybackMuteGeneration: 1, Intent: &config.Intent{Enabled: true, Source: "desk", Mode: "direct", Monitor: "off"}}
	registry := snoofer.NewControls()
	handler := func(context.Context, snoofer.Request) error { return nil }
	if err := registry.Publish("audio", controls(state), handler); err != nil {
		t.Fatal(err)
	}
	var request snoofer.Request
	for _, c := range registry.Snapshot() {
		if c.ID == "audio.speaker-mute" {
			request = snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
		}
	}
	if request.ID == "" {
		t.Fatal("missing playback mute control")
	}
	state.Revision++
	state.Intent = state.Intent.Clone()
	state.Intent.MicMuted = true
	if err := registry.Publish("audio", controls(state), handler); err != nil {
		t.Fatal(err)
	}
	if err := registry.Dispatch(context.Background(), request); err != nil {
		t.Fatal("unrelated mute edit invalidated captured Playback Mute", err)
	}
}
