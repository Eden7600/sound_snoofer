package audio

import (
	"context"
	"errors"
	"testing"
	"time"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/voicemeeter"
)

func echoState(strip int, effective, playback string) control.State {
	return control.State{
		Live:      true,
		Connected: true,
		Snapshot:  model.Snapshot{Edition: 3, Assignments: map[string]string{"A2": "Speakers (Realtek)"}},
		Plan: &routing.Plan{Topology: &routing.Topology{
			PlaybackTarget: playback,
			Voice:          &routing.VoiceStatus{Strip: strip, Effective: effective},
		}},
	}
}

func TestEchoTargets(t *testing.T) {
	notLive := echoState(0, "desk", "A2")
	notLive.Live = false
	banana := echoState(0, "desk", "A2")
	banana.Snapshot.Edition = 2
	for name, test := range map[string]struct {
		state     control.State
		mic       [2]int
		ref       [8]int
		playback  string
		reasonSet bool
	}{
		"desk to A2":       {state: echoState(0, "desk", "A2"), mic: [2]int{0, 1}, ref: [8]int{8, 9, 10, 11, 12, 13, 14, 15}, playback: "Speakers (Realtek)"},
		"lav to A1":        {state: echoState(1, "lav", "A1"), mic: [2]int{2, 3}, ref: [8]int{0, 1, 2, 3, 4, 5, 6, 7}},
		"VR strip 5 to A5": {state: echoState(4, "vr:quest", "A5"), mic: [2]int{8, 9}, ref: [8]int{32, 33, 34, 35, 36, 37, 38, 39}},
		"virtual strip":    {state: echoState(6, "element", "A1"), mic: [2]int{18, 19}, ref: [8]int{0, 1, 2, 3, 4, 5, 6, 7}},
		"mic off":          {state: echoState(-1, "off", "A2"), mic: [2]int{-1, -1}, ref: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, reasonSet: true},
		"no mic":           {state: echoState(-1, "unavailable", "A2"), mic: [2]int{-1, -1}, ref: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, reasonSet: true},
		"no playback":      {state: echoState(0, "desk", ""), mic: [2]int{-1, -1}, ref: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, reasonSet: true},
		"virtual playback": {state: echoState(0, "desk", "B1"), mic: [2]int{-1, -1}, ref: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, reasonSet: true},
		"not live":         {state: notLive, mic: [2]int{-1, -1}, ref: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, reasonSet: true},
		"not Potato":       {state: banana, mic: [2]int{-1, -1}, ref: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, reasonSet: true},
	} {
		got := echoTargets(test.state)
		if got.Mic != test.mic || got.Reference != test.ref || got.Playback != test.playback || (got.Reason != "") != test.reasonSet {
			t.Errorf("%s: %+v", name, got)
		}
	}
}

func TestEchoTargetsStream(t *testing.T) {
	s := echoState(0, "desk", "A2")
	if got := echoTargets(s).Stream; got != 0 {
		t.Fatal("stream without a callback monitor", got)
	}
	s.Snapshot.Callback = &model.CallbackStatus{Active: true, Starting: 4}
	if got := echoTargets(s).Stream; got != 4 {
		t.Fatal("stream count", got)
	}
	s.Snapshot.Callback.Active = false
	if got := echoTargets(s).Stream; got != 0 {
		t.Fatal("stream from an inactive monitor", got)
	}
}

func echoInstance(s control.State) *Instance {
	return &Instance{actions: make(chan control.Action, 1), done: make(chan struct{}), state: s}
}

func TestSetEchoInsertInstallsWithoutWaiting(t *testing.T) {
	i := echoInstance(control.State{})
	hook := &voicemeeter.InsertHook{Input: 1, Output: 2, Context: 3}
	if err := i.SetEchoInsert(context.Background(), hook); err != nil {
		t.Fatal(err)
	}
	got, generation := i.insertRequest()
	if got == nil || *got != *hook || got == hook || generation != 1 {
		t.Fatal(got, generation)
	}
	if action := <-i.actions; action.Kind != control.Refresh.Kind {
		t.Fatal(action)
	}
}

func TestSetEchoInsertRemovalWaitsForConfirmation(t *testing.T) {
	hook := &voicemeeter.InsertHook{Input: 1, Output: 2, Context: 3}
	i := echoInstance(control.State{Insert: hook, InsertGeneration: 1})
	i.echoInsert, i.echoGeneration = hook, 1
	result := make(chan error, 1)
	go func() { result <- i.SetEchoInsert(context.Background(), nil) }()
	// A step that read the request before removal must not count.
	time.Sleep(60 * time.Millisecond)
	i.mu.Lock()
	i.state = control.State{Insert: nil, InsertGeneration: 1}
	i.mu.Unlock()
	select {
	case err := <-result:
		t.Fatal("released before the removal's generation", err)
	case <-time.After(60 * time.Millisecond):
	}
	i.mu.Lock()
	i.state = control.State{Insert: nil, InsertGeneration: 2}
	i.mu.Unlock()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("removal not confirmed")
	}
}

func TestSetEchoInsertRemovalUnconfirmed(t *testing.T) {
	hook := &voicemeeter.InsertHook{Input: 1}
	// A failed removal keeps the hook at the old generation.
	i := echoInstance(control.State{Insert: hook, InsertGeneration: 1})
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if err := i.SetEchoInsert(ctx, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}

	stopped := echoInstance(control.State{Insert: hook, InsertGeneration: 1})
	stopped.stopError = errors.New("callback cleanup returned -1")
	close(stopped.done)
	if err := stopped.SetEchoInsert(context.Background(), nil); err == nil {
		t.Fatal("uncertain cleanup reported as released")
	}

	clean := echoInstance(control.State{Insert: hook, InsertGeneration: 1})
	close(clean.done)
	if err := clean.SetEchoInsert(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}
