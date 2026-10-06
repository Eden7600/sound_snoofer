package media

import (
	"errors"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestMediaKeysReport(t *testing.T) {
	var keys mediaKeys
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	if r := keys.report(true, now); r.Connection.State != snoofer.ConnectionReady || r.Kind != "connection" || len(r.Operations) != 0 {
		t.Fatalf("idle %+v %+v", r, r.Connection)
	}
	keys.sent("media.play", errors.New("media dispatch incomplete"), now.Add(time.Second))
	r := keys.report(true, now.Add(time.Second))
	if r.Connection.State != snoofer.ConnectionError || r.Connection.Details[0].Value != "play" || r.Connection.LastError != "media dispatch incomplete" {
		t.Fatalf("failed %+v", r.Connection)
	}
	keys.sent("media.next", nil, now.Add(2*time.Second))
	r = keys.report(true, now.Add(2*time.Second))
	if r.Connection.State != snoofer.ConnectionReady || r.Connection.LastError == "" || !r.Connection.LastActivity.Equal(now.Add(2*time.Second)) {
		t.Fatalf("recovered %+v", r.Connection)
	}
	if r = keys.report(false, now); r.Connection.State != snoofer.ConnectionOff {
		t.Fatalf("preview %+v", r.Connection)
	}
}
