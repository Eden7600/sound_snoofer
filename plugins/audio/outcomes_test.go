package audio

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

func TestOnlyExplicitMuteActionsCaptureGenerations(t *testing.T) {
	s := control.State{Revision: 10, MicMuteGeneration: 2, PlaybackMuteGeneration: 4, Intent: &config.Intent{Enabled: true, Source: "desk", Mode: "direct", Monitor: "off"}}
	for _, test := range []struct {
		id, row    string
		generation uint64
	}{
		{"mic-mute", "mic-mute", 2}, {"speaker-mute", "speaker-mute", 4},
		{"gain-mic", "mic-mute", 0}, {"gain-playback", "speaker-mute", 0},
	} {
		a, err := action(s, snoofer.Request{ID: "audio." + test.id, Operation: "press"})
		if err != nil || a.MuteGeneration != test.generation || a.Value != "true" || a.Row != test.row || a.Revision != 10 {
			t.Fatal(test.id, a, err)
		}
	}
	i := &Instance{state: s, actions: make(chan control.Action, 1)}
	results := make(chan snoofer.Outcome, 1)
	if err := i.SetMicMuteWithResult(true, "discord", 7, func(o snoofer.Outcome) { results <- o }); err != nil {
		t.Fatal(err)
	}
	a := <-i.actions
	if a.Revision != 7 || a.MuteGeneration != 0 || a.Complete == nil {
		t.Fatal("linked mute recaptured target", a)
	}
	a.Complete(snoofer.Outcome{Status: snoofer.Rejected, Detail: "stale linked audio"})
	if result := <-results; result.Detail != "stale linked audio" {
		t.Fatal(result)
	}
}

func TestPriorityOutcomeWaitsForWorkerReload(t *testing.T) {
	i := editInstance(t, func(string, json.RawMessage, json.RawMessage) error { return nil })
	i.statePath = t.TempDir() + "/audio"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { i.runEdits(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	value, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "move", Index: 1, Value: "up"})
	results := make(chan snoofer.Outcome, 1)
	i.edits <- editRequest{expected: string(i.raw), control: "audio.priority-edit", value: string(value), complete: func(o snoofer.Outcome) { results <- o }}
	var reload control.Action
	select {
	case reload = <-i.actions:
	case <-time.After(time.Second):
		t.Fatal("reload not queued")
	}
	select {
	case result := <-results:
		t.Fatal("save claimed reload completed", result)
	default:
	}
	if reload.Complete == nil {
		t.Fatal("reload lost completion")
	}
	reload.Complete(snoofer.Outcome{Status: snoofer.Failed, Detail: "reload denied"})
	if result := <-results; result.Detail != "reload denied" {
		t.Fatal(result)
	}
}

func TestPriorityOutcomeReportsSaveFailureAndQueuedStop(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	i := editInstance(t, func(string, json.RawMessage, json.RawMessage) error {
		close(started)
		<-release
		return errors.New("save denied")
	})
	i.statePath = t.TempDir() + "/audio"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { i.runEdits(ctx); close(done) }()
	value, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "move", Index: 1, Value: "up"})
	first, second := make(chan snoofer.Outcome, 1), make(chan snoofer.Outcome, 1)
	i.edits <- editRequest{expected: string(i.raw), control: "audio.priority-edit", value: string(value), complete: func(o snoofer.Outcome) { first <- o }}
	<-started
	i.edits <- editRequest{expected: string(i.raw), control: "audio.priority-edit", value: string(value), complete: func(o snoofer.Outcome) { second <- o }}
	cancel()
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("edit worker did not stop")
	}
	if result := <-first; result.Status != snoofer.Failed || result.Detail != "save denied" {
		t.Fatal(result)
	}
	if result := <-second; result.Status != snoofer.Unknown {
		t.Fatal(result)
	}
	select {
	case a := <-i.actions:
		t.Fatal("failed save queued reload", a)
	default:
	}
}

func TestSoundboardVolumeCompletionQueueAndStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := control.State{Revision: 8, Live: true, Connected: true, ObservedAt: time.Now(), Snapshot: model.Snapshot{Edition: 3, Numbers: map[string]float32{"Strip[7].A1": 0, "Strip[7].B2": 0, "Strip[7].B3": 0, "Strip[7].Gain": -6}}, Plan: &routing.Plan{Topology: &routing.Topology{Operations: []routing.Operation{{Parameter: "Strip[7].A1", Value: 0}}}}}
	i := &Instance{ctx: ctx, state: s, soundboardReserved: true, soundboard: &SoundboardRoutes{}, actions: make(chan control.Action, 1)}
	c, handler := i.SoundboardVolume()
	registry := snoofer.NewControls()
	if err := registry.Publish("soundboard", []snoofer.Control{c}, handler); err != nil {
		t.Fatal(err)
	}
	published := registry.Snapshot()[0]
	request := snoofer.Request{ID: published.ID, Revision: published.Revision, Operation: "adjust", Delta: 1}
	results := make(chan snoofer.Outcome, 3)
	if err := registry.DispatchWithResult(context.Background(), request, func(o snoofer.Outcome) { results <- o }); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-results:
		t.Fatal("queued audio gain completed early", result)
	default:
	}
	if err := registry.DispatchWithResult(context.Background(), request, func(o snoofer.Outcome) { results <- o }); err == nil {
		t.Fatal("full audio queue accepted")
	}
	if result := <-results; result.Status != snoofer.Rejected {
		t.Fatal(result)
	}
	a := <-i.actions
	if a.Revision != 8 || a.Complete == nil {
		t.Fatal(a)
	}
	cancel()
	select {
	case result := <-results:
		if result.Status != snoofer.Unknown {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("provider stop leaked audio completion")
	}
	a.Complete(snoofer.Outcome{Status: snoofer.Processed})
	select {
	case result := <-results:
		t.Fatal("late audio completion delivered twice", result)
	default:
	}
}

func TestQueuedPriorityIndexRetainsPublishedConfiguration(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var saves atomic.Int32
	i := editInstance(t, func(string, json.RawMessage, json.RawMessage) error {
		if saves.Add(1) == 1 {
			close(started)
			<-release
		}
		return nil
	})
	i.statePath = t.TempDir() + "/audio"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { i.runEdits(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	value, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "remove", Index: 0})
	firstResult, secondResult := make(chan snoofer.Outcome, 1), make(chan snoofer.Outcome, 1)
	edit := editRequest{expected: string(i.raw), control: "audio.priority-edit", value: string(value), complete: func(o snoofer.Outcome) { firstResult <- o }}
	i.edits <- edit
	<-started
	edit.complete = func(o snoofer.Outcome) { secondResult <- o }
	i.edits <- edit
	close(release)
	reload := <-i.actions
	reload.Complete(snoofer.Outcome{Status: snoofer.Processed, Detail: "Reloaded"})
	if first := <-firstResult; first.Status != snoofer.Processed {
		t.Fatal(first)
	}
	select {
	case second := <-secondResult:
		if second.Status != snoofer.Rejected {
			t.Fatal("stale queued edit was not refused", second)
		}
	case secondReload := <-i.actions:
		secondReload.Complete(snoofer.Outcome{Status: snoofer.Processed})
		t.Fatal("queued index edit retargeted the newer priority list")
	case <-time.After(time.Second):
		t.Fatal("stale edit did not complete")
	}
	if saves.Load() != 1 {
		t.Fatal("stale queued edit reached save", saves.Load())
	}
}

func TestPriorityValidationIsRejectedBeforeSave(t *testing.T) {
	var saves atomic.Int32
	i := editInstance(t, func(string, json.RawMessage, json.RawMessage) error { saves.Add(1); return nil })
	i.statePath = t.TempDir() + "/audio"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { i.runEdits(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	results := make(chan snoofer.Outcome, 1)
	i.edits <- editRequest{expected: string(i.raw), control: "audio.priority-edit", value: "invalid JSON", complete: func(o snoofer.Outcome) { results <- o }}
	select {
	case result := <-results:
		if result.Status != snoofer.Rejected {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("invalid edit did not complete")
	}
	if saves.Load() != 0 {
		t.Fatal("invalid edit reached save")
	}
}
