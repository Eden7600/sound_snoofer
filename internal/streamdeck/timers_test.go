package streamdeck

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestTimerPanelRendering(t *testing.T) {
	art := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			art.Set(x, y, color.NRGBA{R: uint8(x * 4), G: 120, B: uint8(y * 4), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, art); err != nil {
		t.Fatal(err)
	}
	artwork := base64.StdEncoding.EncodeToString(encoded.Bytes())
	frame := Frame{}
	frame.Dials[0] = Tile{Label: "Volume", Value: "0.0 dB"}
	frame.Dials[1].Timers[0] = TimerTile{Artwork: artwork, Label: "emotional-damage", Time: "0:07"}
	frame.Dials[2].Timers = [2]TimerTile{{Icon: "soundboard-play", Label: "bruh", Time: "0:02"}, {Artwork: artwork, Label: "scotland-forever", Time: "12:34"}}
	frame.Dials[3].Timers[0] = TimerTile{Icon: "soundboard-play", Label: "vine-boom", Time: "10:05"}
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
	lit := func(x0, y0, x1, y1 int) bool {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				if r, g, b, _ := strip.At(x, y).RGBA(); r>>8+g>>8+b>>8 > 200 {
					return true
				}
			}
		}
		return false
	}
	if !lit(208, 18, 272, 82) || !lit(284, 26, 380, 54) {
		t.Fatal("single timer lacks artwork or time")
	}
	if !lit(408, 6, 448, 46) || !lit(408, 54, 448, 94) {
		t.Fatal("paired timers lack thumbnails")
	}
	if lit(800, 0, 1000, 100) {
		t.Fatal("unused panel drawn")
	}
	writePreview(t, os.Getenv("SNOOFER_TIMER_PREVIEW"), strip)
}
