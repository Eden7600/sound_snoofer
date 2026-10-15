package audio

import (
	"context"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/snoofer"
)

func TestMeetingsUsesExistingAudioSaveBoundary(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	meetings := snoofer.NewMeetings(snoofer.NewControls(), true)
	if err := meetings.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stop, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		if err := meetings.Close(stop); err != nil {
			t.Error(err)
		}
	})
	i := &Instance{ctx: ctx, actions: make(chan control.Action, 8), state: control.State{Revision: 11, Intent: &config.Intent{MicMuted: true}}}
	if err := i.attachMeetings(snoofer.Services{Meetings: meetings}); err != nil {
		t.Fatal(err)
	}
	i.reportMeetingMute()
	if !meetings.Snapshot().Mute.Value {
		t.Fatal("saved preference not reported")
	}
	results := make(chan snoofer.Outcome, 1)
	if err := meetings.Request(ctx, snoofer.MeetingMute, meetings.Snapshot().Mute.Revision, false, func(o snoofer.Outcome) { results <- o }); err != nil {
		t.Fatal(err)
	}
	var action control.Action
	select {
	case action = <-i.actions:
	case <-time.After(time.Second):
		t.Fatal("no serialized audio action")
	}
	if action.Kind != control.Edit || action.Row != "mic-mute" || action.Value != "false" || action.Revision != 11 || action.Origin != "meetings" || action.Complete == nil {
		t.Fatalf("wrong save path: %+v", action)
	}
	if !meetings.Snapshot().Mute.Value {
		t.Fatal("target changed before audio save")
	}
	select {
	case <-results:
		t.Fatal("request confirmed before save")
	default:
	}
	// Simulate the native worker's successful save and resulting publication.
	i.mu.Lock()
	i.state = control.State{Revision: 12, Intent: &config.Intent{MicMuted: false}}
	i.mu.Unlock()
	i.reportMeetingMute()
	action.Complete(snoofer.Outcome{Status: snoofer.Processed, Detail: "Saved"})
	select {
	case result := <-results:
		if result.Status != snoofer.Processed {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("save completion not forwarded")
	}
	if meetings.Snapshot().Mute.Value {
		t.Fatal("saved state did not reach shared target")
	}
}

func TestMeetingsAudioRegistrationIsOptionalAndStopsCleanly(t *testing.T) {
	i := &Instance{}
	if err := i.attachMeetings(snoofer.Services{}); err != nil {
		t.Fatal(err)
	}
	i.reportMeetingMute()
	ctx, cancel := context.WithCancel(context.Background())
	meetings := snoofer.NewMeetings(snoofer.NewControls(), true)
	if err := meetings.Start(ctx); err != nil {
		t.Fatal(err)
	}
	i.ctx = ctx
	i.actions = make(chan control.Action, 8)
	if err := i.attachMeetings(snoofer.Services{Meetings: meetings}); err != nil {
		t.Fatal(err)
	}
	cancel()
	stop, done := context.WithTimeout(context.Background(), time.Second)
	defer done()
	if err := meetings.Close(stop); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-i.actions:
		t.Fatal("shutdown queued audio work", action)
	default:
	}
}
