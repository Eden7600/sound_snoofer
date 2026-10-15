package artwork

import (
	"image"
	"testing"

	"golang.org/x/image/draw"
)

func TestThumbnailRejectsInvalidInput(t *testing.T) {
	pixel := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	for _, tc := range []struct {
		name   string
		source image.Image
		size   int
		scaler draw.Scaler
	}{
		{"nil source", nil, 64, draw.CatmullRom},
		{"empty bounds", image.NewNRGBA(image.Rectangle{}), 64, draw.CatmullRom},
		{"zero width", image.NewNRGBA(image.Rect(3, 4, 3, 9)), 64, draw.CatmullRom},
		{"zero size", pixel, 0, draw.CatmullRom},
		{"negative size", pixel, -1, draw.CatmullRom},
		{"oversized thumbnail", pixel, 129, draw.CatmullRom},
		{"nil scaler", pixel, 64, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if result, err := Thumbnail(tc.source, tc.size, tc.scaler); err == nil || result != "" {
				t.Fatalf("Thumbnail = %q, %v", result, err)
			}
		})
	}
}
