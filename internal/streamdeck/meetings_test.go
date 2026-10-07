package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
)

func TestMeetingsPresentation(t *testing.T) {
	icons := []string{"camera-privacy", "camera-privacy-muted", "tracking", "framing", "camera-reset", "discord-deafen", "discord-deafen-muted",
		"discord-video", "discord-video-off", "screen-share", "call-leave", "discord-connect"}
	drawn := map[string][]byte{}
	for _, icon := range icons {
		canvas := image.NewRGBA(image.Rect(0, 0, 448, 448))
		if !drawIcon(canvas, icon, color.RGBA{255, 255, 255, 255}) {
			t.Fatal("missing icon", icon)
		}
		drawn[icon] = canvas.Pix
	}
	// Each meeting symbol is distinct; the connect icon reuses the link rings.
	unique := []string{"camera-privacy", "tracking", "framing", "camera-reset", "screen-share", "call-leave"}
	for i, a := range unique {
		for _, b := range unique[i+1:] {
			if bytes.Equal(drawn[a], drawn[b]) {
				t.Fatal("indistinguishable icons", a, b)
			}
		}
	}
	// State is carried by shape, not only color: hidden, deafened and off are slashed.
	for plain, slashed := range map[string]string{"camera-privacy": "camera-privacy-muted", "discord-deafen": "discord-deafen-muted", "discord-video": "discord-video-off"} {
		if bytes.Equal(drawn[plain], drawn[slashed]) {
			t.Fatal("state not visible in shape", slashed)
		}
	}
	if keyAccent("ON", "camera-privacy-muted", false, false) != criticalColor || keyAccent("OFF", "discord-video-off", false, false) != neutralColor {
		t.Fatal("privacy reads like a mute; camera off stays neutral")
	}

	examples := []struct{ label, value, icon string }{
		{"Privacy", "Off", "camera-privacy"},
		{"Privacy", "On", "camera-privacy-muted"},
		{"Tracking", "Group", "tracking"},
		{"Framing", "Half body", "framing"},
		{"Reset", "", "camera-reset"},
		{"Mute", "On", "mic-mute-muted"},
		{"Deafen", "On", "discord-deafen-muted"},
		{"Camera", "Off", "discord-video-off"},
		{"Share", "On", "screen-share"},
		{"Leave", "", "call-leave"},
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 112*len(examples), 112))
	for n, e := range examples {
		raw := renderArtwork([]string{e.label, e.value}, 112, 112, false, e.icon, false, "")
		im, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		// Undo the hardware rotation for the review sheet.
		for y := 0; y < 112; y++ {
			for x := 0; x < 112; x++ {
				sheet.Set(n*112+x, y, im.At(y, 111-x))
			}
		}
	}
	writePreview(t, os.Getenv("SNOOFER_MEETINGS_PREVIEW"), sheet)
}
