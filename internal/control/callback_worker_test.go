package control

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/voicemeeter"
)

type monitorWorkerClient struct {
	Client
	held           *atomic.Bool
	active         bool
	transitions    chan bool
	ownershipError atomic.Bool
	present        bool // Keep A1 hardware present so a stall can qualify.
	insert         *voicemeeter.InsertHook
	inserts        chan *voicemeeter.InsertHook // Optional: receives each insert change.
}

func (b *monitorWorkerClient) SetCallback(enable bool, insert *voicemeeter.InsertHook) error {
	if (enable || insert != nil) && !b.held.Load() {
		b.ownershipError.Store(true)
		return fmt.Errorf("unowned callback")
	}
	if enable != b.active {
		b.active = enable
		b.transitions <- enable
	}
	if b.inserts != nil && (insert == nil) != (b.insert == nil) {
		b.inserts <- insert
	}
	b.insert = insert
	return nil
}
func (b *monitorWorkerClient) Snapshot() (model.Snapshot, error) {
	_, s := callbackFixture()
	// Missing hardware never authorizes restart, even with no buffers.
	s.Devices[1].Available = b.present
	if !b.active {
		s.Callback = nil
	}
	return s, nil
}
func (b *monitorWorkerClient) Set(string, model.Device) error {
	return fmt.Errorf("unexpected device write")
}
func (b *monitorWorkerClient) Close() error { return b.SetCallback(false, nil) }
func TestWorkerMonitorOnlyWithinLiveOwnership(t *testing.T) {
	c, _ := callbackFixture()
	c.Studio.Voice = &config.Voice{Source: "off", Mode: "direct", Monitor: "off"}
	c.Intent = &config.Intent{Version: 1, Source: "off", Mode: "direct", Monitor: "off", AutoRecover: true}
	var held atomic.Bool
	b := &monitorWorkerClient{held: &held, transitions: make(chan bool, 8)}
	ctx, cancel := context.WithCancel(context.Background())
	actions := make(chan Action, 4)
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{Open: func(string) (Client, error) { return b, nil }, Acquire: func() (func(), error) {
		held.Store(true)
		return func() {
			if b.active {
				b.ownershipError.Store(true)
			}
			held.Store(false)
		}, nil
	}}
	go Work(ctx, c, filepath.Join(t.TempDir(), "config"), "", false, deps, actions, states, done)
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("worker did not stop")
		}
	}()
	select {
	case <-states:
	case <-time.After(3 * time.Second):
		t.Fatal("no preview state")
	}
	select {
	case <-b.transitions:
		t.Fatal("preview registered monitor")
	default:
	}
	actions <- ToggleLive
	select {
	case on := <-b.transitions:
		if !on {
			t.Fatal("expected enable")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no live monitor")
	}
	actions <- ToggleLive
	select {
	case on := <-b.transitions:
		if on {
			t.Fatal("expected disable")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no monitor cleanup")
	}
	cancel()
	<-done
	if held.Load() || b.ownershipError.Load() {
		t.Fatal("monitor ownership violated")
	}
}

func TestWorkerInsertOnlyWhileLive(t *testing.T) {
	c, _ := callbackFixture()
	c.Studio.Voice = &config.Voice{Source: "off", Mode: "direct", Monitor: "off"}
	c.Intent = &config.Intent{Version: 1, Source: "off", Mode: "direct", Monitor: "off"}
	var held atomic.Bool
	b := &monitorWorkerClient{held: &held, transitions: make(chan bool, 8), inserts: make(chan *voicemeeter.InsertHook, 8)}
	var wanted atomic.Pointer[voicemeeter.InsertHook]
	var generation atomic.Uint64
	ctx, cancel := context.WithCancel(context.Background())
	actions := make(chan Action, 4)
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{
		Open: func(string) (Client, error) { return b, nil },
		Acquire: func() (func(), error) {
			held.Store(true)
			return func() { held.Store(false) }, nil
		},
		Insert: func() (*voicemeeter.InsertHook, uint64) { return wanted.Load(), generation.Load() },
	}
	go Work(ctx, c, filepath.Join(t.TempDir(), "config"), "", true, deps, actions, states, done)
	defer func() {
		cancel()
		<-done
	}()
	receive := func(what string) *voicemeeter.InsertHook {
		select {
		case insert := <-b.inserts:
			return insert
		case <-time.After(3 * time.Second):
			t.Fatal(what)
			return nil
		}
	}
	// published waits for a state reporting the insert at a generation.
	published := func(want *voicemeeter.InsertHook, at uint64) {
		deadline := time.After(3 * time.Second)
		for {
			select {
			case s := <-states:
				if s.InsertGeneration == at && (s.Insert == nil) == (want == nil) && (want == nil || *s.Insert == *want) {
					return
				}
			case <-deadline:
				t.Fatalf("no state with insert %v at generation %d", want, at)
			}
		}
	}
	hook := &voicemeeter.InsertHook{Input: 1, Output: 2, Context: 3}
	wanted.Store(hook)
	generation.Store(1)
	actions <- Refresh
	if got := receive("insert not installed"); got == nil || *got != *hook {
		t.Fatal(got)
	}
	published(hook, 1)
	wanted.Store(nil)
	generation.Store(2)
	actions <- Refresh
	if got := receive("insert not removed"); got != nil {
		t.Fatal(got)
	}
	published(nil, 2)
	wanted.Store(hook)
	generation.Store(3)
	actions <- Refresh
	receive("insert not reinstalled")
	actions <- ToggleLive
	if got := receive("insert not removed on leaving live"); got != nil {
		t.Fatal(got)
	}
	published(nil, 3)
	if b.ownershipError.Load() {
		t.Fatal("callback changed without ownership")
	}
}

func TestWorkerDetectsStallWithoutAutoRecover(t *testing.T) {
	c, _ := callbackFixture()
	c.Studio.Voice = &config.Voice{Source: "off", Mode: "direct", Monitor: "off"}
	c.Intent = &config.Intent{Version: 1, Source: "off", Mode: "direct", Monitor: "off"}
	var held atomic.Bool
	b := &monitorWorkerClient{held: &held, transitions: make(chan bool, 8), present: true}
	ctx, cancel := context.WithCancel(context.Background())
	actions := make(chan Action, 4)
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{Open: func(string) (Client, error) { return b, nil }, Acquire: func() (func(), error) {
		held.Store(true)
		return func() { held.Store(false) }, nil
	}}
	go Work(ctx, c, filepath.Join(t.TempDir(), "config"), "", true, deps, actions, states, done)
	defer func() {
		cancel()
		<-done
	}()
	select {
	case on := <-b.transitions:
		if !on {
			t.Fatal("expected live monitor without Auto-recover")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("monitor not registered without Auto-recover")
	}
	deadline := time.After(15 * time.Second)
	for {
		select {
		case s := <-states:
			if s.Stalled {
				if s.Health != StallMessage+" · restart required" || s.RecoveryPending {
					t.Fatalf("alert-only stall reported %q (pending %v)", s.Health, s.RecoveryPending)
				}
				return
			}
		case <-deadline:
			t.Fatal("stall not reported without Auto-recover")
		}
	}
}
