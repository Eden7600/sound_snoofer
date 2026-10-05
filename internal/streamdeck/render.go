package streamdeck

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/controller"
	"strings"
)

// Five-column glyphs keep the device display independent of installed fonts.
var glyphs = map[rune][5]byte{
	'A': {126, 9, 9, 9, 126}, 'B': {127, 73, 73, 73, 54}, 'C': {62, 65, 65, 65, 34}, 'D': {127, 65, 65, 34, 28}, 'E': {127, 73, 73, 73, 65}, 'F': {127, 9, 9, 9, 1}, 'G': {62, 65, 73, 73, 122}, 'H': {127, 8, 8, 8, 127}, 'I': {65, 65, 127, 65, 65}, 'J': {32, 64, 65, 63, 1}, 'K': {127, 8, 20, 34, 65}, 'L': {127, 64, 64, 64, 64}, 'M': {127, 2, 12, 2, 127}, 'N': {127, 4, 8, 16, 127}, 'O': {62, 65, 65, 65, 62}, 'P': {127, 9, 9, 9, 6}, 'Q': {62, 65, 81, 33, 94}, 'R': {127, 9, 25, 41, 70}, 'S': {38, 73, 73, 73, 50}, 'T': {1, 1, 127, 1, 1}, 'U': {63, 64, 64, 64, 63}, 'V': {31, 32, 64, 32, 31}, 'W': {63, 64, 56, 64, 63}, 'X': {99, 20, 8, 20, 99}, 'Y': {7, 8, 112, 8, 7}, 'Z': {97, 81, 73, 69, 67},
	'0': {62, 81, 73, 69, 62}, '1': {0, 66, 127, 64, 0}, '2': {66, 97, 81, 73, 70}, '3': {33, 65, 69, 75, 49}, '4': {24, 20, 18, 127, 16}, '5': {39, 69, 69, 69, 57}, '6': {60, 74, 73, 73, 48}, '7': {1, 113, 9, 5, 3}, '8': {54, 73, 73, 73, 54}, '9': {6, 73, 73, 41, 30}, '-': {8, 8, 8, 8, 8}, '.': {0, 96, 96, 0, 0}, '/': {32, 16, 8, 4, 2}, '?': {2, 1, 81, 9, 6}, '+': {8, 8, 62, 8, 8}}

func text(im *image.RGBA, x, y, scale int, s string, c color.RGBA) {
	for _, r := range strings.ToUpper(s) {
		g := glyphs[r]
		for col, v := range g {
			for row := 0; row < 7; row++ {
				if v&(1<<row) != 0 {
					for dx := 0; dx < scale; dx++ {
						for dy := 0; dy < scale; dy++ {
							im.SetRGBA(x+col*scale+dx, y+row*scale+dy, c)
						}
					}
				}
			}
		}
		x += 6 * scale
	}
}
func render(lines []string, w, h int, on bool) []byte {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	background := color.RGBA{14, 22, 30, 255}
	if on {
		background = color.RGBA{0, 90, 110, 255}
	}
	for _, line := range lines {
		if line == "ERROR" || line == "UNAVAIL" {
			background = color.RGBA{150, 30, 35, 255}
		}
		if line == "PENDING" {
			background = color.RGBA{120, 95, 0, 255}
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.SetRGBA(x, y, background)
		}
	}
	for n, line := range lines {
		if len(line) > 9 && w == 112 {
			line = line[:9]
		}
		text(im, 7, 12+n*22, 2, line, color.RGBA{230, 242, 245, 255})
	}
	// + XL requires 90-degree counterclockwise JPEG images.
	rotated := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rotated.SetRGBA(y, w-1-x, im.RGBAAt(x, y))
		}
	}
	var b bytes.Buffer
	jpeg.Encode(&b, rotated, &jpeg.Options{Quality: 80})
	return b.Bytes()
}

var bindings = []string{"record-start", "record-stop", "record-computer", "record-mic", "record-tap", "snippet-play", "snippet-stop", "record-loop", "record-vst", "mic-mute", "speaker-mute", "monitor", "mode", "media-prev", "media-play", "media-next", "media-stop", "open-controls"}
var labels = []string{"RECORD", "STOP REC", "PC RECORD", "MIC REC", "MIC STAGE", "PLAY TAPE", "STOP TAPE", "LOOP", "TO VST", "MIC MUTE", "SPKR MUTE", "MONITOR", "VST MODE", "PREVIOUS", "PLAY/PAUSE", "NEXT", "STOP MEDIA", "CONTROLS"}

func display(s control.State, layout []string) ([][]byte, []byte) {
	tiles := make([][]byte, Keys)
	for n := 0; n < Keys; n++ {
		if n >= len(layout) || layout[n] == "" {
			tiles[n] = render(nil, 112, 112, false)
			continue
		}
		value := Value(s, layout[n])
		if !s.Connected {
			value = "UNAVAIL"
		}
		if layout[n] == "record-start" {
			value = s.Recorder.State()
		}
		if !s.Live {
			value = "PREVIEW"
		}
		label := strings.ToUpper(strings.ReplaceAll(layout[n], "-", " "))
		for j, key := range bindings {
			if layout[n] == key {
				label = labels[j]
				break
			}
		}
		if s.NoticeKind == control.NoticeError && s.Notice != "" {
			value = "ERROR"
		}
		if s.Plan != nil && s.Plan.HasChanges() {
			value = "PENDING"
		}
		tiles[n] = render([]string{label, value}, 112, 112, value == "On" || value == "Recording")
	}
	touchImage := image.NewRGBA(image.Rect(0, 0, 1200, 100))
	for n, target := range []string{"A1", "A2", "mic"} {
		p := controller.GainTarget(s.Plan, target)
		value := "?"
		if gain, ok := s.Snapshot.Numbers[p]; ok && s.Connected {
			value = fmt.Sprintf("%.1f DB", gain)
		}
		if s.Snapshot.Numbers[strings.TrimSuffix(p, "Gain")+"Mute"] == 1 {
			value += " MUTE"
		}
		text(touchImage, n*200+8, 12, 2, target, color.RGBA{30, 200, 220, 255})
		text(touchImage, n*200+8, 40, 2, value, color.RGBA{240, 240, 240, 255})
		name := s.Snapshot.Assignments[target]
		if target == "mic" && s.Plan != nil && s.Plan.Topology != nil && s.Plan.Topology.Voice != nil {
			name = s.Plan.Topology.Voice.Effective
		}
		if len(name) > 15 {
			name = name[:15]
		}
		text(touchImage, n*200+8, 72, 2, name, color.RGBA{150, 160, 170, 255})
	}
	rotated := image.NewRGBA(image.Rect(0, 0, 100, 1200))
	for y := 0; y < 100; y++ {
		for x := 0; x < 1200; x++ {
			rotated.SetRGBA(y, 1199-x, touchImage.RGBAAt(x, y))
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, rotated, &jpeg.Options{Quality: 80})
	return tiles, buf.Bytes()
}
