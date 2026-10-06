package streamdeck

import (
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestCompactPresentationPreservesDiagnostics(t *testing.T) {
	c := snoofer.Control{Label: "Mic mute", ShortLabel: "Mute", Icon: "mic-mute", Value: "On"}
	for status, want := range map[string]string{"": "On", "Pending": "Wait", "Unknown": "N/A", "Disconnected": "N/A", "native write failed": "Error", "VR override": "VR"} {
		c.Status = status
		tile := controlTile(c, c.Value, time.Now())
		if tile.Label != "Mute" || tile.Value != want || c.Status != status {
			t.Fatal(tile, c)
		}
	}
	c.ShortLabel = ""
	if controlTile(c, c.Value, time.Now()).Label != c.Label {
		t.Fatal("lost custom label")
	}
}

func TestEmptyProviderSlotRendersBlank(t *testing.T) {
	now := time.Now()
	binding := Binding{Control: "hue.room-scene-5", Label: "Relax"}
	if tile := bindingTile(binding, snoofer.Control{ID: binding.Control}, true, now); tile != (device.Tile{}) {
		t.Fatalf("empty slot rendered %+v", tile)
	}
	unavailable := snoofer.Control{ID: "hue.sync", ShortLabel: "Sync", Icon: "huesync-sync"}
	if tile := bindingTile(Binding{Control: unavailable.ID}, unavailable, true, now); tile.Value != "N/A" || tile.Label != "Sync" {
		t.Fatalf("labelled unavailable control rendered %+v", tile)
	}
	if tile := bindingTile(binding, snoofer.Control{}, false, now); tile.Value != "N/A" || tile.Label != "Relax" {
		t.Fatalf("missing provider rendered %+v", tile)
	}
	if tile := bindingTile(Binding{}, snoofer.Control{}, false, now); tile != (device.Tile{}) {
		t.Fatalf("unbound position rendered %+v", tile)
	}
}

func TestHiddenControlRendersBlank(t *testing.T) {
	hidden := snoofer.Control{ID: "hue.sync-mode", ShortLabel: "Mode", Icon: "huesync-mode", Value: "video", Hidden: true}
	if tile := bindingTile(Binding{Control: hidden.ID, Label: "Mode"}, hidden, true, time.Now()); tile != (device.Tile{}) {
		t.Fatalf("hidden control rendered %+v", tile)
	}
}
