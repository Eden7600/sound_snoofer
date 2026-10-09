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
	bounds := source.Bounds()
	thumb := image.NewNRGBA(image.Rect(0, 0, coverSize, coverSize))
	longest := max(bounds.Dx(), bounds.Dy())
	width, height := max(1, coverSize*bounds.Dx()/longest), max(1, coverSize*bounds.Dy()/longest)
	x, y := (coverSize-width)/2, (coverSize-height)/2
	draw.CatmullRom.Scale(thumb, image.Rect(x, y, x+width, y+height), source, bounds, draw.Src, nil)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, thumb); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded.Bytes()), nil
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
