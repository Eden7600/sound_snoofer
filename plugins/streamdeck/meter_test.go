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
	room := snoofer.Control{ID: "hue.group", Label: "Hue Room", ShortLabel: "Room", Value: "bc32966d-3aaa", Options: []string{"bc32966d-3aaa"},
		OptionLabels: map[string]string{"bc32966d-3aaa": "Cody Office"}, Operations: []string{"set"}, Available: true}
	if tile := bindingTile(Binding{Control: room.ID}, room, true, time.Now()); tile.Value != "Cody Office" {
		t.Fatal(tile)
	}
	room.Value = "unknown-id"
	if tile := bindingTile(Binding{Control: room.ID}, room, true, time.Now()); tile.Value != "unknown-id" {
		t.Fatal("unlabelled value", tile)
	}
}

func TestInactiveControlTile(t *testing.T) {
	c := snoofer.Control{ID: "audio.mic-mute", Label: "Mic Mute", ShortLabel: "Mute", Icon: "mic-mute", Value: "On", Available: true, Inactive: true}
	tile := bindingTile(Binding{Control: c.ID}, c, true, time.Now())
	if !tile.Inactive || tile.Label != "Mute" || tile.Icon != "mic-mute" || tile.Value != "" || tile.Meter {
		t.Fatal("inactive tile", tile)
	}
}

func TestSceneBrightnessReachesTheTile(t *testing.T) {
	half := 0.5
	c := snoofer.Control{ID: "hue.scene-x", Label: "Relax", Icon: "hue-scene", Value: "Ready", Available: true, Operations: []string{"press"}, Brightness: &half}
	if tile := bindingTile(Binding{Control: c.ID}, c, true, time.Now()); !tile.BrightnessKnown || tile.Brightness != 0.5 {
		t.Fatal("brightness lost on the way to the tile", tile)
	}
	c.Brightness = nil
	if tile := bindingTile(Binding{Control: c.ID}, c, true, time.Now()); tile.BrightnessKnown {
		t.Fatal("brightness invented", tile)
	}
}
