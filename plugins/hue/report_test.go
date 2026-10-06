package hue

import (
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func detailValue(c *snoofer.Connection, label string) string {
	for _, d := range c.Details {
		if d.Label == label {
			return d.Value
		}
	}
	return ""
}

func TestBridgeReport(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	report := h.waitControl("hue.app-bridge", func(c snoofer.Control) bool {
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionConnected
	})
	conn := report.Connection
	if report.Kind != "connection" || !report.SurfaceOnly || len(report.Operations) != 0 || report.Status != "" {
		t.Fatalf("report control %+v", report)
	}
	if conn.Endpoint != bridge.address() || detailValue(conn, "Bridge ID") != "b1" || detailValue(conn, "Software") != "1978293000" ||
		detailValue(conn, "Event stream") != "Open" || detailValue(conn, "Required") != "Yes" || conn.Since.IsZero() || conn.LastActivity.IsZero() {
		t.Fatalf("connection %+v", conn)
	}
	for _, d := range conn.Details {
		if strings.Contains(d.Value, bridge.key) {
			t.Fatal("application key in report")
		}
	}
	waitFor(t, func() bool { return bridge.streamCount() == 1 })
	bridge.closeStreams()
	recovered := h.waitControl("hue.app-bridge", func(c snoofer.Control) bool {
		return c.Connection.State == snoofer.ConnectionConnected && detailValue(c.Connection, "Reconnects") == "1"
	})
	if !strings.Contains(recovered.Connection.LastError, "event stream ended") || !recovered.Connection.Since.After(conn.Since) {
		t.Fatalf("recovery lost the error or since: %+v", recovered.Connection)
	}
}

func TestUnpairedBridgeReport(t *testing.T) {
	bridge := newFakeBridge(t, "b1")
	h := startHarness(t, bridge, Settings{}, true)
	report := h.waitControl("hue.app-bridge", func(c snoofer.Control) bool {
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionUnconfigured
	})
	if report.Value != "Not paired" || report.Connection.Endpoint != bridge.address() || detailValue(report.Connection, "Required") != "" {
		t.Fatalf("unpaired report %+v %+v", report, report.Connection)
	}
}

func TestSyncReport(t *testing.T) {
	h, _, app := startJoined(t, syncStateSyncing, true)
	report := h.waitControl("hue.app-sync", func(c snoofer.Control) bool {
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionConnected
	})
	if report.Value != "Syncing" || detailValue(report.Connection, "Mode") != "Video" || detailValue(report.Connection, "Brightness") != "50%" ||
		!strings.HasPrefix(report.Connection.Endpoint, "ws://127.0.0.1:") {
		t.Fatalf("sync report %+v %+v", report, report.Connection)
	}
	app.drop()
	h.waitControl("hue.app-sync", func(c snoofer.Control) bool { return c.Connection.LastError != "" })
	h.waitControl("hue.app-sync", func(c snoofer.Control) bool {
		return c.Connection.State == snoofer.ConnectionConnected && detailValue(c.Connection, "Reconnects") == "1"
	})
}

func TestSyncUnreachableReport(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	report := h.waitControl("hue.app-sync", func(c snoofer.Control) bool {
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionDisconnected && c.Connection.LastError != ""
	})
	if report.Value != "Unreachable" || !strings.Contains(report.Connection.LastError, "Third-party control") {
		t.Fatalf("unreachable report %+v %+v", report, report.Connection)
	}
}
