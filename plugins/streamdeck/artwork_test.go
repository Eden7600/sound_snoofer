package streamdeck

import (
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestArtworkAt(t *testing.T) {
	c := snoofer.Control{Label: "Clip", Available: true, Artwork: "a", Animation: []snoofer.ArtworkFrame{{Artwork: "a", Delay: 100 * time.Millisecond}, {Artwork: "b", Delay: 50 * time.Millisecond}}}
	base := time.Unix(0, 0).Add(150 * time.Millisecond * 1000) // A whole number of loops.
	for _, step := range []struct {
		after time.Duration
		want  string
	}{{0, "a"}, {99 * time.Millisecond, "a"}, {100 * time.Millisecond, "b"}, {149 * time.Millisecond, "b"}, {150 * time.Millisecond, "a"}} {
		if got := artworkAt(c, base.Add(step.after)); got != step.want {
			t.Fatalf("after %v: %q, want %q", step.after, got, step.want)
		}
	}
	if got := artworkAt(snoofer.Control{Artwork: "still"}, base); got != "still" {
		t.Fatal("static artwork", got)
	}
	tile := bindingTile(Binding{Control: "clip"}, c, true, base.Add(120*time.Millisecond))
	if tile.Artwork != "b" {
		t.Fatal("key does not show the current frame", tile.Artwork)
	}
}

func TestProgressDialTile(t *testing.T) {
	at := time.Unix(5000, 0)
	c := snoofer.Control{Label: "Now playing", ShortLabel: "Song", Available: true, Value: "Playing", Status: "Pending",
		Progress: snoofer.Progress{Known: true, Playing: true, PositionMs: 60_000, DurationMs: 240_000, Rate: 1, At: at}}
	tile := withPosition(bindingTile(Binding{Control: "nowplaying.dial"}, c, true, at.Add(5*time.Second)))
	if tile.Value != "1:05 / 4:00" || tile.Status != "Wait" || !tile.PositionKnown || tile.Label != "Song" {
		t.Fatalf("%+v", tile)
	}
}
