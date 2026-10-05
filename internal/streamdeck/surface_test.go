package streamdeck

import (
	"bytes"
	"image/jpeg"
	"testing"
)

func TestGenericTilesAndNavigationDisplay(t *testing.T) {
	frame := Frame{}
	frame.Keys[0] = Tile{Label: "Mic", Value: "On", Icon: "mic-mute"}
	frame.Keys[1] = Tile{Label: "Missing", Value: "Unavailable"}
	frame.Dials[5] = Tile{Label: "Page", Value: "Home"}
	tiles, touch := renderFrame(frame)
	if len(tiles) != Keys {
		t.Fatal(len(tiles))
	}
	for _, data := range tiles {
		im, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil || im.Bounds().Dx() != 112 || im.Bounds().Dy() != 112 {
			t.Fatal("invalid key image", err)
		}
	}
	im, err := jpeg.Decode(bytes.NewReader(touch))
	if err != nil || im.Bounds().Dx() != 100 || im.Bounds().Dy() != 1200 {
		t.Fatal("invalid dial strip", err)
	}
	next := frame
	next.Keys[1].Value = "On"
	updated, _ := renderFrame(next)
	if !bytes.Equal(tiles[0], updated[0]) || bytes.Equal(tiles[1], updated[1]) {
		t.Fatal("unrelated tile changed or availability invisible")
	}
}
func FuzzHIDDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1, 3, 7, 0, 1, 1, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) { d := Decoder{}; _, _ = d.Decode(b) })
}
