//go:build windows

package streamdeck

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

// TestGUIFixture keeps app/web/test/fixtures/streamdeck.json equal to what the
// editor publishes for a fixed layout and fixed provider controls (the
// generated audio fixture plus soundboard clips). SNOOFER_UPDATE_FIXTURES=1
// rewrites it.
func TestGUIFixture(t *testing.T) {
	fixtures := filepath.Join("..", "..", "app", "web", "test", "fixtures")
	raw, err := os.ReadFile(filepath.Join(fixtures, "audio.live.json"))
	if err != nil {
		t.Fatal(err)
	}
	var audio []snoofer.Control
	if err := json.Unmarshal(raw, &audio); err != nil {
		t.Fatal(err)
	}
	registry := snoofer.NewControls()
	clips := []snoofer.Control{}
	for n, name := range []string{"fah", "sadge", "instinct", "airhorn", "ping"} {
		clips = append(clips, snoofer.Control{ID: "soundboard.clip-" + name, Label: name, Kind: "command", Value: "Ready", Available: true,
			Operations: []string{"press", "hold"}, Collection: "soundboard.clips", CollectionLabel: "Soundboard Clips", Order: n + 1})
	}

	// The shipped status bindings must validate when audio publishes first.
	if err := registry.Publish("audio", audio, nil); err != nil {
		t.Fatal(err)
	}
	if err := registry.Publish("soundboard", clips, nil); err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"plus-xl", "plus", "xl", "mk2", "neo"} {
		if err := DefaultLayoutFor(model).Validate(registry.Snapshot()); err != nil {
			t.Fatalf("%s default with audio present: %v", model, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	settings := Settings{
		Layout:  DefaultLayoutFor("plus-xl"),
		Devices: map[string]Deck{"A00WA6261J4HRK": {Model: "plus", Name: "Desk"}},
		Serials: map[string]Layout{"A00WA6261J4HRK": DefaultLayoutFor("plus")},
	}
	services := snoofer.Services{Controls: registry, SaveSettings: func(string, json.RawMessage, json.RawMessage) error { return nil }}
	surface := idleSurface(make(chan device.Frame, 1), make(chan device.Event, 1), make(chan struct{}))
	instance, err := startWithSurface(ctx, services, snoofer.MarshalSettings(settings), surface)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		stop, release := context.WithTimeout(context.Background(), time.Second)
		defer release()
		if err := instance.Stop(stop); err != nil {
			t.Error(err)
		}
	}()

	var published []snoofer.Control
	deadline := time.Now().Add(5 * time.Second)
	for published == nil {
		for _, c := range registry.Snapshot() {
			if c.ID == "streamdeck.binding" && slices.Contains(c.Options, "soundboard.clip-ping") {
				published = registry.Snapshot()
			}
		}
		if published == nil && time.Now().After(deadline) {
			t.Fatal("the editor never published its preview")
		}
		time.Sleep(10 * time.Millisecond)
	}
	deck := []snoofer.Control{}
	for _, c := range published {
		if strings.HasPrefix(c.ID, "streamdeck.") {
			c.Revision = 1 // Publish counts differ between runs; the GUI only compares revisions.
			if c.Connection != nil && !c.Connection.Since.IsZero() {
				c.Connection.Since = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) // Fixed, like the audio fixture.
			}
			deck = append(deck, c)
		}
	}
	slices.SortFunc(deck, func(a, b snoofer.Control) int { return strings.Compare(a.ID, b.ID) })
	got, err := json.MarshalIndent(deck, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join(fixtures, "streamdeck.json")
	if os.Getenv("SNOOFER_UPDATE_FIXTURES") == "1" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err, "; run with SNOOFER_UPDATE_FIXTURES=1 to create it")
	}
	if !bytes.Equal(bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")), got) {
		t.Fatalf("%s is out of date with the Stream Deck editor; run with SNOOFER_UPDATE_FIXTURES=1 and review the diff", path)
	}
}
