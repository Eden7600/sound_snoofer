package soundboard

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// Pixel fingerprints captured before extracting the shared fit/encode implementation.
func TestThumbnailPixels(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		want          string
	}{
		{"wide", 7, 5, "9cac540b293e4fd4cb02b5a20a0a96278567f8b60300a76809ccf2a417ad2c53"},
		{"tall", 5, 7, "5b4b5d94654f552827cca562b1e525cfb425c65a010b54a9f7aebd615bad6b46"},
		{"thin", 1, 13, "72a3c7c987d16cf06b089a1cff2b1b3eaf11a3c8da01f595000d820612bedd39"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := image.NewNRGBA(image.Rect(11, 7, 11+tc.width, 7+tc.height))
			for y := range tc.height {
				for x := range tc.width {
					source.SetNRGBA(x+11, y+7, color.NRGBA{R: uint8(x*37 + y*11), G: uint8(x*13 + y*43), B: uint8(x*59 + y*7), A: uint8(80 + (x*19+y*23)%176)})
				}
			}
			before := bytes.Clone(source.Pix)
			art, err := thumbnail(source)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(source.Pix, before) {
				t.Fatal("thumbnail changed source pixels")
			}
			raw, err := base64.StdEncoding.DecodeString(art)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := png.Decode(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			pixels, ok := decoded.(*image.NRGBA)
			if !ok || pixels.Bounds() != image.Rect(0, 0, 64, 64) {
				t.Fatalf("unexpected thumbnail %T %v", decoded, decoded.Bounds())
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(pixels.Pix)); got != tc.want {
				t.Errorf("pixels %s; want %s", got, tc.want)
			}
		})
	}
}
