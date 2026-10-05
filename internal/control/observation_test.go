package control

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type observationClient struct{ fail bool }

func (f *observationClient) Snapshot() (model.Snapshot, error) {
	if f.fail {
		return model.Snapshot{}, errors.New("read failed")
	}
	return model.Snapshot{Edition: 2}, nil
}
func (f *observationClient) Set(string, model.Device) error { return errors.New("write failed") }
func (f *observationClient) Close() error                   { return nil }

func TestWorkerRoutingErrorDoesNotDisconnect(t *testing.T) {
	// A Potato-only profile on Banana creates a plan error, but the API is connected.
	cfg := config.Config{Studio: &config.Studio{Voice: &config.Voice{}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	states := make(chan State, 1)
	done := make(chan struct{})
	go Work(ctx, cfg, filepath.Join(t.TempDir(), "config"), "", false, Dependencies{Open: func(string) (Client, error) { return &observationClient{}, nil }}, make(chan Action), states, done)
	select {
	case s := <-states:
		if !s.Connected || s.Error == "" || s.Plan != nil {
			t.Fatalf("routing diagnostic misrepresented: %+v", s)
		}
	case <-time.After(time.Second):
		t.Fatal("no observation")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}
func TestObservationFailureRecovery(t *testing.T) {
	f := &observationClient{}
	o := &observed{Client: f}
	if _, err := o.Snapshot(); err != nil || o.readError != nil {
		t.Fatal("initial read")
	}
	f.fail = true
	if _, err := o.ParameterSnapshot(); err == nil || o.readError == nil {
		t.Fatal("failure hidden")
	}
	f.fail = false
	if _, err := o.ParameterSnapshot(); err != nil || o.readError != nil {
		t.Fatal("failure did not clear")
	}
}
