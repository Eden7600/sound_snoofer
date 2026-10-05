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
)

type monitorWorkerClient struct {
	Client
	held           *atomic.Bool
	active         bool
	transitions    chan bool
	ownershipError atomic.Bool
}

func (b *monitorWorkerClient) SetMonitoring(enable bool) error {
	if enable && !b.held.Load() {
		b.ownershipError.Store(true)
		return fmt.Errorf("unowned callback")
	}
	if enable != b.active {
		b.active = enable
		b.transitions <- enable
	}
	return nil
}
func (b *monitorWorkerClient) Snapshot() (model.Snapshot, error) {
	_, s := callbackFixture()
	// Missing hardware never authorizes restart, even with no buffers.
	s.Devices[1].Available = false
	if !b.active {
		s.Callback = nil
	}
	return s, nil
}
func (b *monitorWorkerClient) Set(string, model.Device) error {
	return fmt.Errorf("unexpected device write")
}
func (b *monitorWorkerClient) Close() error { return b.SetMonitoring(false) }
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
