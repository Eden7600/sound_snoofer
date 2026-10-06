package streamdeck

import (
	"sound-snoofer/snoofer"
	"testing"
	"time"
)

func TestMeterExpiresWithoutNewPublication(t *testing.T) {
	now := time.Now()
	c := snoofer.Control{Label: "A1", Meter: snoofer.Meter{Present: true, Known: true, DB: -12, At: now}}
	if tile := controlTile(c, "0 dB", now); !tile.Meter || !tile.LevelKnown || tile.LevelDB != -12 {
		t.Fatal(tile)
	}
	if controlTile(c, "0 dB", now.Add(501*time.Millisecond)).LevelKnown {
		t.Fatal("stale meter")
	}
	c.Meter.Known = false
	if controlTile(c, "0 dB", now).LevelKnown {
		t.Fatal("unknown meter")
	}
	if controlTile(snoofer.Control{}, "Home", now).Meter {
		t.Fatal("navigation has meter")
	}
}

func TestSelectionKeyShowsOptionLabel(t *testing.T) {
	room := snoofer.Control{ID: "hue.group", Label: "Hue room", ShortLabel: "Room", Value: "bc32966d-3aaa", Options: []string{"bc32966d-3aaa"},
		OptionLabels: map[string]string{"bc32966d-3aaa": "Cody Office"}, Operations: []string{"set"}, Available: true}
	if tile := bindingTile(Binding{Control: room.ID}, room, true, time.Now()); tile.Value != "Cody Office" {
		t.Fatal(tile)
	}
	room.Value = "unknown-id"
	if tile := bindingTile(Binding{Control: room.ID}, room, true, time.Now()); tile.Value != "unknown-id" {
		t.Fatal("unlabelled value", tile)
	}
}
