package placeholder

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"os"
	"testing"
)

func TestArtIsDeterministicAndDistinct(t *testing.T) {
	if Art("Discord", App) != Art(" discord ", App) {
		t.Fatal("same name, different art")
	}
	if Art("Discord", App) == Art("Discord", Media) {
		t.Fatal("glyph ignored")
	}
	hues := map[float64]string{}
	for _, name := range []string{"Discord", "Steam", "Game", "Launcher", "Brave · youtube.com", "Brave · twitch.tv", "Spotify", "VLC"} {
		hues[Hue(name)] = name
	}
	if len(hues) < 7 {
		t.Fatal("names share colours too often", hues)
	}
	data, err := base64.StdEncoding.DecodeString(Art("Steam", App))
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil || im.Bounds().Dx() != 64 || im.Bounds().Dy() != 64 {
		t.Fatal("not a 64 px PNG", err)
	}
	if _, _, _, a := im.At(0, 0).RGBA(); a != 0 {
		t.Fatal("corner not transparent")
	}
	if _, _, _, a := im.At(32, 50).RGBA(); a == 0 {
		t.Fatal("tile not filled")
	}
	if path := os.Getenv("SNOOFER_PLACEHOLDER_PREVIEW"); path != "" {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
