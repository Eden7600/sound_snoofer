package hue

import (
	"encoding/json"
	"slices"
	"testing"

	"sound-snoofer/snoofer"
)

// sortScene is a room-1 scene setting one light to the given action.
func sortScene(id, name, action string) resource {
	r := resource{ID: id, Type: "scene", Metadata: &metadata{Name: name}, Group: &reference{RID: "room-1", RType: "room"}}
	if action == "" {
		return r
	}
	if err := json.Unmarshal([]byte(`{"actions":[{"target":{"rid":"light-1"},"action":`+action+`}]}`), &r); err != nil {
		panic(err)
	}
	return r
}

func sortScenes() []resource {
	return []resource{
		sortScene("s-blue", "Blue", `{"on":{"on":true},"dimming":{"brightness":40},"color":{"xy":{"x":0.15,"y":0.06}}}`),
		sortScene("s-red", "red", `{"on":{"on":true},"dimming":{"brightness":100},"color":{"xy":{"x":0.6915,"y":0.3083}}}`),
		sortScene("s-green", "Green", `{"on":{"on":true},"dimming":{"brightness":40},"color":{"xy":{"x":0.17,"y":0.7}}}`),
		sortScene("s-white", "Daylight", `{"on":{"on":true},"dimming":{"brightness":70},"color_temperature":{"mirek":200}}`),
		sortScene("s-empty", "Empty", ""),
	}
}

func names(scenes []sceneInfo) []string {
	var out []string
	for _, s := range scenes {
		out = append(out, s.ShortLabel)
	}
	return out
}

func TestSceneSortOrders(t *testing.T) {
	m := newModel(append(studio(), sortScenes()...))
	for _, tc := range []struct {
		mode string
		want []string
	}{
		{"", []string{"Blue", "Bright", "Daylight", "Empty", "Green", "red"}},
		{sortName, []string{"Blue", "Bright", "Daylight", "Empty", "Green", "red"}},
		// Colours by hue angle, then white, then scenes without colour data by name.
		{sortHue, []string{"red", "Green", "Blue", "Daylight", "Bright", "Empty"}},
		// Brightest first; equal brightness by name; unknown last.
		{sortBrightness, []string{"red", "Daylight", "Blue", "Green", "Bright", "Empty"}},
	} {
		if got := names(m.roomScenes("room-1", tc.mode)); !slices.Equal(got, tc.want) {
			t.Errorf("%q order %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestHueSaturation(t *testing.T) {
	for _, tc := range []struct {
		s          swatch
		angle, sat float64
	}{
		{swatch{R: 255}, 0, 1},
		{swatch{G: 255}, 120, 1},
		{swatch{B: 255}, 240, 1},
		{swatch{R: 255, B: 128}, 330, 1},
		{swatch{R: 200, G: 200, B: 200}, 0, 0},
		{swatch{}, 0, 0},
	} {
		angle, sat := hueSaturation(tc.s)
		if angle < tc.angle-0.5 || angle > tc.angle+0.5 || sat < tc.sat-0.01 || sat > tc.sat+0.01 {
			t.Errorf("%+v: %.1f° %.2f, want %.0f° %.2f", tc.s, angle, sat, tc.angle, tc.sat)
		}
	}
}

func TestSceneSortSetting(t *testing.T) {
	if validate(json.RawMessage(`{"scene_sort":"colour"}`)) == nil {
		t.Fatal("unknown sort accepted")
	}
	if err := validate(json.RawMessage(`{"scene_sort":"brightness"}`)); err != nil {
		t.Fatal(err)
	}
	bridge := newFakeBridge(t, "b1", append(studio(), sortScenes()...)...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	sort := h.value("hue.scene-sort", "name")
	if !slices.Equal(sort.Options, []string{"name", "hue", "brightness"}) || !sort.Available || sort.Icon != "hue-sort" {
		t.Fatalf("sort control %+v", sort)
	}
	h.value("hue.room-scene-1", "Ready")
	if c, _ := h.find("hue.room-scene-1"); c.Label != "Blue" || c.Order != 1 {
		t.Fatalf("first slot by name %+v", c)
	}
	h.mustDispatch("hue.scene-sort", "set", 0, "brightness")
	h.waitControl("hue.room-scene-1", func(c snoofer.Control) bool { return c.Label == "red" })
	controls := map[string]snoofer.Control{}
	for _, c := range h.controls.Snapshot() {
		controls[c.Label] = c
	}
	if controls["Studio red"].Order != 1 || controls["Studio Daylight"].Order != 2 || controls["Studio Empty"].Order != 6 {
		t.Fatalf("scene orders red %d daylight %d empty %d", controls["Studio red"].Order, controls["Studio Daylight"].Order, controls["Studio Empty"].Order)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.saved.SceneSort != "brightness" {
		t.Fatalf("sort not saved: %+v", h.saved)
	}
}
