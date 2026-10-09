package streamdeck

import (
	"bytes"
	"image/color"
	"image/jpeg"
	"testing"
	"time"
)

func TestScrollOffsetHoldsMovesAndReturns(t *testing.T) {
	const distance = 60 // Two seconds of travel at 30 px/s.
	for _, c := range []struct {
		phase time.Duration
		want  int
	}{
		{0, 0},
		{scrollHold - time.Millisecond, 0},
		{scrollHold + time.Second, 30},
		{scrollHold + 2*time.Second, distance},
		{2*scrollHold + 2*time.Second - time.Millisecond, distance},
		{2*scrollHold + 2*time.Second, 0}, // Back to the start.
	} {
		if got := scrollOffset(c.phase, distance); got != c.want {
			t.Errorf("phase %v: offset %d, want %d", c.phase, got, c.want)
		}
	}
	if scrollOffset(5*time.Second, 0) != 0 {
		t.Fatal("text that fits moved")
	}
}

const longTitle = "Midnight City (Eric Prydz Private Remix) - Extended"

func TestLongTitleScrollsInsideItsBox(t *testing.T) {
	art := testArtwork(t)
	start, scrolls := drawKey([]string{longTitle, "Paused"}, 112, 112, false, "media-play", false, false, art, 0)
	if !scrolls {
		t.Fatal("long title does not scroll")
	}
	later, _ := drawKey([]string{longTitle, "Paused"}, 112, 112, false, "media-play", false, false, art, scrollHold+2*time.Second)
	if bytes.Equal(start, later) {
		t.Fatal("scrolling title did not move")
	}
	if _, scrolls := drawKey([]string{"Lofi beats", "Paused"}, 112, 112, false, "media-play", false, false, art, 0); scrolls {
		t.Fatal("a title that fits scrolls")
	}
	// Mid-scroll, the label row stays clear of the corner inset.
	im, err := jpeg.Decode(bytes.NewReader(later))
	if err != nil {
		t.Fatal(err)
	}
	for y := bandHeight + 2; y < bandHeight+22; y++ {
		for _, x := range []int{0, 2, 4, 6, 106, 108, 110} {
			// Undo the hardware rotation: key (x, y) is stored at (y, 111-x).
			r, g, b, _ := im.At(y, 111-x).RGBA()
			if !near(rgba(r, g, b), keyColor, 30) {
				t.Fatalf("scrolling text drawn at (%d, %d)", x, y)
			}
		}
	}
}

func TestOnlyScrollingKeysRedraw(t *testing.T) {
	frame := Frame{}
	frame.Keys[0] = Tile{Label: longTitle, Value: "Paused", Icon: "media-play"}
	frame.Keys[1] = Tile{Label: "Mic", Value: "Live", Icon: "mic-mute"}
	cache := &keyCache{}
	first, _, scrolling := renderChangedFrame(frame, cache, 0)
	if !scrolling {
		t.Fatal("frame with a long title reports no scrolling")
	}
	second, _, _ := renderChangedFrame(frame, cache, scrollHold+time.Second)
	if &first[1][0] != &second[1][0] {
		t.Fatal("static key redrew while another scrolled")
	}
	if bytes.Equal(first[0], second[0]) {
		t.Fatal("scrolling key did not redraw")
	}
	frame.Keys[0].Label = "Lofi beats"
	if _, _, scrolling := renderChangedFrame(frame, cache, 0); scrolling {
		t.Fatal("scrolling reported when everything fits")
	}
}

func TestMediaDialTitleScrolls(t *testing.T) {
	var knobs [Encoders]knobPresentation
	knobs[0] = knobPresentation{Artwork: testArtwork(t), Target: longTitle, Value: "1:42 / 3:30", Icon: "media-play"}
	if _, scrolls := renderKnobs(knobs, 0); !scrolls {
		t.Fatal("long dial title does not scroll")
	}
	knobs[0].Target = "Lofi beats"
	if _, scrolls := renderKnobs(knobs, 0); scrolls {
		t.Fatal("short dial title scrolls")
	}
}

// rgba narrows 16-bit colour components.
func rgba(r, g, b uint32) color.RGBA {
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
}
