package control

import (
	"context"
	"errors"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakeClient struct{ closed, writes atomic.Int32 }

func (f *fakeClient) Snapshot() (model.Snapshot, error) {
	return model.Snapshot{Edition: 2, Assignments: map[string]string{"A1": "speakers"}, Devices: []model.Device{{Name: "speakers", Driver: "wdm", Direction: "output", Available: true}}}, nil
}
func (f *fakeClient) Set(string, model.Device) error { f.writes.Add(1); return nil }
func (f *fakeClient) Close() error                   { f.closed.Add(1); return nil }
func cfg(t *testing.T) config.Config {
	t.Helper()
	c, e := config.Decode([]byte(`{"version":1,"routes":[{"target":"A1","candidates":[{"driver":"wdm","pattern":"speakers"}]}]}`))
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func nextState(t *testing.T, ch <-chan State) State {
	t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not publish")
		return State{}
	}
}
func TestWorkerControlsCleanupAndReloadFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &fakeClient{}
	var held atomic.Int32
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Acquire: func() (func(), error) { held.Add(1); return func() { held.Add(-1) }, nil }, Load: func(string) (config.Config, error) { return config.Config{}, errors.New("bad regex") }}
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	go Work(ctx, cfg(t), "config", "", false, deps, actions, states, done)
	s := nextState(t, states)
	if s.Live || !s.Connected || client.writes.Load() != 0 {
		t.Fatal(s)
	}
	actions <- ToggleLive
	s = nextState(t, states)
	if !s.Live || held.Load() != 1 {
		t.Fatal(s)
	}
	actions <- Reload
	s = nextState(t, states)
	if !strings.Contains(s.Notice, "bad regex") || s.Plan == nil {
		t.Fatal("reload did not preserve valid config", s)
	}
	actions <- ToggleLive
	s = nextState(t, states)
	if s.Live || held.Load() != 0 {
		t.Fatal(s)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	if client.closed.Load() != 1 || held.Load() != 0 {
		t.Fatal("cleanup failed")
	}
}
func TestConnectionFailureStaysAliveAndRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempts := 0
	client := &fakeClient{}
	deps := Dependencies{Open: func(string) (Client, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("DLL unavailable")
		}
		return client, nil
	}, Load: config.Load}
	actions := make(chan Action, 1)
	states := make(chan State, 1)
	done := make(chan struct{})
	go Work(ctx, cfg(t), "config", "", false, deps, actions, states, done)
	s := nextState(t, states)
	if !strings.Contains(s.Error, "DLL unavailable") {
		t.Fatal(s)
	}
	actions <- Refresh
	s = nextState(t, states)
	if !s.Connected {
		t.Fatal(s)
	}
	cancel()
	<-done
}
func TestWriterConflictRemainsDry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &fakeClient{}
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Acquire: func() (func(), error) { return nil, errors.New("writer active") }, Load: config.Load}
	states := make(chan State, 1)
	done := make(chan struct{})
	go Work(ctx, cfg(t), "config", "", true, deps, make(chan Action), states, done)
	s := nextState(t, states)
	if s.Live || !strings.Contains(s.Notice, "writer active") {
		t.Fatal(s)
	}
	cancel()
	<-done
}
