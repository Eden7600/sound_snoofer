package control

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/windowsaudio"
)

func TestMuteTargetsAndCurrentHealth(t *testing.T) {
	s := State{Connected: true, Intent: &config.Intent{Enabled: true, Source: "desk", MicMuted: true}, Plan: &routing.Plan{Topology: &routing.Topology{Voice: &routing.VoiceStatus{Strip: 3}, MicStrips: []int{0, 1, 2, 3, 6}}}, Snapshot: model.Snapshot{Numbers: map[string]float32{"Strip[3].Mute": 1, "Strip[6].Mute": 0}}}
	if m, _ := ObserveMute(s, "mic-mute"); !m.Pending || !m.Known {
		t.Fatal(m)
	}
	if !DependsOn(s, "monitor", routing.Operation{Parameter: "Strip[3].A2"}) {
		t.Fatal("VR monitor not recognized")
	}
	r := recovery{status: "Engine responding; audio continuity unverified"}
	if got := r.observe(model.Snapshot{Assignments: map[string]string{"A1": "Volt"}, Numbers: map[string]float32{"Bus[0].device.sr": 0}}, time.Now()); !strings.Contains(got, "unavailable") {
		t.Fatal("old outcome hid current health", got)
	}
}
func TestWorkerDisablesDefaultsOnProfileRemoval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	requestsSeen := make(chan windowsaudio.Request, 32)
	start := func(ctx context.Context) (chan windowsaudio.Request, chan windowsaudio.Result, <-chan struct{}) {
		requests := make(chan windowsaudio.Request, 1)
		results := make(chan windowsaudio.Result, 1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer close(results)
			for {
				select {
				case <-ctx.Done():
					return
				case r := <-requests:
					requestsSeen <- r
					if r.Ack != nil {
						close(r.Ack)
					}
				}
			}
		}()
		return requests, results, done
	}
	cfg := config.Config{Studio: &config.Studio{Voice: &config.Voice{}}, Intent: &config.Intent{ProtectDefaults: true}}
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{StartDefaults: start, Acquire: func() (func(), error) { return func() {}, nil }, Open: func(string) (Client, error) { return nil, fmt.Errorf("offline") }, Load: func(string) (config.Config, error) { return config.Config{}, nil }}
	go Work(ctx, cfg, filepath.Join(t.TempDir(), "config"), "", true, deps, actions, states, done)
	next := func() State {
		select {
		case s := <-states:
			return s
		case <-time.After(time.Second):
			t.Fatal("worker did not publish")
			return State{}
		}
	}
	next()
	select {
	case r := <-requestsSeen:
		if !r.Enabled || !r.Live {
			t.Fatal(r)
		}
	case <-time.After(time.Second):
		t.Fatal("no policy")
	}
	actions <- Reload
	next()
	select {
	case r := <-requestsSeen:
		if r.Enabled || r.Live {
			t.Fatal("profile removal left policy enabled")
		}
	case <-time.After(time.Second):
		t.Fatal("no revocation")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown leaked")
	}
}

type consistencyClient struct {
	Client
	snapshot model.Snapshot
	err      error
	restarts int
}

func (b *consistencyClient) Snapshot() (model.Snapshot, error) { return b.snapshot, b.err }
func (b *consistencyClient) Close() error                      { return nil }
func (b *consistencyClient) Recorder() (model.RecorderSnapshot, error) {
	return model.RecorderSnapshot{}, fmt.Errorf("unknown transport")
}
func (b *consistencyClient) RestartEngine() error { b.restarts++; return nil }
func TestObservationFailureDoesNotRefreshAge(t *testing.T) {
	b := &consistencyClient{snapshot: model.Snapshot{Edition: 3}}
	o := &observed{Client: b}
	o.Snapshot()
	first := o.observedAt
	b.err = fmt.Errorf("read failed")
	o.ParameterSnapshot()
	if first.IsZero() || o.observedAt != first || o.snapshot.Edition != 3 {
		t.Fatal("failed read replaced last successful observation")
	}
	b.err = nil
	o.Snapshot()
	if o.observedAt.Before(first) {
		t.Fatal("successful read did not advance")
	}
}
func TestWorkerRejectsStaleRestartConfirmation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &consistencyClient{snapshot: model.Snapshot{Edition: 3, Assignments: map[string]string{}}}
	start := func(context.Context) (chan windowsaudio.Request, chan windowsaudio.Result, <-chan struct{}) {
		done := make(chan struct{})
		close(done)
		return make(chan windowsaudio.Request, 1), make(chan windowsaudio.Result, 1), done
	}
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{StartDefaults: start, Acquire: func() (func(), error) { return func() {}, nil }, Open: func(string) (Client, error) { return b, nil }}
	go Work(ctx, config.Config{}, filepath.Join(t.TempDir(), "config"), "", true, deps, actions, states, done)
	next := func() State {
		select {
		case s := <-states:
			return s
		case <-time.After(time.Second):
			t.Fatal("no state")
			return State{}
		}
	}
	s := next()
	id := uint64(0)
	send := func(a Action) State {
		id++
		a.Origin = "test"
		a.ID = id
		actions <- a
		for {
			v := next()
			if v.Acks["test"].ID == id {
				return v
			}
		}
	}
	s = send(Action{Kind: Restart, Revision: s.Revision - 1})
	if !strings.Contains(s.Notice, "Stale") || b.restarts != 0 {
		t.Fatal("stale restart submitted", s.Revision, s.Notice, s.Error, b.restarts)
	}
	s = send(Action{Kind: Restart, Revision: s.Revision})
	if !s.RestartConfirmation {
		t.Fatal("unknown transport not confirmed")
	}
	old := s.Revision
	s = send(ToggleLive)
	s = send(Action{Kind: Restart, Revision: old, Confirm: true})
	if s.RestartConfirmation || b.restarts != 0 || !strings.Contains(s.Notice, "Stale") {
		t.Fatal("old confirmation survived mode change")
	}
	cancel()
	<-done
}
func TestPreviewWaitsForDefaultsRevocation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nativeDone := make(chan struct{})
	start := func(context.Context) (chan windowsaudio.Request, chan windowsaudio.Result, <-chan struct{}) {
		return make(chan windowsaudio.Request, 1), make(chan windowsaudio.Result, 1), nativeDone
	}
	released := false
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{StartDefaults: start, Acquire: func() (func(), error) { return func() { released = true }, nil }, Open: func(string) (Client, error) { return nil, fmt.Errorf("offline") }}
	go Work(ctx, config.Config{}, filepath.Join(t.TempDir(), "config"), "", true, deps, actions, states, done)
	<-states
	actions <- ToggleLive
	select {
	case s := <-states:
		if !s.Live || released || !strings.Contains(s.Notice, "shutdown unverified") {
			t.Fatal("released ownership before revocation", s.Notice)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("revocation wait unbounded")
	}
	close(nativeDone)
	cancel()
	<-done
	if !released {
		t.Fatal("completed worker did not release")
	}
}

func TestRestartSubmissionDoesNotBecomeRouteApplied(t *testing.T) {
	s := State{Live: true, Connected: true, RecoveryPending: true, Plan: &routing.Plan{}}
	s.SetNotice("Audio engine restart submitted", NoticePending, time.Now())
	s.ResolveNotice(time.Now())
	if s.NoticeKind != NoticePending || s.Notice != "Audio engine restart submitted" {
		t.Fatal(s.Notice)
	}
}
