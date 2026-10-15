package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"
)

func TestMeetControlPreview(t *testing.T) {
	examples := []struct{ label, value, icon string }{
		{"Mute", "Off", "mic-mute"},
		{"Mute", "On", "mic-mute-muted"},
		{"Mute", "Wait", "mic-mute"},
		{"Mute", "N/A", "mic-mute"},
		{"Mute", "Error", "mic-mute"},
		{"Call", "Brave · Tab 7", "call"},
		{"Camera", "Off", "discord-video-off"},
		{"Camera", "On", "discord-video"},
		{"Camera", "Wait", "discord-video"},
		{"Camera", "N/A", "discord-video"},
		{"Camera", "Error", "discord-video"},
		{"Call", "Disconnected", "call"},
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 6*120, 2*(112+120)))
	row := 0
	for _, id := range []string{"plus-xl", "plus"} {
		m := model(t, id)
		seen := map[string][][]byte{}
		for n, e := range examples {
			raw, scrolls := renderKey(keyPresentation{Label: e.label, Value: e.value, Icon: e.icon}, keyOutput{size: m.KeySize, turn: m.keyTurn}, 0)
			if scrolls && e.label != "Call" {
				t.Fatalf("%s %s %s unexpectedly scrolls", id, e.label, e.value)
			}
			for _, previous := range seen[e.label] {
				if bytes.Equal(previous, raw) {
					t.Fatalf("%s %s %s indistinguishable from another state", id, e.label, e.value)
				}
			}
			seen[e.label] = append(seen[e.label], raw)
			im, err := jpeg.Decode(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			if im.Bounds().Dx() != m.KeySize || im.Bounds().Dy() != m.KeySize {
				t.Fatal("wrong native key dimensions", im.Bounds())
			}
			for y := 0; y < m.KeySize; y++ {
				for x := 0; x < m.KeySize; x++ {
					px, py := x, y
					if m.keyTurn == turnCounterclockwise {
						px, py = y, m.KeySize-1-x
					}
					sheet.Set(n%6*m.KeySize+x, row+n/6*m.KeySize+y, im.At(px, py))
				}
			}
		}
		row += 2 * m.KeySize
	}
	writePreview(t, os.Getenv("SNOOFER_MEET_CONTROL_PREVIEW"), sheet)
}
