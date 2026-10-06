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
