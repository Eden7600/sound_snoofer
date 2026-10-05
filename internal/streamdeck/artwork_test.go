package streamdeck

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func testArtwork(t *testing.T) string {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			c := color.NRGBA{R: 230, G: 120, B: 50, A: 255}
			if x >= 32 {
				c = color.NRGBA{R: 40, G: 150, B: 180, A: 255}
			}
			if y >= 32 {
				c.A = 100
			}
			im.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestArtworkRenderingAndCacheInvalidation(t *testing.T) {
	frame := Frame{}
	frame.Keys[0] = Tile{Label: "Clip", Value: "Playing", Icon: "soundboard-play"}
	before, _ := renderFrame(frame)
	next := frame
	next.Keys[0].Artwork = testArtwork(t)
	after, _ := renderChangedFrame(next, frame, before)
	if bytes.Equal(before[0], after[0]) {
		t.Fatal("artwork did not invalidate cache")
	}
	im, err := jpeg.Decode(bytes.NewReader(after[0]))
	if err != nil {
		t.Fatal(err)
	}
	// Renderer rotates: inspect the upper-left quadrant of the central artwork.
	r, g, b, _ := im.At(30, 111-30).RGBA()
	if r <= g || g <= b {
		t.Fatal("thumbnail pixels missing")
	}
	again, _ := renderChangedFrame(next, next, after)
	if &again[0][0] != &after[0][0] {
		t.Fatal("unchanged artwork was rerendered")
	}
	bad := next
	bad.Keys[0].Artwork = "invalid"
	fallback, _ := renderChangedFrame(bad, next, after)
	if !bytes.Equal(fallback[0], before[0]) {
		t.Fatal("invalid artwork did not restore play icon")
	}
}
