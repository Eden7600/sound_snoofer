// Package artwork fits decoded images to the control thumbnail contract.
package artwork

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"

	"golang.org/x/image/draw"
)

// Thumbnail fits source proportionally inside a transparent square PNG and
// returns its base64 encoding. Size is 1–128 pixels; callers own input decoding
// and limits and choose the interpolation policy.
func Thumbnail(source image.Image, size int, scaler draw.Scaler) (string, error) {
	if source == nil || size < 1 || size > 128 || scaler == nil {
		return "", errors.New("invalid thumbnail source, size or scaler")
	}
	bounds := source.Bounds()
	if bounds.Empty() {
		return "", errors.New("thumbnail source has empty bounds")
	}
	thumb := image.NewNRGBA(image.Rect(0, 0, size, size))
	longest := max(bounds.Dx(), bounds.Dy())
	width, height := max(1, size*bounds.Dx()/longest), max(1, size*bounds.Dy()/longest)
	x, y := (size-width)/2, (size-height)/2
	scaler.Scale(thumb, image.Rect(x, y, x+width, y+height), source, bounds, draw.Src, nil)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, thumb); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded.Bytes()), nil
}
