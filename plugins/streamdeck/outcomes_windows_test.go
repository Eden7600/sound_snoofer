//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func deckOutcome(t *testing.T, results <-chan snoofer.Outcome) snoofer.Outcome {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("missing action result")
		return snoofer.Outcome{}
	}
}

func waitDeckSave(t *testing.T, entered <-chan struct{}, results <-chan snoofer.Outcome) {
	t.Helper()
	select {
	case <-entered:
	case result := <-results:
		t.Fatalf("action completed before save started: %+v", result)
	case <-time.After(time.Second):
		t.Fatal("save did not start")
	}
}

func TestDeckQueuedEditKeepsOriginalContext(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	registry, _ := startOutcomeDeck(t, func(string, json.RawMessage, json.RawMessage) error {
		close(entered)
		<-release
		return nil
	})
	defer close(release)
	first := sendDeck(t, registry, "deck-name", "set", "Desk")
	waitDeckSave(t, entered, first)
	second := sendDeck(t, registry, "name", "set", "Wrong page")
	release <- struct{}{}
	if result := deckOutcome(t, first); result.Status != snoofer.Processed {
		t.Fatal(result)
	}
	if result := deckOutcome(t, second); result.Status != snoofer.Rejected {
		t.Fatal(result)
	}
	if name := deckControl(t, registry, "name", nil).Value; name != "A" {
		t.Fatalf("queued edit rebased: %s", name)
	}
}

func TestDeckQueueFullAndStopCompleteOnce(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var saves atomic.Int32
	registry, cancel := startOutcomeDeck(t, func(string, json.RawMessage, json.RawMessage) error {
		saves.Add(1)
		close(entered)
		<-release
		return nil
	})
	defer close(release)
	results := []<-chan snoofer.Outcome{sendDeck(t, registry, "deck-name", "set", "Desk")}
	waitDeckSave(t, entered, results[0])
	for range 8 {
		results = append(results, sendDeck(t, registry, "name", "set", "Queued"))
	}
	c := deckControl(t, registry, "name", nil)
	refused := make(chan snoofer.Outcome, 2)
	err := registry.DispatchWithResult(context.Background(), snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "set", Value: "Overflow"}, func(r snoofer.Outcome) { refused <- r })
	if err == nil || deckOutcome(t, refused).Status != snoofer.Rejected {
		t.Fatalf("full queue error = %v", err)
	}
	cancel()
	for _, result := range results {
		if got := deckOutcome(t, result); got.Status != snoofer.Unknown {
			t.Fatal(got)
		}
	}
	release <- struct{}{}
	deadline := time.Now().Add(time.Second)
	for len(registry.Snapshot()) != 0 {
		if time.Now().After(deadline) {
			t.Fatal("provider did not stop")
		}
		time.Sleep(time.Millisecond)
	}
	for _, result := range append(results, refused) {
		select {
		case got := <-result:
			t.Fatalf("duplicate completion: %+v", got)
		default:
		}
	}
	if saves.Load() != 1 {
		t.Fatal("queued work ran after stop")
	}
}

func TestDeckValidationCompletesWithoutSave(t *testing.T) {
	registry, _ := startOutcomeDeck(t, func(string, json.RawMessage, json.RawMessage) error { t.Error("invalid edit saved"); return nil })
	for _, edit := range []struct{ id, operation, value string }{{"name", "set", "  "}, {"region-remove", "set", "999"}} {
		before := deckControl(t, registry, edit.id, nil)
		if got := deckOutcome(t, sendDeck(t, registry, edit.id, edit.operation, edit.value)); got.Status != snoofer.Rejected {
			t.Fatal(got)
		}
		deckControl(t, registry, edit.id, func(c snoofer.Control) bool { return c.Revision != before.Revision })
	}
}

func deckControl(t *testing.T, registry *snoofer.Controls, id string, ready func(snoofer.Control) bool) snoofer.Control {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, control := range registry.Snapshot() {
			if control.ID == "streamdeck."+id && (ready == nil || ready(control)) {
				return control
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("missing ready control %s", id)
	return snoofer.Control{}
}

func startOutcomeDeck(t *testing.T, save func(string, json.RawMessage, json.RawMessage) error) (*snoofer.Controls, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	registry := snoofer.NewControls()
	settings := Settings{Layout: Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}}, Devices: map[string]Deck{testDeck: {Model: plusXLModel}}}
	instance, err := startWithSurface(ctx, snoofer.Services{Controls: registry, SaveSettings: save}, snoofer.MarshalSettings(settings), fakeSurface(make(chan device.Frame, 1), make(chan device.Event, 8), make(chan struct{})))
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
	deckControl(t, registry, "device-"+testDeck, func(c snoofer.Control) bool {
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionConnected
	})
	return registry, cancel
}

func sendDeck(t *testing.T, registry *snoofer.Controls, id, operation, value string) <-chan snoofer.Outcome {
	t.Helper()
	control := deckControl(t, registry, id, nil)
	results := make(chan snoofer.Outcome, 2)
	if err := registry.DispatchWithResult(context.Background(), snoofer.Request{ID: control.ID, Revision: control.Revision, Operation: operation, Value: value}, func(result snoofer.Outcome) { results <- result }); err != nil {
		t.Fatal(err)
	}
	return results
}

func TestDeckSaveResultWaitsForPersistence(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "saved", true: "failed"}[fail], func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			registry, _ := startOutcomeDeck(t, func(string, json.RawMessage, json.RawMessage) error {
				close(entered)
				<-release
				if fail {
					return errors.New("disk refused write")
				}
				return nil
			})
			defer close(release)
			results := sendDeck(t, registry, "deck-name", "set", "Desk")
			waitDeckSave(t, entered, results)
			select {
			case result := <-results:
				t.Fatalf("queue acceptance claimed completion: %+v", result)
			default:
			}
			release <- struct{}{}
			result := deckOutcome(t, results)
			want := snoofer.Processed
			if fail {
				want = snoofer.Failed
			}
			if result.Status != want || (fail && result.Detail != "disk refused write") {
				t.Fatalf("result = %+v, want %s", result, want)
			}
		})
	}
}

func TestDeckInitialConnectionRejectsOldEdit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	registry := snoofer.NewControls()
	hardware := &navigationSurface{ctx: ctx, decksSurface: &decksSurface{
		events: make(chan device.Event, 8), done: make(chan struct{}), frames: map[string]device.Frame{},
	}}
	block := hardware.blockPage(testDeck, "A")
	settings := Settings{Layout: Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}}, Devices: map[string]Deck{testDeck: {Model: plusXLModel}}}
	var saves atomic.Int32
	instance, err := startWithSurface(ctx, snoofer.Services{
		Controls: registry,
		SaveSettings: func(string, json.RawMessage, json.RawMessage) error {
			saves.Add(1)
			return nil
		},
	}, snoofer.MarshalSettings(settings), func(ctx context.Context) surface {
		go func() {
			<-ctx.Done()
			close(hardware.done)
		}()
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
	// Remembered editor controls exist before the initial connection is published.
	deckControl(t, registry, "deck", func(c snoofer.Control) bool { return c.Value == testDeck })
	disconnected := deckControl(t, registry, "device-"+testDeck, nil)
	if disconnected.Connection == nil || disconnected.Connection.State != snoofer.ConnectionDisconnected {
		t.Fatalf("remembered deck unexpectedly connected: %+v", disconnected)
	}
	hardware.events <- device.Event{Connected: true, Device: testDeck, Serial: testDeck, Model: plusXLModel}
	select {
	case <-block.entered:
	case <-time.After(time.Second):
		t.Fatal("connection publication did not block")
	}
	result := sendDeck(t, registry, "deck-name", "set", "Desk")
	close(block.release)
	got := deckOutcome(t, result)
	if got.Status != snoofer.Rejected || got.Detail != "Editor changed; try again" || saves.Load() != 0 {
		t.Fatalf("old command = %+v, saves = %d", got, saves.Load())
	}
	deckControl(t, registry, "device-"+testDeck, func(c snoofer.Control) bool {
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionConnected
	})
	got = deckOutcome(t, sendDeck(t, registry, "deck-name", "set", "Desk"))
	if got.Status != snoofer.Processed || saves.Load() != 1 {
		t.Fatalf("connected command = %+v, saves = %d", got, saves.Load())
	}
}
