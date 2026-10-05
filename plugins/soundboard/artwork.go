package soundboard

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func clipArtwork(folder, name string, files map[string]string) (string, string) {
	for _, extension := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif"} {
		if file, ok := files[strings.ToLower(name+extension)]; ok {
			artwork, err := loadArtwork(filepath.Join(folder, file))
			if err != nil {
				return "", fmt.Sprintf("%s artwork: %v", name, err)
			}
			return artwork, ""
		}
	}
	return "", ""
}
func loadArtwork(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(file, 8*1024*1024+1))
	closeErr := file.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if len(data) > 8*1024*1024 {
		return "", fmt.Errorf("image exceeds 8 MiB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	if format != "png" && format != "jpeg" && format != "webp" && format != "gif" {
		return "", fmt.Errorf("use PNG, JPEG, WebP or GIF")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 {
		return "", fmt.Errorf("image dimensions must be 1–4096px")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	if format == "gif" {
		// Decode returns the first frame, which can occupy only part of the canvas.
		canvas := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
		draw.Draw(canvas, source.Bounds(), source, source.Bounds().Min, draw.Src)
		source = canvas
	}
	// ponytail: thumbnails are regenerated every folder scan; cache by file stat
	// if large catalogues make scanning measurably slow.
	thumb := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	longest := max(cfg.Width, cfg.Height)
	width, height := max(1, 64*cfg.Width/longest), max(1, 64*cfg.Height/longest)
	x, y := (64-width)/2, (64-height)/2
	draw.ApproxBiLinear.Scale(thumb, image.Rect(x, y, x+width, y+height), source, source.Bounds(), draw.Src, nil)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, thumb); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded.Bytes()), nil
}
