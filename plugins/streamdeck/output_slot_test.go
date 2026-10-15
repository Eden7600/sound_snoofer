package streamdeck

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestOutputSlotBindingFixture(t *testing.T) {
	raw, err := os.ReadFile("../../app/web/test/fixtures/audio.live.json")
	if err != nil {
		t.Fatal(err)
	}
	var controls []snoofer.Control
	if err := json.Unmarshal(raw, &controls); err != nil {
		t.Fatal(err)
	}
	var slot snoofer.Control
	for _, c := range controls {
		if c.ID == "audio.slot-1" {
			slot = c
		}
	}
	if slot.ID == "" {
		t.Fatal("generated provider fixture lacks slot")
	}
	for _, state := range []string{"In use", "No device", "Disconnected", "No output"} {
		t.Run(state, func(t *testing.T) {
			c := slot
			c.Value = state
			tile := bindingTile(Binding{Control: c.ID}, c, true, time.Time{})
			if tile.Label != "Music" || tile.Value != state {
				t.Fatalf("provider destination became %+v; want Music / %s", tile, state)
			}
		})
	}
	for _, c := range controls {
		if c.ID != "audio.slot-2" {
			continue
		}
		tile := bindingTile(Binding{Control: c.ID}, c, true, time.Time{})
		if tile.Label != "" || tile.Value != "" {
			t.Fatalf("hidden slot became %+v", tile)
		}
	}
	for status, want := range map[string]string{"Pending": "Wait", "Unavailable": "N/A", "write failed": "Error"} {
		c := snoofer.Control{ID: "test.control", Label: "Test", Kind: "toggle", Value: "On", Status: status, Available: true}
		tile := bindingTile(Binding{Control: c.ID}, c, true, time.Time{})
		if tile.Value != want {
			t.Fatalf("%s became %s; want %s", status, tile.Value, want)
		}
	}
}
