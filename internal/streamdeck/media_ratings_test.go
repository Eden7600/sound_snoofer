package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"testing"
)

func TestMediaRatingIcons(t *testing.T) {
	var previous []byte
	for _, icon := range []string{"media-like", "media-dislike"} {
		canvas := image.NewRGBA(image.Rect(0, 0, 448, 448))
		if !drawIcon(canvas, icon, lightValue) {
			t.Fatal("missing rating icon", icon)
		}
		if bytes.Equal(canvas.Pix, make([]byte, len(canvas.Pix))) {
			t.Fatal("empty rating icon", icon)
		}
		if bytes.Equal(canvas.Pix, previous) {
			t.Fatal("Like and Dislike must have distinct shapes")
		}
		previous = canvas.Pix
		if keyArea(icon) != mediaArea {
			t.Fatal("rating does not use the Media area", icon)
		}
	}
}

func TestMediaRatingPresentation(t *testing.T) {
	states := []struct {
		value            string
		background, icon color.RGBA
	}{
		{"Off", keyColor, mediaArea},
		{"On", mediaArea, inkColor},
		{"N/A", keyColor, unknownColor},
		{"Wait", keyColor, attentionColor},
		{"Error", keyColor, criticalColor},
	}
	// Two 112px + XL rows, followed by two 120px + rows, without preview scaling.
	sheet := image.NewRGBA(image.Rect(0, 0, len(states)*120, 2*(112+120)))
	row := 0
	for _, id := range []string{"plus-xl", "plus"} {
		m := model(t, id)
		for _, rating := range []struct{ label, icon string }{{"Like", "media-like"}, {"Dislike", "media-dislike"}} {
			for column, state := range states {
				t.Run(id+"/"+rating.label+"/"+state.value, func(t *testing.T) {
					raw, scrolls := renderKey(keyPresentation{Label: rating.label, Value: state.value, Icon: rating.icon},
						keyOutput{size: m.KeySize, turn: m.keyTurn}, 0)
					if scrolls {
						t.Fatal("compact rating key unexpectedly scrolls")
					}
					encoded, err := jpeg.Decode(bytes.NewReader(raw))
					if err != nil {
						t.Fatal(err)
					}
					if encoded.Bounds().Dx() != m.KeySize || encoded.Bounds().Dy() != m.KeySize {
						t.Fatal("wrong native key size", encoded.Bounds())
					}
					key := image.NewRGBA(encoded.Bounds())
					for y := 0; y < m.KeySize; y++ {
						for x := 0; x < m.KeySize; x++ {
							px, py := x, y
							if m.keyTurn == turnCounterclockwise {
								px, py = y, m.KeySize-1-x
							}
							key.Set(x, y, encoded.At(px, py))
						}
					}
					at := func(x, y int) color.RGBA { return key.RGBAAt(x*m.KeySize/112, y*m.KeySize/112) }
					if got := at(100, 60); !near(got, state.background, 12) {
						t.Fatal("wrong rating background", got, state.background)
					}
					foundIcon := false
					for y := iconTop; y < iconTop+iconSize; y++ {
						for x := keyInset; x < keyInset+iconSize; x++ {
							foundIcon = foundIcon || near(at(x, y), state.icon, 35)
						}
					}
					if !foundIcon {
						t.Fatal("rating glyph or state alert missing")
					}
					if state.value == "Wait" && !near(at(4, 2), attentionColor, 35) {
						t.Fatal("pending dashes missing")
					}
					if state.value == "Error" && !near(at(56, 2), criticalColor, 35) {
						t.Fatal("error frame missing")
					}
					draw.Draw(sheet, image.Rect(column*m.KeySize, row, (column+1)*m.KeySize, row+m.KeySize), key, image.Point{}, draw.Src)
				})
			}
			row += m.KeySize
		}
	}
	writePreview(t, os.Getenv("SNOOFER_MEDIA_RATINGS_PREVIEW"), sheet)
}
