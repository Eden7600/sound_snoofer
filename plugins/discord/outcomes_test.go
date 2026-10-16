package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

type outcomePipe struct {
	bytes.Buffer
	writeErr error
}

func (p *outcomePipe) Write(b []byte) (int, error) {
	if p.writeErr != nil {
		return 0, p.writeErr
	}
	return p.Buffer.Write(b)
}

func (p *outcomePipe) Close() error             { return nil }
func (p *outcomePipe) Read([]byte) (int, error) { return 0, io.EOF }

func outcomeWorker(t *testing.T) (*worker, *outcomePipe, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	w := newWorker(snoofer.Services{Controls: snoofer.NewControls(), Live: true}, nil, configured, nil, http.DefaultClient, "")
	pipe := &outcomePipe{}
	w.conn, w.generation, w.state = pipe, 1, stateReady
	w.voice = voice{settingsKnown: true, channelKnown: true, channelID: "42", videoKnown: true, screenKnown: true}
	attachMeetingFixture(t, w, ctx, nil)
	w.reportMeetings(time.Unix(1000, 0))
	return w, pipe, ctx
}

func dispatchOutcome(t *testing.T, w *worker, ctx context.Context, id string) <-chan snoofer.Outcome {
	t.Helper()
	w.publish(ctx, time.Unix(1000, 0))
	result := make(chan snoofer.Outcome, 4)
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID == id {
			if err := w.services.Controls.DispatchWithResult(ctx, snoofer.Request{ID: id, Revision: c.Revision, Operation: "press"}, func(o snoofer.Outcome) { result <- o }); err != nil {
				t.Fatal(err)
			}
			return result
		}
	}
	t.Fatal("missing", id)
	return result
}

func TestDiscordOutcomeWaitsForNonceResponse(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	result := dispatchOutcome(t, w, ctx, "discord.deafen")
	select {
	case r := <-result:
		t.Fatalf("completed before worker/RPC: %+v", r)
	default:
	}
	w.handle(ctx, <-w.requests, time.Unix(1000, 0))
	f, err := readFrame(&pipe.Buffer)
	if err != nil {
		t.Fatal(err)
	}
	var sent message
	if err := json.Unmarshal(f.payload, &sent); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-result:
		t.Fatalf("completed before response: %+v", r)
	default:
	}
	w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce, Data: raw(map[string]any{})}, time.Unix(1000, 0))
	select {
	case r := <-result:
		if r.Status != snoofer.Processed {
			t.Fatal(r)
		}
	default:
		t.Fatal("response did not complete request")
	}
	if w.observed("discord.deafen") != "Off" || w.pending["discord.deafen"].want != "On" {
		t.Fatal("RPC response was treated as observed state")
	}
}

func TestDiscordQueuedCallCannotFollowChannelReplacement(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	result := dispatchOutcome(t, w, ctx, "discord.leave")
	w.voice.channelID = "99" // Worker changed before its next publication.
	w.handle(ctx, <-w.requests, time.Unix(1000, 0))
	if pipe.Len() != 0 {
		t.Fatal("old Leave reached replacement call")
	}
	select {
	case r := <-result:
		if r.Status != snoofer.Rejected {
			t.Fatal(r)
		}
	default:
		t.Fatal("stale action did not complete")
	}
}

func assertNoOutcome(t *testing.T, results <-chan snoofer.Outcome) {
	t.Helper()
	select {
	case got := <-results:
		t.Fatalf("unexpected outcome: %+v", got)
	default:
	}
}

func expectOutcome(t *testing.T, results <-chan snoofer.Outcome, status snoofer.OutcomeStatus) snoofer.Outcome {
	t.Helper()
	select {
	case got := <-results:
		if got.Status != status {
			t.Fatalf("got %+v; want %s", got, status)
		}
		return got
	case <-time.After(time.Second):
		t.Fatalf("no %s outcome", status)
	}
	return snoofer.Outcome{}
}

func readCommand(t *testing.T, pipe *outcomePipe) message {
	t.Helper()
	f, err := readFrame(&pipe.Buffer)
	if err != nil {
		t.Fatal(err)
	}
	var sent message
	if err := json.Unmarshal(f.payload, &sent); err != nil {
		t.Fatal(err)
	}
	return sent
}

func TestDiscordNonceOutcomesAreSeparateAndOnceOnly(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	now := time.Unix(1000, 0)
	deafen := dispatchOutcome(t, w, ctx, "discord.deafen")
	leave := dispatchOutcome(t, w, ctx, "discord.leave")
	w.handle(ctx, <-w.requests, now)
	first := readCommand(t, pipe)
	w.handle(ctx, <-w.requests, now)
	second := readCommand(t, pipe)
	assertNoOutcome(t, deafen)
	assertNoOutcome(t, leave)
	w.dispatch(ctx, message{Cmd: second.Cmd, Nonce: second.Nonce, Data: raw(map[string]any{})}, now)
	expectOutcome(t, leave, snoofer.Processed)
	assertNoOutcome(t, deafen)
	failed := message{Cmd: first.Cmd, Nonce: first.Nonce, Evt: "ERROR", Data: raw(rpcError{Code: 4010, Message: "Not permitted"})}
	w.dispatch(ctx, failed, now)
	if got := expectOutcome(t, deafen, snoofer.Failed); !strings.Contains(got.Detail, "Not permitted") {
		t.Fatal(got)
	}
	w.dispatch(ctx, failed, now)
	w.dispatch(ctx, message{Cmd: second.Cmd, Nonce: second.Nonce}, now)
	assertNoOutcome(t, deafen)
	assertNoOutcome(t, leave)
}

func TestDiscordTimeoutDropAndWriteErrorAreUnknownWithoutReplay(t *testing.T) {
	for _, path := range []string{"timeout", "drop", "write-error", "close"} {
		t.Run(path, func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			now := time.Unix(1000, 0)
			result := dispatchOutcome(t, w, ctx, "discord.deafen")
			if path == "write-error" {
				pipe.writeErr = errors.New("pipe broke during write")
			}
			w.handle(ctx, <-w.requests, now)
			nonce := strconv.Itoa(w.nonce)
			switch path {
			case "timeout":
				w.step(ctx, now.Add(requestTimeout+time.Second))
			case "drop":
				w.drop(errors.New("lost pipe"), now)
			case "close":
				w.closeConn()
			}
			expectOutcome(t, result, snoofer.Unknown)
			before := pipe.Len()
			w.dispatch(ctx, message{Cmd: "SET_VOICE_SETTINGS", Nonce: nonce}, now)
			assertNoOutcome(t, result)
			if len(w.calls) != 0 || pipe.Len() != before {
				t.Fatal("uncertain command replayed or retained", len(w.calls))
			}
			if path == "write-error" && len(w.pending) != 0 {
				t.Fatal("failed send created an observed-state wait")
			}
		})
	}
}

func TestDiscordCapturedConnectionAndChannelLifetimes(t *testing.T) {
	for _, id := range []string{"discord.deafen", "discord.screenshare", "discord.leave"} {
		t.Run(id+"/connection", func(t *testing.T) {
			w, pipe, ctx := outcomeWorker(t)
			result := dispatchOutcome(t, w, ctx, id)
			w.generation++
			w.handle(ctx, <-w.requests, time.Unix(1000, 0))
			expectOutcome(t, result, snoofer.Rejected)
			if pipe.Len() != 0 {
				t.Fatal("stale connection caused an effect")
			}
		})
	}
	for _, id := range []string{"discord.deafen", "discord.screenshare", "discord.leave"} {
		for _, published := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/channel-ABA/published=%t", id, published), func(t *testing.T) {
				w, pipe, ctx := outcomeWorker(t)
				now := time.Unix(1000, 0)
				var result <-chan snoofer.Outcome
				var old snoofer.Request
				if published {
					w.publish(ctx, now)
					old = publishedRequest(t, w, id)
				} else {
					result = dispatchOutcome(t, w, ctx, id)
				}
				w.leftChannel()
				w.selectChannel("42", "Same channel")
				// Even equal-looking values after rejoining do not restore the old lifetime.
				w.voice.videoKnown, w.voice.screenKnown = true, true
				if published {
					w.publish(ctx, now)
					replies := make(chan snoofer.Outcome, 2)
					result = replies
					err := w.services.Controls.DispatchWithResult(ctx, old, func(o snoofer.Outcome) { replies <- o })
					if err == nil {
						t.Fatal("old drawn request accepted after channel ABA")
					}
				} else {
					w.handle(ctx, <-w.requests, now)
				}
				expectOutcome(t, result, snoofer.Rejected)
				if pipe.Len() != 0 {
					t.Fatal("old channel lifetime wrote a command")
				}
			})
		}
	}
}

func publishedRequest(t *testing.T, w *worker, id string) snoofer.Request {
	t.Helper()
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID == id {
			return snoofer.Request{ID: id, Operation: "press", Revision: c.Revision}
		}
	}
	t.Fatal("missing", id)
	return snoofer.Request{}
}

func TestDiscordDeafenUsesCapturedDesiredValue(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	result := dispatchOutcome(t, w, ctx, "discord.deafen")
	w.voice.deaf = true
	w.handle(ctx, <-w.requests, time.Unix(1000, 0))
	sent := readCommand(t, pipe)
	args := sent.Args.(map[string]any)
	if args["deaf"] != true {
		t.Fatal("inverted newer deafen state", args)
	}
	w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, time.Unix(1000, 0))
	expectOutcome(t, result, snoofer.Processed)
}

func TestDiscordToggleOnlyRPCsRetainKnownAndUnknownState(t *testing.T) {
	for _, id := range []string{"discord.screenshare"} {
		for _, mode := range []string{"known", "unknown", "changed-value", "became-known", "became-unknown"} {
			t.Run(id+"/"+mode, func(t *testing.T) {
				w, pipe, ctx := outcomeWorker(t)
				known, value := &w.voice.videoKnown, &w.voice.video
				wantCmd := "TOGGLE_VIDEO"
				if id == "discord.screenshare" {
					known, value = &w.voice.screenKnown, &w.voice.screen
					wantCmd = "TOGGLE_SCREENSHARE"
				}
				*known = mode != "unknown" && mode != "became-known"
				result := dispatchOutcome(t, w, ctx, id)
				switch mode {
				case "changed-value":
					*value = true
				case "became-known":
					*known = true
				case "became-unknown":
					*known = false
				}
				w.handle(ctx, <-w.requests, time.Unix(1000, 0))
				if mode == "known" || mode == "unknown" {
					sent := readCommand(t, pipe)
					if sent.Cmd != wantCmd {
						t.Fatal(sent)
					}
					w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, time.Unix(1000, 0))
					expectOutcome(t, result, snoofer.Processed)
					if mode == "unknown" {
						if _, ok := w.pending[id]; ok || w.observed(id) != "" {
							t.Fatal("unknown toggle invented an observed target")
						}
					}
				} else {
					expectOutcome(t, result, snoofer.Rejected)
					if pipe.Len() != 0 {
						t.Fatal("changed toggle state reached Discord")
					}
				}
			})
		}
	}
}

func TestDiscordConnectCompletesSubmittedAtWorkerAcceptance(t *testing.T) {
	w, pipe, ctx := outcomeWorker(t)
	w.state = stateUnauthorized
	result := dispatchOutcome(t, w, ctx, "discord.connect")
	assertNoOutcome(t, result)
	w.handle(ctx, <-w.requests, time.Unix(1000, 0))
	expectOutcome(t, result, snoofer.Submitted)
	sent := readCommand(t, pipe)
	if sent.Cmd != "AUTHORIZE" {
		t.Fatal(sent)
	}
	w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce, Evt: "ERROR", Data: raw(rpcError{Message: "Denied"})}, time.Unix(1000, 0))
	assertNoOutcome(t, result)
	if w.state != stateUnauthorized || w.notes["discord.connect"] != "Failed" {
		t.Fatal("later auth failure was lost")
	}
}

func TestDiscordFullQueueAndCancelledLifetimeSettleOnce(t *testing.T) {
	w, pipe, parent := outcomeWorker(t)
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var accepted []<-chan snoofer.Outcome
	for range cap(w.requests) {
		accepted = append(accepted, dispatchOutcome(t, w, ctx, "discord.deafen"))
	}
	rejected := make(chan snoofer.Outcome, 2)
	err := w.services.Controls.DispatchWithResult(ctx, publishedRequest(t, w, "discord.deafen"), func(o snoofer.Outcome) { rejected <- o })
	if err == nil {
		t.Fatal("full queue accepted")
	}
	expectOutcome(t, rejected, snoofer.Rejected)
	cancel()
	for _, r := range accepted {
		expectOutcome(t, r, snoofer.Unknown)
	}
	for range cap(w.requests) {
		w.handle(ctx, <-w.requests, time.Unix(1000, 0))
	}
	for _, r := range accepted {
		assertNoOutcome(t, r)
	}
	assertNoOutcome(t, rejected)
	if pipe.Len() != 0 {
		t.Fatal("cancelled queued command wrote to Discord")
	}
}

func TestDiscordStopAfterSendIgnoresLateReply(t *testing.T) {
	w, pipe, parent := outcomeWorker(t)
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	result := dispatchOutcome(t, w, ctx, "discord.deafen")
	w.handle(ctx, <-w.requests, time.Unix(1000, 0))
	sent := readCommand(t, pipe)
	cancel()
	expectOutcome(t, result, snoofer.Unknown)
	w.closeConn()
	w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, time.Unix(1000, 0))
	assertNoOutcome(t, result)
	if len(w.calls) != 0 {
		t.Fatal("stop retained RPC claims")
	}
}

func TestDiscordToggleCannotUndoAnUnobservedRequest(t *testing.T) {
	for _, id := range []string{"discord.screenshare"} {
		for _, phase := range []string{"queued", "replied", "observed-before-reply", "unknown"} {
			t.Run(id+"/"+phase, func(t *testing.T) {
				w, pipe, ctx := outcomeWorker(t)
				now := time.Unix(1000, 0)
				known, value := &w.voice.videoKnown, &w.voice.video
				if id == "discord.screenshare" {
					known, value = &w.voice.screenKnown, &w.voice.screen
				}
				if phase == "unknown" {
					*known = false
				}
				first := dispatchOutcome(t, w, ctx, id)
				var second <-chan snoofer.Outcome
				if phase != "observed-before-reply" {
					second = dispatchOutcome(t, w, ctx, id) // Both requests captured before worker processing.
				}
				w.handle(ctx, <-w.requests, now)
				sent := readCommand(t, pipe)
				if phase == "replied" {
					w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now)
					expectOutcome(t, first, snoofer.Processed)
					if w.observed(id) != "Off" {
						t.Fatal("response invented readback")
					}
				}
				if phase == "observed-before-reply" {
					*value = true
					w.step(ctx, now)
					second = dispatchOutcome(t, w, ctx, id)
				}
				w.handle(ctx, <-w.requests, now)
				if pipe.Len() != 0 {
					t.Fatal("second toggle could undo the first before its result and observation settled")
				}
				expectOutcome(t, second, snoofer.Rejected)
				if phase != "replied" {
					w.dispatch(ctx, message{Cmd: sent.Cmd, Nonce: sent.Nonce}, now)
					expectOutcome(t, first, snoofer.Processed)
				}
				if phase != "unknown" {
					*value = true
					w.step(ctx, now)
				}
				third := dispatchOutcome(t, w, ctx, id)
				w.handle(ctx, <-w.requests, now)
				next := readCommand(t, pipe)
				w.dispatch(ctx, message{Cmd: next.Cmd, Nonce: next.Nonce}, now)
				expectOutcome(t, third, snoofer.Processed)
			})
		}
	}
}
