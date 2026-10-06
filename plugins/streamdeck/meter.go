package streamdeck

import (
	"math"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

// bindingTile presents a binding's current control. An unavailable control
// without label or icon is an empty provider slot and renders blank.
func bindingTile(b Binding, c snoofer.Control, ok bool, now time.Time) device.Tile {
	if b.Control == "" {
		return device.Tile{}
	}
	if !ok {
		return device.Tile{Label: b.Label, Value: "N/A"}
	}
	if !c.Available {
		if c.Label == "" && c.ShortLabel == "" && c.Icon == "" {
			return device.Tile{}
		}
		c.Status = "Unavailable"
		return controlTile(c, "N/A", now)
	}
	value := c.Value
	if c.Status != "" {
		value = c.Status
	}
	return controlTile(c, value, now)
}

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
