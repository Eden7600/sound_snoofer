package discord

import (
	"context"
	"errors"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestDiscordDeafenRequiresJoinedKnownCall(t *testing.T) {
	for _, mode := range []string{"off", "on", "no-call", "unknown-channel", "unknown-settings", "closed", "preview"} {
		t.Run(mode, func(t *testing.T) {
			w, _, ctx := outcomeWorker(t)
			want := "N/A"
			switch mode {
			case "off":
				want = "Off"
			case "on":
				w.voice.deaf, want = true, "On"
			case "no-call":
				w.leftChannel()
				w.notes["discord.deafen"] = "Failed"
			case "unknown-channel":
				w.voice.channelKnown = false
			case "unknown-settings":
				w.voice.settingsKnown = false
			case "closed":
				w.state = stateClosed
			case "preview":
				w.services.Live = false
			}
			w.publish(ctx, time.Unix(1000, 0))
			for _, c := range w.services.Controls.Snapshot() {
				if c.ID != "discord.deafen" {
					continue
				}
				if c.Value != want || c.Available != (want != "N/A") {
					t.Fatal(c)
				}
				if mode == "no-call" && c.Status != "Not in a call" {
					t.Fatal("old failure hid no-call state", c)
				}
				if !c.Available && w.services.Controls.Dispatch(ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}) == nil {
					t.Fatal("unavailable deafen accepted")
				}
				return
			}
			t.Fatal("missing deafen")
		})
	}
}

func focusedCommand(w *worker, ctx context.Context, action snoofer.MeetingAction) (snoofer.MeetingActionCommand, <-chan snoofer.Outcome) {
	w.reportMeetings(time.Unix(1000, 0))
	value, _ := w.meetingActionValue(action)
	results := make(chan snoofer.Outcome, 4)
	want := true
	if action != snoofer.MeetingLeave && value.Known {
		want = !value.Value
	}
	return snoofer.MeetingActionCommand{Context: ctx, Peer: "discord", Lifetime: w.meetingLifetime(), Action: action, Revision: value.Revision, Value: want, Complete: func(o snoofer.Outcome) { results <- o }}, results
}

func TestDiscordFocusedCapabilitiesFollowExactCall(t *testing.T) {
	w, _, _ := outcomeWorker(t)
	now := time.Unix(1000, 0)
	w.voice.screenKnown = false
	w.reportMeetings(now)
	before := w.meetingReported
	if !before.Deafen.Known || !before.Deafen.Available || before.Share.Known || !before.Share.Available || !before.Leave.Known || !before.Leave.Value || !before.Leave.Available {
		t.Fatal(before)
	}
	w.reportMeetings(now.Add(time.Second))
	if w.meetingReported != before {
		t.Fatal("heartbeat changed capabilities")
	}
	w.voice.deaf = true
	w.reportMeetings(now)
	if w.meetingReported.Deafen.Revision == before.Deafen.Revision || w.meetingReported.Share.Revision != before.Share.Revision {
		t.Fatal("action revisions not independent")
	}
	w.leftChannel()
	w.reportMeetings(now)
	if w.meetingReported.Joined || w.meetingReported.Deafen.Available || w.meetingReported.Share.Available || w.meetingReported.Leave.Available {
		t.Fatal(w.meetingReported)
	}
}

func TestDiscordFocusedActionsWaitForObservation(t *testing.T) {
	for _, action := range []snoofer.MeetingAction{snoofer.MeetingDeafen, snoofer.MeetingShare, snoofer.MeetingLeave} {
		t.Run(string(action), func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			now := time.Unix(1000, 0)
			if action == snoofer.MeetingShare {
				w.voice.screenKnown = false
			}
			command, result := focusedCommand(w, ctx, action)
			w.applyMeetingAction(command, now)
			sent := readCommand(t, pipe)
			w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now)
			assertNoOutcome(t, result)
			if action == snoofer.MeetingShare {
				local := dispatchOutcome(t, w, ctx, "discord.screenshare")
				w.handle(ctx, <-w.requests, now)
				expectOutcome(t, local, snoofer.Rejected)
				if pipe.Len() != 0 {
					t.Fatal("local share toggled while focused share was unconfirmed")
				}
			}
			switch action {
			case snoofer.MeetingDeafen:
				if sent.Cmd != "SET_VOICE_SETTINGS" || sent.Args.(map[string]any)["deaf"] != true {
					t.Fatal(sent)
				}
				w.event(ctx, "VOICE_SETTINGS_UPDATE", raw(map[string]any{"mute": true, "deaf": true}), now)
			case snoofer.MeetingShare:
				if sent.Cmd != "TOGGLE_SCREENSHARE" {
					t.Fatal(sent)
				}
				w.event(ctx, "SCREENSHARE_STATE_UPDATE", raw(map[string]any{"active": true}), now)
			case snoofer.MeetingLeave:
				if sent.Cmd != "SELECT_VOICE_CHANNEL" || sent.Args.(map[string]any)["channel_id"] != nil {
					t.Fatal(sent)
				}
				w.event(ctx, "VOICE_CHANNEL_SELECT", raw(map[string]any{"channel_id": nil}), now)
			}
			expectOutcome(t, result, snoofer.Processed)
			w.reportMeetings(now.Add(10 * time.Second))
			assertNoOutcome(t, result)
		})
	}
}

func TestDiscordFocusedActionsRejectObsoleteCommands(t *testing.T) {
	for _, action := range []snoofer.MeetingAction{snoofer.MeetingDeafen, snoofer.MeetingShare, snoofer.MeetingLeave} {
		for _, change := range []string{"cancel", "peer", "revision", "observed", "left", "rejoined", "preview"} {
			t.Run(string(action)+"/"+change, func(t *testing.T) {
				w, pipe, parent := outcomeWorker(t)
				ctx, cancel := context.WithCancel(parent)
				defer cancel()
				command, result := focusedCommand(w, ctx, action)
				switch change {
				case "cancel":
					cancel()
				case "peer":
					command.Peer = "other"
				case "revision":
					command.Revision++
				case "observed":
					switch action {
					case snoofer.MeetingDeafen:
						w.voice.deaf = true
					case snoofer.MeetingShare:
						w.voice.screen = true
					case snoofer.MeetingLeave:
						w.voice.channelKnown = false
					}
				case "left":
					w.leftChannel()
				case "rejoined":
					w.leftChannel()
					w.selectChannel("42", "Same")
				case "preview":
					w.services.Live = false
				}
				w.applyMeetingAction(command, time.Unix(1000, 0))
				expectOutcome(t, result, snoofer.Rejected)
				if pipe.Len() != 0 {
					t.Fatal("obsolete command reached Discord")
				}
			})
		}
	}
}

func TestDiscordFocusedActionsSettleUncertaintyWithoutReplay(t *testing.T) {
	for _, path := range []string{"timeout", "drop", "close", "failed", "write-error", "replacement"} {
		t.Run(path, func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			now := time.Unix(1000, 0)
			command, result := focusedCommand(w, ctx, snoofer.MeetingShare)
			if path == "write-error" {
				pipe.writeErr = errors.New("broken pipe")
			}
			w.applyMeetingAction(command, now)
			var sent message
			if path != "write-error" {
				sent = readCommand(t, pipe)
			}
			want := snoofer.Unknown
			switch path {
			case "timeout":
				w.reportMeetings(now.Add(verifyDeadline))
			case "drop":
				w.drop(errors.New("lost pipe"), now)
			case "close":
				w.closeConn()
			case "failed":
				w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce, Evt: "ERROR", Data: raw(rpcError{Message: "Denied"})}, now)
				want = snoofer.Failed
			case "replacement":
				w.selectChannel("99", "Other")
				w.reportMeetings(now)
			}
			expectOutcome(t, result, want)
			before := pipe.Len()
			w.reportMeetings(now.Add(20 * time.Second))
			w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now)
			assertNoOutcome(t, result)
			if len(w.meetingActions) != 0 || pipe.Len() != before {
				t.Fatal("uncertain command retained or replayed")
			}
		})
	}
}

func TestDiscordFocusedActionDrainsWithoutUniversalWork(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	var request snoofer.Request
	deadline := time.Now().Add(time.Second)
	for request.ID == "" && time.Now().Before(deadline) {
		for _, c := range w.services.Controls.Snapshot() {
			if c.ID == "meetings.leave" && c.Available {
				request = snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
			}
		}
		if request.ID == "" {
			time.Sleep(time.Millisecond)
		}
	}
	if request.ID == "" {
		t.Fatal("focused Leave unavailable")
	}
	results := make(chan snoofer.Outcome, 2)
	if err := w.services.Controls.DispatchWithResult(ctx, request, func(o snoofer.Outcome) { results <- o }); err != nil {
		t.Fatal(err)
	}
	w.syncMeetings(now)
	sent := readCommand(t, pipe)
	if sent.Cmd != "SELECT_VOICE_CHANNEL" {
		t.Fatal(sent)
	}
	w.event(ctx, "VOICE_CHANNEL_SELECT", raw(map[string]any{"channel_id": nil}), now)
	expectOutcome(t, results, snoofer.Processed)
}

func TestDiscordFocusedActionsCanTurnKnownStateOff(t *testing.T) {
	for _, action := range []snoofer.MeetingAction{snoofer.MeetingDeafen, snoofer.MeetingShare} {
		t.Run(string(action), func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			now := time.Unix(1000, 0)
			w.voice.deaf, w.voice.screen = true, true
			command, result := focusedCommand(w, ctx, action)
			if command.Value {
				t.Fatal("captured wrong target")
			}
			w.applyMeetingAction(command, now)
			sent := readCommand(t, pipe)
			if action == snoofer.MeetingDeafen && sent.Args.(map[string]any)["deaf"] != false {
				t.Fatal(sent)
			}
			w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now)
			assertNoOutcome(t, result)
			if action == snoofer.MeetingDeafen {
				w.event(ctx, "VOICE_SETTINGS_UPDATE", raw(map[string]any{"mute": false, "deaf": false}), now)
			} else {
				w.event(ctx, "SCREENSHARE_STATE_UPDATE", raw(map[string]any{"active": false}), now)
			}
			expectOutcome(t, result, snoofer.Processed)
		})
	}
}
