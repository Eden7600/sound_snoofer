package snoofer

import (
	"context"
	"testing"
	"time"
)

func TestProgressInterpolationAndText(t *testing.T) {
	at := time.Unix(1000, 0)
	p := Progress{Known: true, Playing: true, PositionMs: 60_000, DurationMs: 3_600_000 + 5_000, Rate: 1, At: at}
	if got := p.PositionAt(at.Add(5500 * time.Millisecond)); got != 65_500 {
		t.Fatal(got)
	}
	if got := p.Text(at.Add(5 * time.Second)); got != "1:05 / 1:00:05" {
		t.Fatal(got)
	}
	p.Playing = false
	if got := p.PositionAt(at.Add(time.Minute)); got != 60_000 {
		t.Fatal("paused progress advanced", got)
	}
	p = Progress{Known: true, Playing: true, PositionMs: 9_000, DurationMs: 10_000, Rate: 2, At: at}
	if got := p.PositionAt(at.Add(time.Hour)); got != 10_000 {
		t.Fatal("not clamped", got)
	}
	if got := (Progress{Known: true, PositionMs: 61_000}).Text(at); got != "1:01" {
		t.Fatal(got)
	}
}

func TestProgressKeepsRevision(t *testing.T) {
	c := NewControls()
	publish := func(p Progress) uint64 {
		t.Helper()
		if err := c.Publish("x", []Control{{ID: "x.dial", Value: "Playing", Progress: p, Operations: []string{"adjust"}, Available: true}}, func(context.Context, Request) error { return nil }); err != nil {
			t.Fatal(err)
		}
		return c.Snapshot()[0].Revision
	}
	first := publish(Progress{Known: true, Playing: true, PositionMs: 1000, At: time.Unix(1, 0)})
	if publish(Progress{Known: true, Playing: true, PositionMs: 2000, At: time.Unix(2, 0)}) != first {
		t.Fatal("advancing progress changed the revision")
	}
	if err := c.Dispatch(context.Background(), Request{ID: "x.dial", Revision: first, Operation: "adjust", Delta: 1}); err != nil {
		t.Fatal("input rejected while progress advanced:", err)
	}
}
