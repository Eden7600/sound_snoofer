package snoofer

import (
	"context"
	"testing"
	"time"
)

func TestMeterDoesNotInvalidateInput(t *testing.T) {
	r := NewControls()
	c := Control{ID: "audio.gain", Available: true, Operations: []string{"adjust"}, Value: "0 dB", Meter: Meter{Present: true, Known: true, DB: -30, At: time.Now()}}
	invoked := false
	publish := func() {
		t.Helper()
		if err := r.Publish("audio", []Control{c}, func(context.Context, Request) error { invoked = true; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	publish()
	revision := r.Snapshot()[0].Revision
	c.Meter.DB = -6
	c.Meter.At = time.Now()
	publish()
	if r.Snapshot()[0].Revision != revision || r.Snapshot()[0].Meter.DB != -6 {
		t.Fatal("telemetry changed identity or was lost")
	}
	req := Request{ID: c.ID, Revision: revision, Operation: "adjust", Delta: 1}
	if err := r.Dispatch(context.Background(), req); err != nil || !invoked {
		t.Fatal(err)
	}
	c.Value = "1 dB"
	publish()
	if err := r.Dispatch(context.Background(), req); err == nil {
		t.Fatal("real control changes must invalidate stale input")
	}
}
