package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestBorderlessDistinctSignalIcons(t *testing.T) {
	var previous [][]byte
	for _, icon := range []string{"mode-direct", "mode-element", "tap-pre", "tap-post", "mic-stack"} {
		canvas := image.NewRGBA(image.Rect(0, 0, 448, 448))
		if !drawIcon(canvas, icon, color.RGBA{255, 255, 255, 255}) {
			t.Fatal("missing icon", icon)
		}
		for _, other := range previous {
			if bytes.Equal(canvas.Pix, other) {
				t.Fatal("indistinguishable icons", icon)
			}
		}
		previous = append(previous, append([]byte(nil), canvas.Pix...))
		raw := render([]string{"Mic processing", "Direct"}, 112, 112, true, icon, false)
		im, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		// No perimeter: middle-edge pixels match the key's own fill, sampled
		// clear of text and icon (key pixel (100, 60), rotated).
		fr, fg, fb, _ := im.At(60, 11).RGBA()
		for _, p := range []image.Point{{3, 56}, {108, 56}, {56, 3}, {56, 108}} {
			r, g, b, _ := im.At(p.X, p.Y).RGBA()
			if abs(int(r>>8)-int(fr>>8)) > 12 || abs(int(g>>8)-int(fg>>8)) > 12 || abs(int(b>>8)-int(fb>>8)) > 12 {
				t.Fatal("key perimeter differs from its fill", icon, p)
			}
		}
	}
}
