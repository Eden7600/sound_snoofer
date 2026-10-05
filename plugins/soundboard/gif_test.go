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
	artwork, diagnostic := clipArtwork(dir, "CLIP", map[string]string{"clip.gif": "clip.gif"})
	if diagnostic != "" {
		t.Fatal(diagnostic)
	}
	raw, err := base64.StdEncoding.DecodeString(artwork)
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
