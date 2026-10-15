//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

type navigationBlock struct {
	deck, page string
	entered    chan struct{}
	release    chan struct{}
}

// Block after the worker changes its target but before Publish replaces the old handler.
type navigationSurface struct {
	*decksSurface
	ctx   context.Context
	mu    sync.Mutex
	block *navigationBlock
}

func (s *navigationSurface) Show(id string, frame device.Frame) {
	s.decksSurface.Show(id, frame)
	s.mu.Lock()
	block := s.block
	if block != nil && block.deck == id && block.page == pageOf(frame) {
		s.block = nil
	} else {
		block = nil
	}
	s.mu.Unlock()
	if block != nil {
		close(block.entered)
		select {
		case <-block.release:
		case <-s.ctx.Done():
		}
	}
}

func (s *navigationSurface) blockPage(deck, page string) *navigationBlock {
	block := &navigationBlock{deck: deck, page: page, entered: make(chan struct{}), release: make(chan struct{})}
	s.mu.Lock()
	s.block = block
	s.mu.Unlock()
	return block
}

func startNavigationOutcomes(t *testing.T) (*snoofer.Controls, *navigationSurface) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	registry := snoofer.NewControls()
	hardware := &navigationSurface{ctx: ctx, decksSurface: &decksSurface{
		events: make(chan device.Event, 8), done: make(chan struct{}), frames: map[string]device.Frame{},
	}}
	settings := Settings{
		Layout:  Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}},
		Devices: map[string]Deck{"main": {Model: plusXLModel}, "other": {Model: plusXLModel}},
	}
	instance, err := startWithSurface(ctx, snoofer.Services{
		Controls: registry,
		SaveSettings: func(string, json.RawMessage, json.RawMessage) error {
			t.Error("navigation must not persist settings")
			return nil
		},
	}, snoofer.MarshalSettings(settings), func(ctx context.Context) surface {
		go func() { <-ctx.Done(); close(hardware.done) }()
		return hardware
	})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		stop, release := context.WithTimeout(context.Background(), time.Second)
		defer release()
		if err := instance.Stop(stop); err != nil {
			t.Error(err)
		}
	})
	for _, id := range []string{"main", "other"} {
		before := deckControl(t, registry, "name", nil)
		hardware.events <- device.Event{Device: id, Serial: id, Model: plusXLModel, Connected: true}
		deckControl(t, registry, "name", func(c snoofer.Control) bool { return c.Revision != before.Revision })
	}
	return registry, hardware
}

func queuedNavigation(t *testing.T, registry *snoofer.Controls, block *navigationBlock) <-chan snoofer.Outcome {
	t.Helper()
	select {
	case <-block.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("target-changing frame did not block publication")
	}
	result := sendDeck(t, registry, "goto-c", "press", "")
	select {
	case got := <-result:
		t.Fatalf("queue acceptance completed blocked navigation: %+v", got)
	default:
	}
	close(block.release)
	return result
}

func TestDeckNavigationOutcomeProcessed(t *testing.T) {
	registry, hardware := startNavigationOutcomes(t)
	if result := deckOutcome(t, sendDeck(t, registry, "goto-c", "press", "")); result.Status != snoofer.Processed {
		t.Fatalf("navigation result = %+v", result)
	}
	deckControl(t, registry, "goto-c", func(c snoofer.Control) bool { return c.Value == "Here" })
	for id, want := range map[string]string{"main": "C", "other": "A"} {
		frame, ok := hardware.frame(id)
		if !ok || pageOf(frame) != want {
			t.Errorf("deck %s page = %q, want %q", id, pageOf(frame), want)
		}
	}
}

func TestDeckNavigationOutcomeRejectsChangedFocus(t *testing.T) {
	registry, hardware := startNavigationOutcomes(t)
	other, ok := hardware.frame("other")
	if !ok {
		t.Fatal("missing second deck frame")
	}
	block := hardware.blockPage("other", "B")
	hardware.events <- device.Event{Device: "other", Encoder: 5, Delta: 1, Generation: other.Generation}
	if result := deckOutcome(t, queuedNavigation(t, registry, block)); result.Status != snoofer.Rejected {
		t.Fatalf("navigation crossed focused decks: %+v", result)
	}
	for id, want := range map[string]string{"main": "A", "other": "B"} {
		frame, ok := hardware.frame(id)
		if !ok || pageOf(frame) != want {
			t.Errorf("old navigation changed deck %s to %q, want %q", id, pageOf(frame), want)
		}
	}
}

func TestDeckNavigationOutcomeRejectsRecreatedDeviceID(t *testing.T) {
	registry, hardware := startNavigationOutcomes(t)
	if result := deckOutcome(t, sendDeck(t, registry, "goto-b", "press", "")); result.Status != snoofer.Processed {
		t.Fatal(result)
	}
	deckControl(t, registry, "goto-b", func(c snoofer.Control) bool { return c.Value == "Here" })
	block := hardware.blockPage("main", "A")
	hardware.events <- device.Event{Device: "main", Serial: "main", Model: plusXLModel, Connected: true}
	if result := deckOutcome(t, queuedNavigation(t, registry, block)); result.Status != snoofer.Rejected {
		t.Fatalf("navigation crossed a recreated device ID: %+v", result)
	}
	frame, ok := hardware.frame("main")
	if !ok || pageOf(frame) != "A" {
		t.Fatalf("old navigation changed replacement deck to %q", pageOf(frame))
	}
}
