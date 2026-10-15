package nowplaying

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
		{"wide", 7, 5, "a3bebc3df4bfe704dcb46b5ec2e7a68d194c7a2aa094423131f6adb62f0fca3b"},
		{"tall", 5, 7, "b57d452b889e1d0a35eb01c8491df5195d93bbdb2061991ac57dc666de215998"},
		{"thin", 1, 13, "4e1ff1702bc5b97ec196a8b3a935a7a5d6b3e71779c58f9ed0ac439fa5d1c25c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := image.NewNRGBA(image.Rect(11, 7, 11+tc.width, 7+tc.height))
			for y := range tc.height {
				for x := range tc.width {
					source.SetNRGBA(x+11, y+7, color.NRGBA{R: uint8(x*37 + y*11), G: uint8(x*13 + y*43), B: uint8(x*59 + y*7), A: uint8(80 + (x*19+y*23)%176)})
				}
			}
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, source); err != nil {
				t.Fatal(err)
			}
			data := encoded.Bytes()
			before := bytes.Clone(data)
			art, err := thumbnail(data)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, before) {
				t.Fatal("thumbnail changed encoded source")
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
			if !ok || pixels.Bounds() != image.Rect(0, 0, 128, 128) {
				t.Fatalf("unexpected thumbnail %T %v", decoded, decoded.Bounds())
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(pixels.Pix)); got != tc.want {
				t.Errorf("pixels %s; want %s", got, tc.want)
			}
		})
	}
}

func TestThumbnailInputLimits(t *testing.T) {
	if art, err := thumbnail([]byte("not an image")); art != "" || err == nil {
		t.Fatalf("corrupt artwork = %q, %v", art, err)
	}
	for _, tc := range []struct {
		name          string
		width, height int
		invalid       bool
	}{
		{"maximum width", 4096, 1, false},
		{"maximum height", 1, 4096, false},
		{"excessive width", 4097, 1, true},
		{"excessive height", 1, 4097, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, tc.width, tc.height))); err != nil {
				t.Fatal(err)
			}
			art, err := thumbnail(encoded.Bytes())
			if tc.invalid {
				if err == nil || art != "" {
					t.Fatalf("invalid artwork accepted: %v", err)
				}
			} else if err != nil || art == "" || !validArtwork(art) {
				t.Fatalf("valid artwork rejected: %v", err)
			}
		})
	}
}
