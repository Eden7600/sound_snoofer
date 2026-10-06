package hue

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/draw"
	"image/png"
	"os"
	"sort"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func decodeArtwork(t *testing.T, art string) image.Image {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(art)
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return im
}

func TestColorConversion(t *testing.T) {
	red := xySwatch(xyPoint{X: 0.6915, Y: 0.3083}, 1, 1)
	if red.R < 240 || red.G > 80 || red.B > 80 {
		t.Fatalf("red %+v", red)
	}
	blue := xySwatch(xyPoint{X: 0.1532, Y: 0.0475}, 1, 1)
	if blue.B < 240 || blue.R > blue.B/2 {
		t.Fatalf("blue %+v", blue)
	}
	warm, cool := mirekSwatch(500, 1, 1), mirekSwatch(153, 1, 1)
	if warm.R != 255 || warm.B > 120 || cool.B < 240 || cool.R < 200 {
		t.Fatalf("warm %+v cool %+v", warm, cool)
	}
	dim := xySwatch(xyPoint{X: 0.6915, Y: 0.3083}, artworkLevel(&dimming{Brightness: 0}), 1)
	if dim.R < 0.5*red.R || dim.R >= red.R {
		t.Fatalf("dim scene not scaled to the visible floor: %+v", dim)
	}
}

func scene(actions string, palette string) resource {
	r := resource{ID: "s", Type: "scene"}
	if err := json.Unmarshal([]byte(`{"actions":`+actions+`,"palette":`+palette+`}`), &r); err != nil {
		panic(err)
	}
	return r
}

func TestSceneSwatches(t *testing.T) {
	s := scene(`[
		{"target":{"rid":"a"},"action":{"on":{"on":true},"color":{"xy":{"x":0.6915,"y":0.3083}}}},
		{"target":{"rid":"b"},"action":{"on":{"on":false},"color":{"xy":{"x":0.17,"y":0.7}}}},
		{"target":{"rid":"c"},"action":{"on":{"on":true},"color_temperature":{"mirek":370}}},
		{"target":{"rid":"d"},"action":{"on":{"on":true},"gradient":{"points":[{"color":{"xy":{"x":0.15,"y":0.06}}},{"color":{"xy":{"x":0.6915,"y":0.3083}}}]}}},
		{"target":{"rid":"e"},"action":{"on":{"on":true},"dimming":{"brightness":40}}}]`, `null`)
	swatches := sceneSwatches(s)
	if len(swatches) != 4 {
		t.Fatalf("%d swatches, want red, warm white and two gradient halves: %+v", len(swatches), swatches)
	}
	wedges := mergeSwatches(swatches)
	if len(wedges) != 3 || wedges[0].Weight != 1.5 || wedges[0].R < 240 {
		t.Fatalf("red light and half gradient should merge first: %+v", wedges)
	}

	fallback := scene(`[{"target":{"rid":"a"},"action":{"on":{"on":true},"dimming":{"brightness":0}}}]`,
		`{"color":[{"color":{"xy":{"x":0.561,"y":0.4042}},"dimming":{"brightness":0}}],"color_temperature":[]}`)
	if got := sceneSwatches(fallback); len(got) != 1 || got[0].R < got[0].B {
		t.Fatalf("palette fallback %+v", got)
	}
	if got := sceneSwatches(scene(`[{"target":{"rid":"a"},"action":{"on":{"on":true}}}]`, `null`)); len(got) != 0 {
		t.Fatalf("colorless scene produced %+v", got)
	}
}

func TestMergeKeepsFiveLargest(t *testing.T) {
	var in []swatch
	for n := range 7 {
		in = append(in, swatch{R: float64(n * 40), G: 0, B: 255 - float64(n*35), Weight: float64(n + 1)})
	}
	wedges := mergeSwatches(in)
	if len(wedges) != maxWedges || wedges[0].Weight != 7 {
		t.Fatalf("wedges %+v", wedges)
	}
	if !sort.SliceIsSorted(wedges, func(i, j int) bool { return wedges[i].Weight > wedges[j].Weight }) {
		t.Fatal("wedges not largest first")
	}
}

func TestRenderArtworkGeometry(t *testing.T) {
	art, err := renderArtwork([]swatch{{R: 255, G: 0, B: 0, Weight: 3}, {R: 0, G: 0, B: 255, Weight: 1}})
	if err != nil {
		t.Fatal(err)
	}
	im := decodeArtwork(t, art)
	if im.Bounds().Dx() != artworkSize || im.Bounds().Dy() != artworkSize {
		t.Fatalf("size %v", im.Bounds())
	}
	if _, _, _, a := im.At(0, 0).RGBA(); a != 0 {
		t.Fatal("corner not transparent")
	}
	// The largest wedge starts at 12 o'clock and runs clockwise for 3/4 of the disc.
	if r, _, b, _ := im.At(50, 32).RGBA(); r>>8 < 200 || b>>8 > 50 {
		t.Fatalf("right side should be red, got r=%d b=%d", r>>8, b>>8)
	}
	if r, _, b, _ := im.At(20, 20).RGBA(); b>>8 < 200 || r>>8 > 50 {
		t.Fatalf("upper left should be blue, got r=%d b=%d", r>>8, b>>8)
	}
	if empty, _ := renderArtwork(nil); empty != "" {
		t.Fatal("artwork without colors")
	}
	// SNOOFER_HUE_ART_SAMPLE writes a Storybook-like sample used by GUI and deck previews.
	if path := os.Getenv("SNOOFER_HUE_ART_SAMPLE"); path != "" {
		sample := mergeSwatches([]swatch{
			xySwatch(xyPoint{X: 0.5108, Y: 0.4191}, 1, 2), xySwatch(xyPoint{X: 0.6915, Y: 0.3083}, 1, 1),
			xySwatch(xyPoint{X: 0.17, Y: 0.7}, 1, 1), mirekSwatch(250, 1, 1),
		})
		art, err := renderArtwork(sample)
		if err != nil {
			t.Fatal(err)
		}
		data, err := base64.StdEncoding.DecodeString(art)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSceneArtworkCachedAndRefreshed(t *testing.T) {
	w := &worker{model: newModel([]resource{scene(`[{"target":{"rid":"a"},"action":{"on":{"on":true},"color":{"xy":{"x":0.6915,"y":0.3083}}}}]`, `null`)}),
		artwork: map[string]cachedArtwork{}}
	first := w.sceneArtwork("s")
	if first == "" || w.sceneArtwork("s") != first || len(w.artwork) != 1 {
		t.Fatal("artwork not cached")
	}
	w.model.apply([]event{{Type: "update", Data: []resource{scene(`[{"target":{"rid":"a"},"action":{"on":{"on":true},"color":{"xy":{"x":0.15,"y":0.06}}}}]`, `null`)}}})
	if second := w.sceneArtwork("s"); second == "" || second == first {
		t.Fatal("artwork not refreshed after a color change")
	}
	if (&worker{model: newModel([]resource{{ID: "x", Type: "scene"}}), artwork: map[string]cachedArtwork{}}).sceneArtwork("x") != "" {
		t.Fatal("colorless scene got artwork")
	}
}

func TestSceneControlsCarryArtwork(t *testing.T) {
	items := studio()
	for n := range items {
		if items[n].Type == "scene" {
			items[n].Actions = scene(`[{"target":{"rid":"light-1"},"action":{"on":{"on":true},"color":{"xy":{"x":0.6915,"y":0.3083}}}}]`, `null`).Actions
		}
	}
	bridge := newFakeBridge(t, "b1", items...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	scene := h.waitControl("hue.scene-studio-3f2a9c10", func(c snoofer.Control) bool { return c.Artwork != "" })
	slot := h.waitControl("hue.room-scene-1", func(c snoofer.Control) bool { return c.Artwork != "" })
	if slot.Artwork != scene.Artwork {
		t.Fatal("slot artwork differs from its scene")
	}
}

// TestLiveArtworkPreview renders every scene of a paired bridge into a sheet
// (SNOOFER_HUE_CONFIG and SNOOFER_HUE_ART_PREVIEW) for visual inspection.
func TestLiveArtworkPreview(t *testing.T) {
	config, output := os.Getenv("SNOOFER_HUE_CONFIG"), os.Getenv("SNOOFER_HUE_ART_PREVIEW")
	if config == "" || output == "" {
		t.Skip("set SNOOFER_HUE_CONFIG and SNOOFER_HUE_ART_PREVIEW")
	}
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Plugins map[string]struct {
			Settings Settings `json:"settings"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	settings := envelope.Plugins["hue"].Settings
	target, err := resolve(context.Background(), settings.Address, settings.BridgeID, func(ctx context.Context) ([]string, error) { return discoverMDNS(ctx, 3*time.Second) })
	if err != nil {
		t.Fatal(err)
	}
	client := newClient(target.Address, settings.CertificateSHA256, settings.AppKey)
	defer client.close()
	items, err := client.Resources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	w := &worker{model: newModel(items), artwork: map[string]cachedArtwork{}}
	scenes := w.model.scenes()
	sheet := image.NewNRGBA(image.Rect(0, 0, 10*72, ((len(scenes)+9)/10)*72))
	for n, s := range scenes {
		art := w.sceneArtwork(s.SceneID)
		if art == "" {
			t.Logf("no colors: %s", s.Label)
			continue
		}
		im := decodeArtwork(t, art)
		at := image.Pt((n%10)*72+4, (n/10)*72+4)
		draw.Draw(sheet, im.Bounds().Add(at), im, image.Point{}, draw.Over)
		t.Logf("%2d %s", n, s.Label)
	}
	file, err := os.Create(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, sheet); err != nil {
		t.Fatal(err)
	}
}
