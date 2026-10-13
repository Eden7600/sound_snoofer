package streamdeck

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sound-snoofer/internal/presence"
	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

// discoveryReportID is the report shown while no deck is known.
const discoveryReportID = "streamdeck.app-device"

// deviceLink is a deck's hardware state for its connection report, kept
// apart from layout editor feedback. The plugin goroutine owns it.
type deviceLink struct {
	link      snoofer.ConnectionTracker
	connected bool
	serial    string
	model     string // Model ID.
	product   uint16 // USB product ID; 0 until connected.
	connects  int
}

func (d *deviceLink) connectedTo(serial, model string, product uint16, now time.Time) {
	d.connected = true
	d.serial = serial
	d.model = model
	d.product = product
	d.connects++
	d.link.Activity(now)
}

// failed records a discovery, read or write failure; the surface keeps retrying.
func (d *deviceLink) failed(message string, now time.Time) {
	d.connected = false
	d.link.Fail(message, now)
}

// reportID is a deck's connection report ID: by serial, or a digest of its
// HID path without one.
func reportID(deckID string) string {
	if path, ok := strings.CutPrefix(deckID, "path:"); ok {
		sum := sha1.Sum([]byte(path))
		return "streamdeck.device-" + hex.EncodeToString(sum[:4])
	}
	return "streamdeck.device-" + deckID
}

// report describes a deck, labelled with its name or model; id is the
// report ID and display the deck's display state.
func (d *deviceLink) report(id, name, display string, now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionDisconnected, "Not connected"
	if d.connected {
		state, value = snoofer.ConnectionConnected, "Connected"
	}
	d.link.Observe(state, now)
	label, endpoint := "Stream Deck", "USB HID 0FD9"
	if m, ok := device.ModelByID(d.model); ok {
		label = m.Name
	}
	if d.product != 0 {
		endpoint = fmt.Sprintf("USB HID 0FD9:%04X", d.product)
	}
	var details []snoofer.ConnectionDetail
	if name != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Model", Value: label})
		label = name
	}
	if d.serial != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Serial", Value: d.serial})
	}
	if d.connects > 1 {
		details = append(details, snoofer.ConnectionDetail{Label: "Reconnects", Value: strconv.Itoa(d.connects - 1)})
	}
	if display != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Display", Value: display})
	}
	return snoofer.Control{ID: id, Label: label, Group: "Stream Deck", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: d.link.Report(endpoint, details...)}
}

// displayState describes the deck's display for the connection report.
func displayState(s presence.State) string {
	switch {
	case !s.Known:
		return "On (lock state unknown)"
	case s.Locked:
		return "Dark (locked)"
	case s.DisplayOff:
		return "Dark (displays off)"
	}
	return "On"
}
