package nowplaying

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"sound-snoofer/internal/artwork"
)

// thumbnail fits cover art proportionally inside a coverSize PNG, the full
// artwork contract, so covers stay sharp on the deck. Large or unreadable images are rejected.
func thumbnail(data []byte) (string, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 {
		return "", fmt.Errorf("artwork dimensions must be 1–4096px")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return artwork.Thumbnail(source, coverSize, draw.CatmullRom)
}

// coverSize is the cover thumbnail edge: the artwork contract's 128 px.
const coverSize = 128

// validArtwork accepts only the artwork contract (a square PNG up to
// coverSize) from the extension.
func validArtwork(art string) bool {
	if art == "" {
		return true
	}
	data, err := base64.StdEncoding.DecodeString(art)
	if err != nil {
		return false
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	return err == nil && cfg.Width == cfg.Height && cfg.Width >= 1 && cfg.Width <= coverSize
}
