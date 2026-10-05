package soundboard

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestMatchingArtworkRefreshAndFallback(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	encode := func(format string, width, height int, c color.NRGBA) []byte {
		t.Helper()
		im := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				im.SetNRGBA(x, y, c)
			}
		}
		var b bytes.Buffer
		var err error
		if format == "png" {
			err = png.Encode(&b, im)
		} else {
			err = jpeg.Encode(&b, im, nil)
		}
		if err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	get := func() clip {
		t.Helper()
		clips, err := catalogue(dir)
		if err != nil || len(clips) != 1 {
			t.Fatalf("%v %v", clips, err)
		}
		return clips[0]
	}
	write("Sound.mp3", []byte("clip"))
	if get().Artwork != "" {
		t.Fatal("missing image did not fall back")
	}
	write("sOuNd.JPEG", encode("jpeg", 100, 100, color.NRGBA{R: 255, A: 255}))
	jpegClip := get()
	if jpegClip.Artwork == "" || jpegClip.ArtworkError != "" {
		t.Fatal("JPEG not loaded", jpegClip.ArtworkError)
	}
	write("sound.PNG", encode("png", 128, 128, color.NRGBA{G: 255, A: 255}))
	pngClip := get()
	if pngClip.Artwork == jpegClip.Artwork || pngClip.Artwork == "" {
		t.Fatal("PNG did not take precedence")
	}
	data, err := base64.StdEncoding.DecodeString(pngClip.Artwork)
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil || im.Bounds().Dx() != 64 {
		t.Fatal("thumbnail invalid", err)
	}
	write("sound.PNG", encode("png", 20, 10, color.NRGBA{A: 255}))
	c := get()
	if c.Artwork == "" || c.ArtworkError != "" {
		t.Fatal("rectangle rejected", c.ArtworkError)
	}
	data, err = base64.StdEncoding.DecodeString(c.Artwork)
	if err != nil {
		t.Fatal(err)
	}
	im, err = png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, topAlpha := im.At(32, 0).RGBA()
	_, _, _, centerAlpha := im.At(32, 32).RGBA()
	if topAlpha != 0 || centerAlpha != 65535 {
		t.Fatal("rectangle stretched instead of fitted")
	}
	write("sound.PNG", []byte("not an image"))
	if c := get(); c.Artwork != "" || c.ArtworkError == "" {
		t.Fatal("corrupt image accepted")
	}
	if err := os.Remove(filepath.Join(dir, "sound.PNG")); err != nil {
		t.Fatal(err)
	}
	if get().Artwork != jpegClip.Artwork {
		t.Fatal("removal did not restore JPEG")
	}
	if err := os.Remove(filepath.Join(dir, "sOuNd.JPEG")); err != nil {
		t.Fatal(err)
	}
	if get().Artwork != "" {
		t.Fatal("deleted artwork retained")
	}
}
