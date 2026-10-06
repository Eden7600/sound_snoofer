package hue

import (
	"strconv"
	"time"

	"sound-snoofer/snoofer"
)

// Connection reports for the Third-party apps screen. They are built on the
// worker goroutine from state it owns and never contain the application key.

type details []snoofer.ConnectionDetail

func (d *details) add(label, value string) {
	if value != "" {
		*d = append(*d, snoofer.ConnectionDetail{Label: label, Value: value})
	}
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func (w *worker) bridgeReport(now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionDisconnected, w.status
	switch {
	case w.connected:
		state = snoofer.ConnectionConnected
	case w.connecting:
		state, value = snoofer.ConnectionConnecting, "Connecting"
	case w.status == "Not paired":
		state = snoofer.ConnectionUnconfigured
	case w.status == "Error" || w.status == "Multiple bridges":
		state = snoofer.ConnectionError
	}
	w.bridgeLink.Observe(state, now)
	endpoint := firstNonEmpty(w.lastAddress, w.bridgeAddress, w.settings.Address, "mDNS _hue._tcp.local")
	paired := w.settings.AppKey != ""
	var d details
	d.add("Bridge ID", firstNonEmpty(w.bridgeInfo.BridgeID, w.settings.BridgeID))
	d.add("Name", w.bridgeInfo.Name)
	d.add("Software", w.bridgeInfo.Software)
	d.add("API version", w.bridgeInfo.APIVersion)
	d.add("Paired", yesNo(paired))
	stream := "Closed"
	if w.connected {
		stream = "Open"
	}
	d.add("Event stream", stream)
	if w.connects > 1 {
		d.add("Reconnects", strconv.Itoa(w.connects-1))
	}
	if paired {
		d.add("Required", "Yes")
	}
	return snoofer.Control{ID: "hue.app-bridge", Label: "Hue Bridge", Group: "Hue", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: w.bridgeLink.Report(endpoint, d...)}
}

func (w *worker) syncReport(now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionDisconnected, "Unreachable"
	known := w.sync.conn != nil && w.sync.known
	switch {
	case known && w.sync.state.State == syncStateDisconnected:
		state, value = snoofer.ConnectionAttention, "No bridge"
	case known && w.syncing():
		state, value = snoofer.ConnectionConnected, "Syncing"
	case known:
		state, value = snoofer.ConnectionConnected, "Ready"
	case w.sync.connecting || w.sync.conn != nil:
		state, value = snoofer.ConnectionConnecting, "Connecting"
	}
	w.sync.link.Observe(state, now)
	var d details
	if known {
		d.add("App state", value)
		d.add("Mode", syncModeLabels[w.sync.state.Mode])
		d.add("Intensity", syncIntensityLabels[w.sync.state.Intensity])
		if w.sync.state.Bri != nil {
			d.add("Brightness", percentLabel(*w.sync.state.Bri))
		}
	}
	if w.sync.connects > 1 {
		d.add("Reconnects", strconv.Itoa(w.sync.connects-1))
	}
	return snoofer.Control{ID: "hue.app-sync", Label: "Hue Sync", Group: "Hue", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: w.sync.link.Report(w.sync.url, d...)}
}
