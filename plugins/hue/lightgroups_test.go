package hue

import (
	"encoding/json"
	"math"
	"testing"
)

func TestSaturationAlongItsOwnHue(t *testing.T) {
	// The gamut's red corner is fully saturated; white has no hue.
	if s, ok := saturation(gamutC.Red, gamutC); !ok || math.Abs(s-1) > 0.01 {
		t.Fatal("red corner", s, ok)
	}
	if _, ok := saturation(whitePoint, gamutC); ok {
		t.Fatal("white has a saturation")
	}
	// Desaturating keeps the hue: the new point lies on the same ray.
	blue := xyPoint{0.2, 0.15}
	before, _ := saturation(blue, gamutC)
	moved := withSaturation(blue, gamutC, 0.3)
	after, ok := saturation(moved, gamutC)
	if !ok || math.Abs(after-0.3) > 0.01 {
		t.Fatal("target not reached", before, after)
	}
	cross := (blue.X-whitePoint.X)*(moved.Y-whitePoint.Y) - (blue.Y-whitePoint.Y)*(moved.X-whitePoint.X)
	if math.Abs(cross) > 1e-6 {
		t.Fatal("hue shifted", blue, moved)
	}
	// Full saturation stays inside the gamut.
	if full := withSaturation(blue, gamutC, 1); full == blue {
		t.Fatal("not moved")
	} else if s, _ := saturation(full, gamutC); s > 1 {
		t.Fatal("left the gamut", s)
	}
	// Near-white colours are left alone.
	near := xyPoint{whitePoint.X + 0.001, whitePoint.Y}
	if withSaturation(near, gamutC, 1) != near {
		t.Fatal("near-white colour moved")
	}
}

func TestRoomLightsByFunctionAndColourMode(t *testing.T) {
	var items []resource
	err := json.Unmarshal([]byte(`[
		{"id":"room","type":"room","metadata":{"name":"Office"},"children":[{"rid":"d1","rtype":"device"},{"rid":"d2","rtype":"device"},{"rid":"d3","rtype":"device"}]},
		{"id":"d1","type":"device","services":[{"rid":"strip","rtype":"light"}]},
		{"id":"d2","type":"device","services":[{"rid":"lamp","rtype":"light"}]},
		{"id":"d3","type":"device","services":[{"rid":"white","rtype":"light"}]},
		{"id":"strip","type":"light","metadata":{"name":"Strip","function":"decorative"},"on":{"on":true},"dimming":{"brightness":40},
			"color":{"xy":{"x":0.2,"y":0.15},"gamut":{"red":{"x":0.6915,"y":0.3083},"green":{"x":0.17,"y":0.7},"blue":{"x":0.1532,"y":0.0475}}},
			"color_temperature":{"mirek_valid":false},"gradient":{"points":[{"color":{"xy":{"x":0.6,"y":0.32}}},{"color":{"xy":{"x":0.2,"y":0.15}}}]}},
		{"id":"lamp","type":"light","metadata":{"name":"Lamp","function":"mixed"},"on":{"on":true},"dimming":{"brightness":80},
			"color":{"xy":{"x":0.45,"y":0.41}},"color_temperature":{"mirek_valid":true}},
		{"id":"white","type":"light","metadata":{"name":"White","function":"functional"},"on":{"on":false},"dimming":{"brightness":100}}
	]`), &items)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(items)
	lights := m.roomLights("room")
	if len(lights) != 3 {
		t.Fatal("lights", lights)
	}
	decor := withFunction(lights, decorative)
	if len(decor) != 1 || decor[0].ID != "strip" || !decor[0].ColourMode || len(decor[0].Points) != 2 {
		t.Fatal("decorative", decor)
	}
	lamp := withFunction(lights, mixed)[0]
	if !lamp.Colour || lamp.ColourMode {
		t.Fatal("a light showing a white temperature is in colour mode", lamp)
	}
	if colours := colourLights(lights); len(colours) != 2 {
		t.Fatal("white-only lamp counted as colour", colours)
	}
	if s := summarize(withFunction(lights, mixed)); !s.Known || !s.AnyOn || s.Brightness != 80 {
		t.Fatal("mixed summary", s)
	}
	// Only the strip shows colour; the lamp's white temperature has no saturation.
	if avg, ok := averageSaturation(lights); !ok || avg <= 0 || avg > 1 {
		t.Fatal("average saturation", avg, ok)
	}
	if _, ok := averageSaturation(withFunction(lights, mixed)); ok {
		t.Fatal("white-temperature light gave a saturation")
	}
	// An event update with only xy keeps the gamut.
	m.apply([]event{{Type: "update", Data: []resource{{ID: "strip", Color: &lightColor{XY: &xyPoint{0.3, 0.3}}}}}})
	if got := m.resources["strip"].Color; got.Gamut == nil || got.XY.X != 0.3 {
		t.Fatal("colour update lost the gamut", got)
	}
}
