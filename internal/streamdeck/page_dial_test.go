package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestPageDialThreeLines(t *testing.T) {
	frame := Frame{}
	frame.Dials[5] = Tile{Label: "Home", Value: "Soundboard", Icon: "Soundboard 2"}
	_, raw := renderFrame(frame)
	rotated, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	panel := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			panel.Set(x, y, rotated.At(y, 199-x))
		}
	}
	for _, row := range []struct {
		start, end int
		active     bool
	}{{12, 19, false}, {42, 56, true}, {76, 83, false}} {
		bright := 0
		for y := row.start; y < row.end; y++ {
			for x := 8; x < 192; x++ {
				r, g, b, _ := panel.At(x, y).RGBA()
				if g > 18000 && b > 18000 {
					bright++
					if row.active && b <= r {
						t.Fatal("current page lost active color")
					}
				}
			}
		}
		if bright == 0 {
			t.Fatalf("missing row at %d", row.start)
		}
	}
	if path := os.Getenv("SNOOFER_PAGE_DIAL_PREVIEW"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, panel)
		closeErr := f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
}
