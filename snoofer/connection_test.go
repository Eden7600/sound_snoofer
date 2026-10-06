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

func TestConnectionTrackerSinceAndRetainedError(t *testing.T) {
	var tracker ConnectionTracker
	t0 := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	tracker.Observe(ConnectionConnected, t0)
	tracker.Observe(ConnectionConnected, t0.Add(time.Minute))
	if r := tracker.Report("x"); !r.Since.Equal(t0) || r.State != ConnectionConnected {
		t.Fatalf("since moved without a state change: %+v", r)
	}
	tracker.Fail("refused", t0.Add(2*time.Minute))
	tracker.Observe(ConnectionDisconnected, t0.Add(2*time.Minute))
	tracker.Fail("", t0.Add(3*time.Minute))
	tracker.Observe(ConnectionConnected, t0.Add(4*time.Minute))
	tracker.Activity(t0.Add(5 * time.Minute))
	r := tracker.Report("x", ConnectionDetail{"Mode", "video"})
	if !r.Since.Equal(t0.Add(4*time.Minute)) || r.LastError != "refused" || !r.LastErrorAt.Equal(t0.Add(2*time.Minute)) || !r.LastActivity.Equal(t0.Add(5*time.Minute)) || r.Details[0].Value != "video" {
		t.Fatalf("report %+v", r)
	}
}

func TestConnectionTrackerPersistentFailureKeepsFirstTime(t *testing.T) {
	var tracker ConnectionTracker
	t0 := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	tracker.Observe(ConnectionDisconnected, t0)
	tracker.Fail("refused", t0)
	tracker.Fail("refused", t0.Add(time.Minute))
	if r := tracker.Report(""); !r.LastErrorAt.Equal(t0) {
		t.Fatalf("persistent failure moved: %v", r.LastErrorAt)
	}
	tracker.Fail("timeout", t0.Add(2*time.Minute))
	if r := tracker.Report(""); r.LastError != "timeout" || !r.LastErrorAt.Equal(t0.Add(2*time.Minute)) {
		t.Fatalf("new failure not recorded: %+v", r)
	}
	tracker.Observe(ConnectionConnected, t0.Add(3*time.Minute))
	tracker.Fail("timeout", t0.Add(4*time.Minute))
	if r := tracker.Report(""); !r.LastErrorAt.Equal(t0.Add(4 * time.Minute)) {
		t.Fatalf("recurrence after recovery not recorded: %v", r.LastErrorAt)
	}
}
