package streamdeck

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func model(t *testing.T, id string) Model {
	t.Helper()
	m, ok := ModelByID(id)
	if !ok {
		t.Fatalf("no model %q", id)
	}
	return m
}

func TestModelsHaveUniqueIDsAndProducts(t *testing.T) {
	ids, products := map[string]bool{}, map[uint16]bool{}
	for _, m := range Models() {
		if ids[m.ID] || m.Keys() == 0 || m.KeySize == 0 {
			t.Fatalf("bad model %+v", m)
		}
		ids[m.ID] = true
		for _, p := range m.Products {
			if products[p] {
				t.Fatalf("product %04x listed twice", p)
			}
			products[p] = true
		}
		if m.Keys() > Keys || m.Dials > Encoders || m.Keys()+m.TouchKeys > Keys {
			t.Fatalf("%s exceeds a Frame", m.ID)
		}
	}
	for path, want := range map[string]string{
		`\\?\HID#VID_0FD9&PID_0084#a&376904d9&0&0000#{4d1e55b2}`: "plus",
		`\\?\hid#vid_0fd9&pid_00c6&mi_00#x`:                      "plus-xl",
		`\\?\HID#VID_0FD9&PID_006D#x`:                            "mk2",
		`\\?\HID#VID_0FD9&PID_008F#x`:                            "xl",
		`\\?\HID#VID_0FD9&PID_009A#x`:                            "neo",
	} {
		if m, ok := modelForPath(path); !ok || m.ID != want {
			t.Fatalf("%s: got %q %v, want %q", path, m.ID, ok, want)
		}
	}
	for _, path := range []string{`\\?\HID#VID_0FD9&PID_0063#mini`, `\\?\HID#VID_0FD9&PID_0086#pedal`, `\\?\HID#VID_1234&PID_0084#x`} {
		if _, ok := modelForPath(path); ok {
			t.Fatalf("%s accepted", path)
		}
	}
}

func TestDecoderFollowsModelCounts(t *testing.T) {
	plus := Decoder{Model: model(t, "plus")}
	keys := make([]byte, 8)
	plus.Decode(report(0, keys))
	keys[7] = 1
	if e, err := plus.Decode(report(0, append(keys, 0, 0, 0))); err != nil || len(e) != 1 || e[0].Key != 7 {
		t.Fatal("extra payload bytes must be ignored", e, err)
	}
	if _, err := plus.Decode(report(0, make([]byte, 7))); err == nil {
		t.Fatal("short key report accepted")
	}
	if e, err := plus.Decode(report(3, []byte{1, 0, 0, 0, 255})); err != nil || len(e) != 1 || e[0].Encoder != 3 || e[0].Delta != -1 {
		t.Fatal("fourth dial turn", e, err)
	}
	if _, err := plus.Decode(report(3, []byte{1, 0, 0, 0})); err == nil {
		t.Fatal("short dial report accepted")
	}

	neo := Decoder{Model: model(t, "neo")}
	states := make([]byte, 10)
	neo.Decode(report(0, states))
	states[9] = 1
	if e, err := neo.Decode(report(0, states)); err != nil || len(e) != 1 || e[0].Key != 9 || !e[0].Press {
		t.Fatal("touch button", e, err)
	}
	if _, err := neo.Decode(report(3, []byte{1, 1})); err == nil {
		t.Fatal("dial report accepted by a deck without dials")
	}

	mk2 := Decoder{Model: model(t, "mk2")}
	if _, err := mk2.Decode(report(0, make([]byte, 15))); err != nil {
		t.Fatal(err)
	}
}

func TestDisplayReportHeaders(t *testing.T) {
	image := make([]byte, 2000)
	plus, err := displayReports(model(t, "plus"), image, 1024)
	if err != nil || len(plus) != 2 {
		t.Fatal(len(plus), err)
	}
	first, last := plus[0], plus[1]
	if first[0] != 2 || first[1] != 0x0c || binary.LittleEndian.Uint16(first[6:8]) != 800 || binary.LittleEndian.Uint16(first[8:10]) != 100 || first[10] != 0 {
		t.Fatalf("first + strip header % x", first[:16])
	}
	if binary.LittleEndian.Uint16(first[13:15]) != 1024-lcdHeader || last[10] != 1 || binary.LittleEndian.Uint16(last[11:13]) != 1 ||
		binary.LittleEndian.Uint16(last[13:15]) != uint16(2000-(1024-lcdHeader)) {
		t.Fatalf("+ strip chunking % x / % x", first[:16], last[:16])
	}
	for _, id := range []string{"plus-xl", "neo"} {
		list, err := displayReports(model(t, id), image, 1024)
		if err != nil || list[0][1] != 0x0b || list[0][2] != 0 {
			t.Fatal(id, err)
		}
	}
}

func TestFramesRenderAtEachModelsSize(t *testing.T) {
	frame := Frame{}
	for n := range frame.Keys {
		frame.Keys[n] = Tile{Label: "Mic", Value: "On", Icon: "mic-mute"}
	}
	frame.Screen = Tile{Label: "Home", Value: "Lights", Icon: "Media"}
	want := map[string]image.Point{"plus-xl": {100, 1200}, "plus": {800, 100}, "neo": {screenWidth, screenHeight}}
	for _, m := range Models() {
		tiles, display := renderFrame(m, frame)
		if len(tiles) != m.Keys() {
			t.Fatalf("%s: %d tiles", m.ID, len(tiles))
		}
		key, err := jpeg.Decode(bytes.NewReader(tiles[0]))
		if err != nil || key.Bounds().Dx() != m.KeySize || key.Bounds().Dy() != m.KeySize {
			t.Fatalf("%s key %v %v", m.ID, key.Bounds(), err)
		}
		size, ok := want[m.ID]
		if !ok {
			if display != nil {
				t.Fatalf("%s has no strip or screen", m.ID)
			}
			continue
		}
		im, err := jpeg.Decode(bytes.NewReader(display))
		if err != nil || im.Bounds().Size() != size {
			t.Fatalf("%s display %v %v", m.ID, im.Bounds(), err)
		}
	}
}

func TestTurns(t *testing.T) {
	// Two pixels: a marker at the left of a 2×1 image.
	im := image.NewRGBA(image.Rect(0, 0, 2, 1))
	marker := color.RGBA{255, 0, 0, 255}
	im.SetRGBA(0, 0, marker)
	if turnImage(im, upright).RGBAAt(0, 0) != marker {
		t.Fatal("upright moved")
	}
	if half := turnImage(im, turnHalf); half.RGBAAt(1, 0) != marker {
		t.Fatal("half turn")
	}
	// Counterclockwise: the left edge goes to the bottom.
	if ccw := turnImage(im, turnCounterclockwise); ccw.Bounds().Dx() != 1 || ccw.RGBAAt(0, 1) != marker {
		t.Fatal("counterclockwise turn")
	}
}

// TestPlusXLKeysUnchanged compares against the encoder as it was before
// models: average and turn counterclockwise in one pass.
func TestPlusXLKeysUnchanged(t *testing.T) {
	const scale, w, h = 4, 112, 112
	im := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	for y := 0; y < h*scale; y++ {
		for x := 0; x < w*scale; x++ {
			im.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), uint8(x ^ y), 255})
		}
	}
	rotated := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b uint32
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					p := im.RGBAAt(x*scale+dx, y*scale+dy)
					r += uint32(p.R)
					g += uint32(p.G)
					b += uint32(p.B)
				}
			}
			rotated.SetRGBA(y, w-1-x, color.RGBA{uint8(r / 16), uint8(g / 16), uint8(b / 16), 255})
		}
	}
	var before bytes.Buffer
	jpeg.Encode(&before, rotated, &jpeg.Options{Quality: 92})
	if !bytes.Equal(encodeKey(im, scale, w, h, keyOutput{size: plusXL.KeySize, turn: plusXL.keyTurn}), before.Bytes()) {
		t.Fatal("+ XL key bytes changed")
	}
}
