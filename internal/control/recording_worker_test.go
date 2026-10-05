package control

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type recorderClient struct {
	*ruleClient
	mu        sync.Mutex
	transport string
	calls     int
}

func (c *recorderClient) Recorder() (model.RecorderSnapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := model.RecorderSnapshot{Values: map[string]float32{}}
	if c.transport == "unknown" {
		return r, fmt.Errorf("connection lost")
	}
	for _, p := range model.RecorderParameters() {
		r.Values[p] = 0
	}
	for _, s := range model.RecorderSetup() {
		r.Values[s.Parameter] = float32(s.Value)
	}
	r.Values["Recorder."+c.transport] = 1
	return r, nil
}
func (c *recorderClient) SetRecorder(string, int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return nil
}
func (c *recorderClient) set(v string) { c.mu.Lock(); defer c.mu.Unlock(); c.transport = v }
func TestRecorderExternalObservationAndWorkerOwnership(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	c.Studio.Recording = &config.Recording{}
	c.Validate()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &recorderClient{ruleClient: &ruleClient{&fakeClient{}}, transport: "record"}
	failSave := true
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Acquire: func() (func(), error) { return nil, fmt.Errorf("writer conflict") }, Load: func(string) (config.Config, error) { return c, nil }, Save: func(_ string, _ config.Config, _ *config.Intent, _ string) (string, error) {
		if failSave {
			return "", fmt.Errorf("disk full")
		}
		return "saved", nil
	}}
	states := make(chan State, 1)
	actions := make(chan Action, 8)
	done := make(chan struct{})
	go Work(ctx, c, "unused", "", false, deps, actions, states, done)
	s := nextState(t, states)
	if s.Recorder.State() != "Recording" {
		t.Fatal(s)
	}
	actions <- Action{Kind: editRule, Row: "record-mic", Value: "true", Revision: s.Revision}
	s = nextState(t, states)
	if s.Intent.Recording.MicEnabled || !strings.Contains(s.Notice, "disk full") {
		t.Fatal(s)
	}
	actions <- ToggleLive
	s = nextState(t, states)
	if s.Live || !strings.Contains(s.Notice, "writer conflict") {
		t.Fatal(s)
	}
	prior := s.Revision
	client.set("unknown")
	actions <- Refresh
	s = nextState(t, states)
	if s.Recorder.State() != "Unknown" || s.Revision == prior {
		t.Fatal(s)
	}
	client.set("stop")
	actions <- Refresh
	s = nextState(t, states)
	if s.Recorder.State() != "Stopped" {
		t.Fatal(s)
	}
	actions <- Reload
	s = nextState(t, states)
	if s.Intent.Recording.MicEnabled {
		t.Fatal(s)
	}
	cancel()
	<-done
	if client.calls != 0 {
		t.Fatal("external transport fought")
	}
}

func TestRecordingWorkerDryAndStale(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	c.Studio.Recording = &config.Recording{}
	c.Validate()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &ruleClient{&fakeClient{}}
	saves := 0
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Save: func(_ string, _ config.Config, _ *config.Intent, _ string) (string, error) {
		saves++
		return "saved", nil
	}}
	states := make(chan State, 1)
	actions := make(chan Action, 8)
	done := make(chan struct{})
	go Work(ctx, c, "unused", "", false, deps, actions, states, done)
	s := nextState(t, states)
	actions <- Action{Kind: editRule, Row: "record-mic", Value: "true", Revision: s.Revision}
	s = nextState(t, states)
	if !s.Intent.Recording.MicEnabled || saves != 1 || client.writes.Load() != 0 {
		t.Fatal(s)
	}
	actions <- Action{Kind: startRecording, Revision: s.Revision - 1}
	s = nextState(t, states)
	if !strings.Contains(s.Notice, "Stale") {
		t.Fatal(s)
	}
	actions <- Action{Kind: startRecording, Revision: s.Revision}
	for n := 0; n < 3; n++ {
		s = nextState(t, states)
		if strings.Contains(s.Notice, "live mode") {
			break
		}
	}
	if !strings.Contains(s.Notice, "live mode") || client.writes.Load() != 0 {
		t.Fatal(s)
	}
	cancel()
	<-done
}
