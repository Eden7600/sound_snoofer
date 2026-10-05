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
