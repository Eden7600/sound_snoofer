package soundboard

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"sound-snoofer/snoofer"
)

const (
	maxFrames       = 120                    // Longer GIFs stay static.
	maxAnimatedArea = 1024 * 1024            // Larger GIF canvases stay static, bounding decoded frames.
	defaultDelay    = 100 * time.Millisecond // Browsers' delay for GIF frames of 10 ms or less.
)

// artwork is one decoded image file. Err is a note for the clip; Image may
// still be set when only the animation was dropped.
type artwork struct {
	Image     string
	Animation []snoofer.ArtworkFrame
	Err       string
	path      string
	size      int64
	modified  time.Time
}

// artworkCache keeps decoded artwork by path while the file's size and
// modification time are unchanged. The scanning goroutine owns it.
type artworkCache map[string]artwork

func clipArtwork(folder, name string, files map[string]string, cache artworkCache) artwork {
	for _, extension := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif"} {
		file, ok := files[strings.ToLower(name+extension)]
		if !ok {
			continue
		}
		path := filepath.Join(folder, file)
		info, err := os.Stat(path)
		if err != nil {
			return artwork{Err: fmt.Sprintf("%s artwork: %v", name, err), path: path}
		}
		if cached, ok := cache[path]; ok && cached.size == info.Size() && cached.modified.Equal(info.ModTime()) {
			return cached
		}
		image, animation, err := loadArtwork(path)
		result := artwork{Image: image, Animation: animation, path: path, size: info.Size(), modified: info.ModTime()}
		if err != nil {
			result.Err = fmt.Sprintf("%s artwork: %v", name, err)
		}
		if cache != nil {
			cache[path] = result
		}
		return result
	}
	return artwork{}
}

// loadArtwork returns a thumbnail and, for an animated GIF, its frames. A GIF
// shown still returns its thumbnail with an error explaining why.
func loadArtwork(path string) (string, []snoofer.ArtworkFrame, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, 8*1024*1024+1))
	closeErr := file.Close()
	if err != nil {
		return "", nil, err
	}
	if closeErr != nil {
		return "", nil, closeErr
	}
	if len(data) > 8*1024*1024 {
		return "", nil, fmt.Errorf("image exceeds 8 MiB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	if format != "png" && format != "jpeg" && format != "webp" && format != "gif" {
		return "", nil, fmt.Errorf("use PNG, JPEG, WebP or GIF")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 {
		return "", nil, fmt.Errorf("image dimensions must be 1–4096px")
	}
	if format == "gif" {
		return gifArtwork(data, cfg)
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	thumb, err := thumbnail(source)
	return thumb, nil, err
}

// gifArtwork composites each frame onto the logical canvas, honouring
// disposal, and thumbnails it.
func gifArtwork(data []byte, cfg image.Config) (string, []snoofer.ArtworkFrame, error) {
	canvas := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	if cfg.Width*cfg.Height > maxAnimatedArea {
		first, err := gif.Decode(bytes.NewReader(data))
		if err != nil {
			return "", nil, err
		}
		draw.Draw(canvas, first.Bounds(), first, first.Bounds().Min, draw.Over)
		thumb, err := thumbnail(canvas)
		if err != nil {
			return "", nil, err
		}
		return thumb, nil, fmt.Errorf("animation over 1024×1024px; shown still")
	}
	// ponytail: DecodeAll holds every paletted frame; the canvas and 8 MiB file
	// limits bound typical files, not pathological compression.
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	still := len(g.Image) > maxFrames
	if still {
		g.Image = g.Image[:1]
	}
	var frames []snoofer.ArtworkFrame
	for n, frame := range g.Image {
		var disposal byte
		if n < len(g.Disposal) {
			disposal = g.Disposal[n]
		}
		var previous []byte
		if disposal == gif.DisposalPrevious {
			previous = bytes.Clone(canvas.Pix)
		}
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		thumb, err := thumbnail(canvas)
		if err != nil {
			return "", nil, err
		}
		delay := defaultDelay
		if n < len(g.Delay) && g.Delay[n] > 1 {
			delay = time.Duration(g.Delay[n]) * 10 * time.Millisecond
		}
		frames = append(frames, snoofer.ArtworkFrame{Artwork: thumb, Delay: delay})
		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			copy(canvas.Pix, previous)
		}
	}
	if still {
		return frames[0].Artwork, nil, fmt.Errorf("animation over %d frames; shown still", maxFrames)
	}
	if len(frames) < 2 {
		return frames[0].Artwork, nil, nil
	}
	return frames[0].Artwork, frames, nil
}

// thumbnail fits an image proportionally inside a 64px square PNG.
func thumbnail(source image.Image) (string, error) {
	bounds := source.Bounds()
	thumb := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	longest := max(bounds.Dx(), bounds.Dy())
	width, height := max(1, 64*bounds.Dx()/longest), max(1, 64*bounds.Dy()/longest)
	x, y := (64-width)/2, (64-height)/2
	draw.ApproxBiLinear.Scale(thumb, image.Rect(x, y, x+width, y+height), source, bounds, draw.Src, nil)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, thumb); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded.Bytes()), nil
}
