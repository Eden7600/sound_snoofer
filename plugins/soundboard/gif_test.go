package soundboard

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGIFUsesFirstFrameOnLogicalCanvas(t *testing.T) {
	palette := color.Palette{color.NRGBA{}, color.NRGBA{R: 255, A: 255}, color.NRGBA{B: 255, A: 255}}
	first := image.NewPaletted(image.Rect(4, 4, 12, 12), palette)
	second := image.NewPaletted(image.Rect(0, 0, 16, 16), palette)
	for n := range first.Pix {
		first.Pix[n] = 1
	}
	for n := range second.Pix {
		second.Pix[n] = 2
	}
	var b bytes.Buffer
	err := gif.EncodeAll(&b, &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{10, 10}, Config: image.Config{ColorModel: palette, Width: 16, Height: 16}})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clip.gif"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	art := clipArtwork(dir, "CLIP", map[string]string{"clip.gif": "clip.gif"}, nil)
	if art.Err != "" {
		t.Fatal(art.Err)
	}
	// Delays of 10 ms play at 100 ms, as in browsers; the first frame is the still.
	if len(art.Animation) != 2 || art.Animation[0].Artwork != art.Image || art.Animation[1].Delay != 100*time.Millisecond {
		t.Fatalf("animation %d frames", len(art.Animation))
	}
	raw, err := base64.StdEncoding.DecodeString(art.Image)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	r, _, blue, a := decoded.At(32, 32).RGBA()
	_, _, _, edgeAlpha := decoded.At(0, 0).RGBA()
	if r != 65535 || blue != 0 || a != 65535 || edgeAlpha != 0 {
		t.Fatal("GIF frame, positioning or transparency lost")
	}
}

// writeGIF stores an animation of solid-colour frames, each covering rect.
func writeGIF(t *testing.T, dir string, rects []image.Rectangle, colours []uint8, disposal []byte, delay []int) {
	t.Helper()
	palette := color.Palette{color.NRGBA{}, color.NRGBA{R: 255, A: 255}, color.NRGBA{B: 255, A: 255}}
	if delay == nil {
		delay = make([]int, len(rects))
	}
	g := &gif.GIF{Config: image.Config{ColorModel: palette, Width: 16, Height: 16}, Disposal: disposal, Delay: delay}
	for n, rect := range rects {
		frame := image.NewPaletted(rect, palette)
		for i := range frame.Pix {
			frame.Pix[i] = colours[n]
		}
		g.Image = append(g.Image, frame)
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "clip.gif"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

// pixel decodes a frame thumbnail and returns the colour at a point.
func pixel(t *testing.T, artwork string, x, y int) color.NRGBA {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(artwork)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return color.NRGBAModel.Convert(decoded.At(x, y)).(color.NRGBA)
}

func TestGIFDisposal(t *testing.T) {
	full, corner := image.Rect(0, 0, 16, 16), image.Rect(0, 0, 8, 8)
	files := map[string]string{"clip.gif": "clip.gif"}
	for _, test := range []struct {
		name     string
		disposal byte
		corner   uint8 // Palette index left in the corner for the third frame; 0 is transparent.
	}{
		{"none keeps the frame", gif.DisposalNone, 2},
		{"background clears it", gif.DisposalBackground, 0},
		{"previous restores the canvas", gif.DisposalPrevious, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			// Red canvas, then blue over the top-left corner with the disposal
			// under test, then red over the bottom-right corner.
			writeGIF(t, dir, []image.Rectangle{full, corner, image.Rect(8, 8, 16, 16)}, []uint8{1, 2, 1},
				[]byte{gif.DisposalNone, test.disposal, gif.DisposalNone}, []int{5, 5, 5})
			art := clipArtwork(dir, "clip", files, nil)
			if art.Err != "" || len(art.Animation) != 3 || art.Animation[0].Delay != 50*time.Millisecond {
				t.Fatalf("animation %+v", art.Err)
			}
			got := pixel(t, art.Animation[2].Artwork, 16, 16)
			want := map[uint8]color.NRGBA{0: {}, 1: {R: 255, A: 255}, 2: {B: 255, A: 255}}[test.corner]
			if got != want {
				t.Fatalf("corner %+v, want %+v", got, want)
			}
		})
	}
}

func TestGIFLimitsAndCache(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"clip.gif": "clip.gif"}
	rects := make([]image.Rectangle, maxFrames+1)
	colours := make([]uint8, maxFrames+1)
	for n := range rects {
		rects[n] = image.Rect(0, 0, 16, 16)
		colours[n] = uint8(1 + n%2)
	}
	writeGIF(t, dir, rects, colours, nil, nil)
	long := clipArtwork(dir, "clip", files, nil)
	if long.Image == "" || long.Animation != nil || long.Err != "clip artwork: animation over 120 frames; shown still" {
		t.Fatalf("long animation %q, %d frames", long.Err, len(long.Animation))
	}

	writeGIF(t, dir, rects[:1], colours[:1], nil, nil)
	cache := artworkCache{}
	still := clipArtwork(dir, "clip", files, cache)
	if still.Image == "" || still.Animation != nil || still.Err != "" {
		t.Fatal("single frame GIF animated")
	}
	// An unchanged file is served from cache even if it can no longer be read.
	path := filepath.Join(dir, "clip.gif")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	cached := cache[path]
	cached.Image = "cached"
	cache[path] = cached
	if clipArtwork(dir, "clip", files, cache).Image != "cached" {
		t.Fatal("unchanged artwork decoded again")
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if clipArtwork(dir, "clip", files, cache).Image == "cached" {
		t.Fatal("changed artwork served from cache")
	}
}
