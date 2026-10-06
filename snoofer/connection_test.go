package snoofer

import (
	"testing"
	"time"
)

func TestConnectionReportsAreIsolatedCopies(t *testing.T) {
	controls := NewControls()
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	report := &Connection{State: ConnectionConnected, Endpoint: "172.16.102.3", Since: at, Details: []ConnectionDetail{{"Bridge ID", "a"}}}
	control := Control{ID: "hue.app-bridge", Label: "Hue Bridge", Kind: "connection", Value: "Connected", Available: true, SurfaceOnly: true, Connection: report}
	if err := controls.Publish("hue", []Control{control}, nil); err != nil {
		t.Fatal(err)
	}
	report.Details[0].Value = "mutated by provider"
	report.State = ConnectionError
	first := controls.Snapshot()[0]
	if first.Connection.Details[0].Value != "a" || first.Connection.State != ConnectionConnected {
		t.Fatalf("provider mutation leaked: %+v", first.Connection)
	}
	first.Connection.Details[0].Value = "mutated by reader"
	if controls.Snapshot()[0].Connection.Details[0].Value != "a" {
		t.Fatal("reader mutation leaked into the registry")
	}

	control.Connection = &Connection{State: ConnectionConnected, Endpoint: "172.16.102.3", Since: at, Details: []ConnectionDetail{{"Bridge ID", "a"}}}
	if err := controls.Publish("hue", []Control{control}, nil); err != nil {
		t.Fatal(err)
	}
	if controls.Snapshot()[0].Revision != first.Revision {
		t.Fatal("identical report changed revision")
	}
	control.Connection = &Connection{State: ConnectionConnected, Endpoint: "172.16.102.3", Since: at, LastActivity: at.Add(time.Second), Details: []ConnectionDetail{{"Bridge ID", "a"}}}
	if err := controls.Publish("hue", []Control{control}, nil); err != nil {
		t.Fatal(err)
	}
	if controls.Snapshot()[0].Revision == first.Revision {
		t.Fatal("changed report kept its revision")
	}
}
