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
		// These middle-edge pixels held the old perimeter, independent of rotation.
		for _, p := range []image.Point{{3, 56}, {108, 56}, {56, 3}, {56, 108}} {
			r, g, b, _ := im.At(p.X, p.Y).RGBA()
			if r>>8 > 40 || g>>8 > 45 || b>>8 > 50 {
				t.Fatal("key perimeter remains", icon, p)
			}
		}
	}
}
