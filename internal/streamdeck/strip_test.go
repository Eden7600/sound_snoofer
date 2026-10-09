package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"

	"sound-snoofer/internal/placeholder"
)

func TestStripPreview(t *testing.T) {
	art := testArtwork(t)
	frame := Frame{}
	frame.Dials[0] = Tile{Label: "Playback", Value: "-12.0 dB", Icon: "speaker-mute", Meter: true, LevelKnown: true, LevelDB: -14,
		PeakKnown: true, PeakDB: -9, PositionKnown: true, Position: 0.67, ZeroMark: 0.83}
	frame.Dials[1] = Tile{Label: "Mic", Value: "0.0 dB", Icon: "mic-mute", Meter: true, Status: "Wait"}
	frame.Dials[4] = Tile{Artwork: placeholder.Art("Discord", placeholder.App), Label: "Discord", Value: "42%", Icon: "app-audio"}
	frame.Dials[2] = Tile{Artwork: art, Label: "Lofi beats", Value: "1:42 / 3:30", Icon: "media-play", PositionKnown: true, Position: 0.49}
	frame.Dials[3] = Tile{Timers: [2]TimerTile{{Artwork: art, Label: "Airhorn", Time: "0:03"}}}
	frame.Dials[5] = Tile{Label: "Media", Value: "Home", Icon: "Lights"}
	_, raw := renderFrame(frame)
	rotated, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	bounds := rotated.Bounds()
	strip := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := 0; y < bounds.Dx(); y++ {
		for x := 0; x < bounds.Dy(); x++ {
			strip.Set(x, y, rotated.At(y, bounds.Dy()-1-x))
		}
	}
	// The unknown mic level says so rather than drawing an empty meter.
	lit := 0
	for x := 212; x < 300; x++ {
		for y := 86; y < 100; y++ {
			if r, _, _, _ := strip.At(x, y).RGBA(); r>>8 > 70 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Fatal("unknown level not shown")
	}
	writePreview(t, os.Getenv("SNOOFER_STRIP_PREVIEW"), strip)
}
