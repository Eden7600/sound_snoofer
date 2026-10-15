package discord

import (
	"context"
	"fmt"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func capturedMeeting(w *worker, property snoofer.MeetingProperty, want bool) (snoofer.MeetingCommand, <-chan snoofer.Outcome) {
	results := make(chan snoofer.Outcome, 4)
	c := snoofer.MeetingCommand{Context: context.Background(), Peer: "discord", Lifetime: w.meetingLifetime(), Property: property, Revision: w.meetingValue(property).Revision, Value: want, TargetRevision: 1,
		Complete: func(outcome snoofer.Outcome) { results <- outcome }}
	return c, results
}

func requestTarget(t *testing.T, w *worker, ctx context.Context, property snoofer.MeetingProperty, value bool) {
	t.Helper()
	target := w.sharedTarget(property)
	done := make(chan snoofer.Outcome, 1)
	if err := w.services.Meetings.Request(ctx, property, target.Revision, value, func(outcome snoofer.Outcome) { done <- outcome }); err != nil {
		t.Fatal(err)
	}
	expectOutcome(t, done, snoofer.Processed)
}

func TestSharedDiscordCameraRequiresExactKnownCurrentLifetime(t *testing.T) {
	for _, change := range []string{"connection", "channel-aba", "value-aba", "unknown", "canceled", "preview"} {
		t.Run(change, func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			now := time.Unix(1000, 0)
			c, result := capturedMeeting(w, snoofer.MeetingCamera, true)
			switch change {
			case "connection":
				w.generation++
			case "channel-aba":
				w.leftChannel()
				w.selectChannel("42", "Same")
				w.voice.videoKnown = true
			case "value-aba":
				w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{"active": true}), now)
				w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{"active": false}), now)
			case "unknown":
				w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{}), now)
			case "canceled":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				c.Context = canceled
			case "preview":
				w.services.Live = false
			}
			w.reportMeetings(now)
			w.applyMeeting(c, now)
			expectOutcome(t, result, snoofer.Rejected)
			if pipe.Len() != 0 {
				t.Fatal("obsolete camera operation wrote")
			}
		})
	}
}

func TestSharedDiscordCameraCompletesOnlyOnReadback(t *testing.T) {
	for _, readbackFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(readbackFirst), func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			now := time.Unix(1000, 0)
			c, result := capturedMeeting(w, snoofer.MeetingCamera, true)
			w.applyMeeting(c, now)
			sent := readCommand(t, pipe)
			if sent.Cmd != "TOGGLE_VIDEO" {
				t.Fatal(sent)
			}
			if !readbackFirst {
				w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now)
				assertNoOutcome(t, result)
				if w.voice.video {
					t.Fatal("RPC acceptance invented video state")
				}
			}
			w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{"active": true}), now)
			w.syncMeetings(now)
			expectOutcome(t, result, snoofer.Processed)
			w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now)
			assertNoOutcome(t, result)
			if pipe.Len() != 0 || len(w.meetingWrites) != 0 {
				t.Fatal("replayed or retained observed command")
			}
		})
	}
}

func TestSharedDiscordAlreadyEqualIsObservedWithoutToggle(t *testing.T) {
	w, pipe, _ := outcomeWorker(t)
	c, result := capturedMeeting(w, snoofer.MeetingCamera, false)
	w.applyMeeting(c, time.Unix(1000, 0))
	expectOutcome(t, result, snoofer.Processed)
	if pipe.Len() != 0 {
		t.Fatal("equal camera target toggled")
	}
}

func TestSharedDiscordReportsEveryExternalEdge(t *testing.T) {
	w, _, ctx := outcomeWorker(t)
	first := w.sharedTarget(snoofer.MeetingCamera)
	w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{"active": true}), time.Unix(1000, 0))
	on := w.sharedTarget(snoofer.MeetingCamera)
	w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{"active": false}), time.Unix(1000, 0))
	off := w.sharedTarget(snoofer.MeetingCamera)
	if !on.Known || !on.Value || off.Value || on.Revision <= first.Revision || off.Revision <= on.Revision {
		t.Fatal("event edges coalesced", first, on, off)
	}
}

func TestSharedDiscordLatestIntentWaitsForOldEffectWithoutAdoptingEcho(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	requestTarget(t, w, ctx, snoofer.MeetingCamera, true)
	w.syncMeetings(now)
	first := readCommand(t, pipe)
	old := w.meetingWrites[snoofer.MeetingCamera]
	if old == nil {
		t.Fatal("missing pending camera")
	}
	requestTarget(t, w, ctx, snoofer.MeetingCamera, false)
	if old.command.Context.Err() == nil {
		t.Fatal("new intent did not cancel obsolete work")
	}
	w.syncMeetings(now)
	if pipe.Len() != 0 {
		t.Fatal("second toggle sent before first effect resolved")
	}
	w.dispatch(ctx, message{Cmd: first.Cmd, Nonce: first.Nonce}, now)
	w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{"active": true}), now)
	w.syncMeetings(now)
	if w.sharedTarget(snoofer.MeetingCamera).Value {
		t.Fatal("old echo overwrote latest target")
	}
	second := readCommand(t, pipe)
	if second.Cmd != "TOGGLE_VIDEO" {
		t.Fatal(second)
	}
	w.event(ctx, "VIDEO_STATE_UPDATE", raw(map[string]any{"active": false}), now)
	w.syncMeetings(now)
	w.dispatch(ctx, message{Cmd: second.Cmd, Nonce: second.Nonce}, now)
	if pipe.Len() != 0 || len(w.meetingWrites) != 0 || w.sharedTarget(snoofer.MeetingCamera).Value {
		t.Fatal("latest camera target did not converge")
	}
}

func TestSharedDiscordDeafenDoesNotAdoptSyntheticMute(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	w.event(ctx, "VOICE_SETTINGS_UPDATE", raw(map[string]any{"mute": true, "deaf": true}), now)
	w.syncMeetings(now)
	if w.sharedTarget(snoofer.MeetingMute).Value || pipe.Len() != 0 {
		t.Fatal("synthetic mute adopted")
	}
	requestTarget(t, w, ctx, snoofer.MeetingMute, true)
	w.syncMeetings(now)
	if pipe.Len() != 0 {
		t.Fatal("mute written while deafened")
	}
	w.event(ctx, "VOICE_SETTINGS_UPDATE", raw(map[string]any{"mute": false, "deaf": false}), now)
	w.syncMeetings(now)
	sent := readCommand(t, pipe)
	if sent.Cmd != "SET_VOICE_SETTINGS" || sent.Args.(map[string]any)["mute"] != true {
		t.Fatal("undeafen did not impose target", sent)
	}
	w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce, Data: raw(map[string]any{"mute": true, "deaf": false})}, now)
	if len(w.meetingWrites) != 1 {
		t.Fatal("RPC response substituted for mute event")
	}
	w.event(ctx, "VOICE_SETTINGS_UPDATE", raw(map[string]any{"mute": true, "deaf": false}), now)
	w.syncMeetings(now)
	if len(w.meetingWrites) != 0 || pipe.Len() != 0 {
		t.Fatal("mute readback failed or looped")
	}
}

func TestSharedDiscordUncertainMuteDoesNotReplayAcrossReconnect(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	requestTarget(t, w, ctx, snoofer.MeetingMute, true)
	w.syncMeetings(now)
	readCommand(t, pipe)
	w.syncMeetings(now.Add(4 * time.Second))
	if w.sharedTarget(snoofer.MeetingMute).Status != "Not confirmed" {
		t.Fatal(w.sharedTarget(snoofer.MeetingMute))
	}
	w.drop(nil, now)
	w.conn = pipe
	w.generation++
	w.state = stateReady
	w.selectChannel("42", "Same call")
	w.voice.settingsKnown = true
	w.syncMeetings(now)
	if w.meetingReported.NewCall || pipe.Len() != 0 {
		t.Fatal("reconnect replayed uncertainty")
	}
	requestTarget(t, w, ctx, snoofer.MeetingMute, false)
	requestTarget(t, w, ctx, snoofer.MeetingMute, true)
	w.syncMeetings(now)
	if sent := readCommand(t, pipe); sent.Cmd != "SET_VOICE_SETTINGS" {
		t.Fatal(sent)
	}
}

func TestSharedDiscordMicAndCameraHaveIndependentReadback(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	requestTarget(t, w, ctx, snoofer.MeetingMute, true)
	requestTarget(t, w, ctx, snoofer.MeetingCamera, true)
	w.syncMeetings(now)
	readCommand(t, pipe)
	readCommand(t, pipe)
	w.event(ctx, "VOICE_SETTINGS_UPDATE", raw(map[string]any{"mute": true, "deaf": false}), now)
	w.syncMeetings(now)
	if w.meetingWrites[snoofer.MeetingMute] != nil || w.meetingWrites[snoofer.MeetingCamera] == nil {
		t.Fatal("mute settled independent camera")
	}
	w.syncMeetings(now.Add(4 * time.Second))
	if target := w.sharedTarget(snoofer.MeetingCamera); target.Status != "Not confirmed" || !target.Value {
		t.Fatal(target)
	}
	if target := w.sharedTarget(snoofer.MeetingMute); target.Status != "" || !target.Value {
		t.Fatal(target)
	}
}

func TestLegacyDiscordBindingsUseSharedRevisionAndActualReadouts(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	queued := dispatchOutcome(t, w, ctx, "discord.video")
	requestTarget(t, w, ctx, snoofer.MeetingCamera, true)
	w.handle(ctx, <-w.requests, now)
	expectOutcome(t, queued, snoofer.Rejected)
	if pipe.Len() != 0 {
		t.Fatal("compatibility binding bypassed shared revision")
	}
	w.publish(ctx, now)
	for _, c := range w.services.Controls.Snapshot() {
		switch c.ID {
		case "discord.video":
			if c.Value != "On" || c.Mirrors != "meetings.camera" {
				t.Fatal(c)
			}
		case "discord.call-camera":
			if c.Value != "Off" || !c.SurfaceOnly || len(c.Operations) != 0 {
				t.Fatal(c)
			}
		}
	}
	// Shared controls remain usable if this provider loses its connection.
	w.conn = nil
	result := dispatchOutcome(t, w, ctx, "discord.video")
	w.handle(ctx, <-w.requests, now)
	expectOutcome(t, result, snoofer.Processed)
	if w.sharedTarget(snoofer.MeetingCamera).Value {
		t.Fatal("alias failed to change shared intent")
	}
}

func TestSharedDiscordUnchangedReportsDoNotWakeWorkerAgain(t *testing.T) {
	w, _, _ := outcomeWorker(t)
	select {
	case <-w.meetings.Changed():
	default:
	}
	for range 20 {
		w.syncMeetings(time.Unix(1000, 0))
	}
	select {
	case <-w.meetings.Changed():
		t.Fatal("unchanged report fed the worker's wake channel")
	default:
	}
}

func TestSharedDiscordPublicationUsesOneTargetSnapshot(t *testing.T) {
	w, _, ctx := outcomeWorker(t)
	targets := w.meetingSnapshot()
	requestTarget(t, w, ctx, snoofer.MeetingCamera, true)
	controls := w.controlsFor(time.Unix(1000, 0), targets)
	c := w.captureTargets(snoofer.Request{ID: "discord.video"}, targets)
	for _, control := range controls {
		if control.ID == "discord.video" {
			if control.Value != "Off" || control.Epoch != targets.Camera.Revision || c.targetRevision != control.Epoch || !c.desired {
				t.Fatal("display and captured action used different snapshots", control, c)
			}
		}
	}
	result := make(chan snoofer.Outcome, 1)
	c.complete = func(outcome snoofer.Outcome) { result <- outcome }
	w.handle(ctx, c, time.Unix(1000, 0))
	expectOutcome(t, result, snoofer.Rejected)
}

func TestSharedDiscordTerminalFailuresSettleOnce(t *testing.T) {
	for _, path := range []string{"timeout", "drop", "write-error", "close", "rpc-error"} {
		t.Run(path, func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			now := time.Unix(1000, 0)
			c, result := capturedMeeting(w, snoofer.MeetingCamera, true)
			if path == "write-error" {
				pipe.writeErr = fmt.Errorf("broken pipe")
			}
			w.applyMeeting(c, now)
			nonce := fmt.Sprint(w.nonce)
			want := snoofer.Unknown
			switch path {
			case "timeout":
				// A delayed worker pass expires both the RPC and readback deadlines.
				w.step(ctx, now.Add(requestTimeout+time.Second))
			case "drop":
				w.drop(fmt.Errorf("lost connection"), now)
			case "close":
				w.closeConn()
			case "rpc-error":
				want = snoofer.Failed
				w.dispatch(ctx, message{Cmd: "TOGGLE_VIDEO", Nonce: nonce, Evt: "ERROR", Data: raw(rpcError{Message: "Denied"})}, now)
			}
			expectOutcome(t, result, want)
			before := pipe.Len()
			w.dispatch(ctx, message{Cmd: "TOGGLE_VIDEO", Nonce: nonce}, now)
			assertNoOutcome(t, result)
			if len(w.calls) != 0 || len(w.meetingWrites) != 0 || pipe.Len() != before {
				t.Fatal("terminal meeting command was retained or replayed")
			}
		})
	}
}

func TestDiscordMuteAliasWaitsForSavedPreferenceBeforeNativeWrite(t *testing.T) {
	for _, saved := range []bool{false, true} {
		t.Run(fmt.Sprint(saved), func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			type save struct {
				value    bool
				revision uint64
				complete snoofer.Completion
			}
			saves := make(chan save, 1)
			audio, err := w.services.Meetings.AttachAudio(ctx, func(value bool, revision uint64, complete snoofer.Completion) error {
				saves <- save{value, revision, complete}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(audio.Close)
			audio.Report(snoofer.MeetingMicPreference{Known: true, Applied: true, Revision: 7})
			result := dispatchOutcome(t, w, ctx, "discord.mute")
			now := time.Unix(1000, 0)
			w.handle(ctx, <-w.requests, now)
			var pending save
			select {
			case pending = <-saves:
			case <-time.After(time.Second):
				t.Fatal("shared mute did not reach the audio save boundary")
			}
			if !pending.value || pending.revision != 7 {
				t.Fatal("save lost captured audio revision", pending)
			}
			w.syncMeetings(now)
			assertNoOutcome(t, result)
			if pipe.Len() != 0 || w.sharedTarget(snoofer.MeetingMute).Value {
				t.Fatal("unsaved target reached Discord")
			}
			if !saved {
				pending.complete(snoofer.Outcome{Status: snoofer.Failed, Detail: "disk refused save"})
				expectOutcome(t, result, snoofer.Failed)
				w.syncMeetings(now)
				if pipe.Len() != 0 || w.sharedTarget(snoofer.MeetingMute).Value {
					t.Fatal("failed save reached Discord")
				}
				return
			}
			audio.Report(snoofer.MeetingMicPreference{Known: true, Muted: true, Applied: true, Revision: 8})
			pending.complete(snoofer.Outcome{Status: snoofer.Processed, Detail: "Saved"})
			expectOutcome(t, result, snoofer.Processed)
			w.syncMeetings(now)
			sent := readCommand(t, pipe)
			if sent.Cmd != "SET_VOICE_SETTINGS" || sent.Args.(map[string]any)["mute"] != true {
				t.Fatal(sent)
			}
			if w.voice.mute {
				t.Fatal("saved target invented Discord readback")
			}
		})
	}
}

func TestSharedDiscordLateReadbackClearsUncertaintyWithoutReplay(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	requestTarget(t, w, ctx, snoofer.MeetingMute, true)
	w.syncMeetings(now)
	sent := readCommand(t, pipe)
	w.syncMeetings(now.Add(4 * time.Second))
	if w.notes["discord.call-mute"] != "Not confirmed" {
		t.Fatal("missing uncertainty")
	}
	w.event(ctx, "VOICE_SETTINGS_UPDATE", raw(map[string]any{"mute": true, "deaf": false}), now.Add(5*time.Second))
	w.syncMeetings(now.Add(5 * time.Second))
	w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now.Add(5*time.Second))
	if w.notes["discord.call-mute"] != "" || w.sharedTarget(snoofer.MeetingMute).Status != "" || pipe.Len() != 0 {
		t.Fatal("late actual state did not resolve uncertainty without replay")
	}
}

func TestDiscordPreviewRetainsOnlyKnownJoinedActualValues(t *testing.T) {
	for _, state := range []string{"known", "unknown", "absent", "not-in-call"} {
		t.Run(state, func(t *testing.T) {
			w, _, _ := outcomeWorker(t)
			w.services.Live = false
			w.voice.mute, w.voice.video = true, true
			switch state {
			case "unknown":
				w.voice.settingsKnown, w.voice.videoKnown = false, false
			case "absent":
				w.conn, w.state = nil, stateClosed
			case "not-in-call":
				w.leftChannel()
			}
			want := "N/A"
			if state == "known" {
				want = "On"
			}
			for _, control := range w.controls(time.Unix(1000, 0)) {
				if control.ID == "discord.call-mute" || control.ID == "discord.call-camera" {
					if control.Value != want || !control.SurfaceOnly || len(control.Operations) != 0 {
						t.Fatal(control)
					}
					if state == "known" && control.Status != "Preview" {
						t.Fatal(control)
					}
				}
			}
		})
	}
}
