package streamdeck

import (
	"math"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

// bindingTile presents a binding's current control. Hidden controls and
// unavailable controls without label or icon (empty provider slots) render blank.
func bindingTile(b Binding, c snoofer.Control, ok bool, now time.Time) device.Tile {
	if b.Control == "" {
		return device.Tile{}
	}
	if !ok {
		return device.Tile{Label: b.Label, Value: "N/A"}
	}
	if c.Hidden {
		return device.Tile{}
	}
	if !c.Available {
		if c.Label == "" && c.ShortLabel == "" && c.Icon == "" {
			return device.Tile{}
		}
		c.Status = "Unavailable"
		return controlTile(c, "N/A", now)
	}
	value := c.Value
	if label := c.OptionLabels[c.Value]; label != "" {
		value = label // Selections show their option label, never a raw ID.
	}
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
	return device.Tile{Artwork: artworkAt(c, now), Label: label, Value: value, Icon: c.Icon, Meter: m.Present,
		LevelKnown: known, LevelDB: m.DB}
}

// artworkAt is the animation frame shown at now. Every key runs on the same
// clock, so equal animations stay in step; static artwork is returned as is.
func artworkAt(c snoofer.Control, now time.Time) string {
	var loop time.Duration
	for _, f := range c.Animation {
		loop += f.Delay
	}
	if loop <= 0 {
		return c.Artwork
	}
	at := time.Duration(now.UnixNano() % int64(loop))
	for _, f := range c.Animation {
		if at < f.Delay {
			return f.Artwork
		}
		at -= f.Delay
	}
	return c.Artwork
}
