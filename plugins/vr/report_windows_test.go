//go:build windows

package vr

import (
	"testing"
	"time"

	"sound-snoofer/plugins/audio"
	"sound-snoofer/snoofer"
)

func TestSteamVRReport(t *testing.T) {
	var link snoofer.ConnectionTracker
	t0 := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	r := steamVRReport(&link, audio.VRPolicy{Known: true}, t0)
	if r.Kind != "connection" || r.Value != "Not running" || r.Connection.State != snoofer.ConnectionDisconnected || !r.Connection.LastActivity.Equal(t0) {
		t.Fatalf("not running %+v %+v", r, r.Connection)
	}
	r = steamVRReport(&link, audio.VRPolicy{Known: true, Running: true}, t0.Add(time.Second))
	if r.Value != "Running" || r.Connection.State != snoofer.ConnectionConnected || !r.Connection.Since.Equal(t0.Add(time.Second)) {
		t.Fatalf("running %+v", r.Connection)
	}
	link.Fail("process snapshot failed", t0.Add(2*time.Second))
	r = steamVRReport(&link, audio.VRPolicy{Running: true}, t0.Add(2*time.Second))
	if r.Value != "Unknown" || r.Connection.State != snoofer.ConnectionUnknown || r.Connection.LastError != "process snapshot failed" ||
		!r.Connection.LastActivity.Equal(t0.Add(time.Second)) {
		t.Fatalf("unknown %+v", r.Connection)
	}
}
