package hue

import (
	"strings"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

// lounge is a room with a decorative gradient strip, a mixed colour lamp
// showing red, and a functional white lamp.
func lounge() []resource {
	falseValue := false
	xy := func(x, y float64) *xyPoint { return &xyPoint{x, y} }
	strip := resource{ID: "strip", Type: "light", Metadata: &metadata{Name: "Strip", Function: decorative},
		On: &onState{On: true}, Dimming: &dimming{Brightness: 60}, Color: &lightColor{XY: xy(0.2, 0.15)},
		ColorTemperature: &lightTemperature{MirekValid: &falseValue}, Gradient: &lightGradient{}}
	strip.Gradient.Points = make([]struct {
		Color xyColor `json:"color"`
	}, 2)
	strip.Gradient.Points[0].Color.XY = xyPoint{0.6, 0.32}
	strip.Gradient.Points[1].Color.XY = xyPoint{0.2, 0.15}
	return []resource{
		{ID: "room-1", Type: "room", Metadata: &metadata{Name: "Lounge"},
			Children: []reference{{RID: "dev-1", RType: "device"}, {RID: "dev-2", RType: "device"}, {RID: "dev-3", RType: "device"}},
			Services: []reference{{RID: "gl-1", RType: "grouped_light"}}},
		{ID: "dev-1", Type: "device", Services: []reference{{RID: "strip", RType: "light"}}},
		{ID: "dev-2", Type: "device", Services: []reference{{RID: "lamp", RType: "light"}}},
		{ID: "dev-3", Type: "device", Services: []reference{{RID: "white", RType: "light"}}},
		{ID: "gl-1", Type: "grouped_light", On: &onState{On: true}, Dimming: &dimming{Brightness: 50}},
		strip,
		{ID: "lamp", Type: "light", Metadata: &metadata{Name: "Lamp", Function: mixed}, On: &onState{On: true}, Dimming: &dimming{Brightness: 80},
			Color: &lightColor{XY: xy(0.6, 0.32)}, ColorTemperature: &lightTemperature{MirekValid: &falseValue}},
		{ID: "white", Type: "light", Metadata: &metadata{Name: "White", Function: "functional"}, On: &onState{On: false}, Dimming: &dimming{Brightness: 100}},
	}
}

func TestDecorStagesThenAppliesOnPress(t *testing.T) {
	bridge := newFakeBridge(t, "b1", lounge()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.decor", "60%")
	for range 4 {
		h.mustDispatch("hue.decor", "adjust", -1, "")
	}
	staged := h.value("hue.decor", "→ 40%")
	if !staged.Subdued {
		t.Fatal("staged target not subdued")
	}
	if puts := bridge.putLog(); len(puts) != 0 {
		t.Fatalf("turning wrote %v", puts)
	}
	h.mustDispatch("hue.decor", "press", 0, "")
	h.value("hue.decor", "40%")
	h.settled("hue.decor")
	if puts := bridge.putLog(); len(puts) != 1 || puts[0] != `light/strip:{"dimming":{"brightness":40},"on":{"on":true}}` {
		t.Fatalf("press wrote %v", puts)
	}
}

func TestMixedPressTogglesWithoutStage(t *testing.T) {
	bridge := newFakeBridge(t, "b1", lounge()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.mixed", "80%")
	h.mustDispatch("hue.mixed", "press", 0, "")
	h.value("hue.mixed", "Off")
	if puts := bridge.putLog(); len(puts) != 1 || puts[0] != `light/lamp:{"on":{"on":false}}` {
		t.Fatalf("toggle wrote %v", puts)
	}
	// Saturation has nothing to toggle.
	h.mustDispatch("hue.saturation", "press", 0, "")
	time.Sleep(5 * testTiming.lightGap)
	if puts := bridge.putLog(); len(puts) != 1 {
		t.Fatalf("saturation press without a stage wrote %v", puts)
	}
}

func TestStageExpires(t *testing.T) {
	bridge := newFakeBridge(t, "b1", lounge()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.mixed", "80%")
	h.mustDispatch("hue.mixed", "adjust", 2, "")
	h.value("hue.mixed", "→ 90%")
	h.value("hue.mixed", "80%") // Dropped after stageHold without a press.
	if puts := bridge.putLog(); len(puts) != 0 {
		t.Fatalf("expired stage wrote %v", puts)
	}
}

func TestSaturationKeepsHueAcrossLights(t *testing.T) {
	bridge := newFakeBridge(t, "b1", lounge()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	start := h.waitControl("hue.saturation", func(c snoofer.Control) bool { return strings.HasSuffix(c.Value, "%") })
	h.mustDispatch("hue.saturation", "adjust", -40, "") // Down to 0%, then clamped.
	h.value("hue.saturation", "→ 0%")
	h.mustDispatch("hue.saturation", "adjust", 6, "")
	h.value("hue.saturation", "→ 30%")
	h.mustDispatch("hue.saturation", "press", 0, "")
	h.value("hue.saturation", "30%")
	puts := bridge.putLog()
	if len(puts) != 2 {
		t.Fatalf("saturation wrote %v (started at %s)", puts, start.Value)
	}
	// The lamp gets a colour, the gradient strip every point; the white lamp nothing.
	if !strings.HasPrefix(puts[0], `light/lamp:{"color":`) && !strings.HasPrefix(puts[1], `light/lamp:{"color":`) {
		t.Fatalf("lamp colour not written: %v", puts)
	}
	for _, put := range puts {
		if strings.HasPrefix(put, "light/white") {
			t.Fatal("white-only lamp written")
		}
	}
	lights := h.lightsAfter()
	for _, l := range lights {
		for _, xy := range lightColours(l) {
			if s, ok := saturation(xy, l.Gamut); ok && (s < 0.28 || s > 0.32) {
				t.Fatalf("%s at saturation %.2f", l.ID, s)
			}
		}
	}
}

func TestGroupDialsHiddenWithoutMatchingLights(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.brightness", "50%")
	for _, id := range []string{"hue.decor", "hue.mixed", "hue.saturation"} {
		if c, ok := h.find(id); !ok || !c.Hidden || c.Available {
			t.Fatalf("%s in a room without such lights: %+v", id, c)
		}
	}
}

func TestGroupWritesPacedAndConfirmed(t *testing.T) {
	bridge := newFakeBridge(t, "b1", lounge()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.decor", "60%")
	// A throttled write is retried after a longer gap.
	bridge.set(func(b *fakeBridge) { b.throttle = 1 })
	h.mustDispatch("hue.decor", "adjust", 1, "")
	h.mustDispatch("hue.decor", "press", 0, "")
	h.value("hue.decor", "65%")
	if puts := bridge.putLog(); len(puts) != 2 {
		t.Fatalf("throttled write not retried once: %v", puts)
	}
	// An unconfirmed light is reported, not retried.
	bridge.set(func(b *fakeBridge) { b.ignorePuts = true })
	h.mustDispatch("hue.decor", "adjust", 1, "")
	h.mustDispatch("hue.decor", "press", 0, "")
	c := h.waitControl("hue.decor", func(c snoofer.Control) bool { return c.Status != "" })
	if c.Status != "Not applied: 1 light" {
		t.Fatalf("unconfirmed status %q", c.Status)
	}
	if puts := bridge.putLog(); len(puts) != 3 {
		t.Fatalf("unconfirmed write retried: %v", puts)
	}
}

// lightsAfter reads the fake bridge's lights in room-1 once writes settle.
func (h *harness) lightsAfter() []roomLight {
	h.t.Helper()
	time.Sleep(5 * testTiming.lightGap)
	h.bridge.mu.Lock()
	defer h.bridge.mu.Unlock()
	return model{resources: h.bridge.resources}.roomLights("room-1")
}
