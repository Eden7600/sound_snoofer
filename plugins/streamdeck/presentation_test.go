package streamdeck

import (
	"testing"
	"time"

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
