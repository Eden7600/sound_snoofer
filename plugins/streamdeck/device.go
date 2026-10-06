package streamdeck

import (
	"strconv"
	"time"

	"sound-snoofer/snoofer"
)

// deviceLink is the Stream Deck hardware state for the connection report,
// kept apart from layout editor feedback. The plugin goroutine owns it.
type deviceLink struct {
	link      snoofer.ConnectionTracker
	connected bool
	serial    string
	connects  int
}

func (d *deviceLink) connectedTo(serial string, now time.Time) {
	d.connected = true
	d.serial = serial
	d.connects++
	d.link.Activity(now)
}

// failed records a discovery, read or write failure; the surface keeps retrying.
func (d *deviceLink) failed(message string, now time.Time) {
	d.connected = false
	d.link.Fail(message, now)
}

func (d *deviceLink) report(now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionDisconnected, "Not connected"
	if d.connected {
		state, value = snoofer.ConnectionConnected, "Connected"
	}
	d.link.Observe(state, now)
	var details []snoofer.ConnectionDetail
	if d.serial != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Serial", Value: d.serial})
	}
	if d.connects > 1 {
		details = append(details, snoofer.ConnectionDetail{Label: "Reconnects", Value: strconv.Itoa(d.connects - 1)})
	}
	return snoofer.Control{ID: "streamdeck.app-device", Label: "Stream Deck + XL", Group: "Stream Deck", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: d.link.Report("USB HID 0FD9:00C6", details...)}
}
