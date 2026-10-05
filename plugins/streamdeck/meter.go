package streamdeck

import (
	"math"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func controlTile(c snoofer.Control, value string, now time.Time) device.Tile {
	label := c.ShortLabel
	if label == "" {
		label = c.Label
	}
	if c.Status != "" {
		switch c.Status {
		case "Pending":
			value = "Wait"
		case "Unknown", "Disconnected", "Unavailable":
			value = "N/A"
		case "VR override":
			value = "VR"
		default:
			value = "Error"
		}
	}
	m := c.Meter
	age := now.Sub(m.At)
	known := m.Known && age >= 0 && age <= 500*time.Millisecond && !math.IsNaN(m.DB) && !math.IsInf(m.DB, 0)
	if !known {
		m.DB = 0
	}
	return device.Tile{Artwork: c.Artwork, Label: label, Value: value, Icon: c.Icon, Meter: m.Present,
		LevelKnown: known, LevelDB: m.DB}
}
