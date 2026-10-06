package hue

import (
	"math"
	"testing"
)

func TestGroupViewRoomAndZone(t *testing.T) {
	m := newModel(studio())
	room := m.group("room-1")
	if !room.Found || room.GroupedLight != "gl-1" || !room.OnKnown || !room.On || room.Brightness != 50 {
		t.Fatalf("room %+v", room)
	}
	if !room.CTCapable || room.MirekMin != 153 || room.MirekMax != 454 {
		t.Fatalf("range %+v", room)
	}
	if room.Temperature != temperatureKnown || math.Round(room.Kelvin) != 4000 {
		t.Fatalf("temperature %+v", room)
	}
	zone := m.group("zone-1")
	if !zone.Found || zone.GroupedLight != "gl-2" || zone.Temperature != temperatureKnown {
		t.Fatalf("zone %+v", zone)
	}
	if missing := m.group("gone"); missing.Found {
		t.Fatal("missing group found")
	}
	if light := m.group("light-1"); light.Found {
		t.Fatal("light treated as group")
	}
}

func TestGroupTemperatureMixedAndUnknown(t *testing.T) {
	m := newModel(studio())
	m.apply([]event{{Type: "update", Data: []resource{{ID: "light-2", ColorTemperature: &colorTemperature{Mirek: intPtr(400)}}}}})
	if view := m.group("room-1"); view.Temperature != temperatureMixed {
		t.Fatalf("expected mixed, got %+v", view)
	}
	m.apply([]event{{Type: "update", Data: []resource{
		{ID: "light-1", ColorTemperature: &colorTemperature{Valid: boolPtr(false)}},
		{ID: "light-2", On: &onState{On: false}},
	}}})
	view := m.group("room-1")
	if view.Temperature != temperatureUnknown || !view.CTCapable {
		t.Fatalf("expected unknown, got %+v", view)
	}
}

func TestGroupRangeFallsBackToUnion(t *testing.T) {
	m := newModel(studio())
	m.apply([]event{{Type: "update", Data: []resource{
		{ID: "light-2", ColorTemperature: &colorTemperature{Schema: &mirekSchema{Minimum: 460, Maximum: 500}}},
	}}})
	view := m.group("room-1")
	if view.MirekMin != 153 || view.MirekMax != 500 {
		t.Fatalf("range %d-%d", view.MirekMin, view.MirekMax)
	}
}

func TestPartialUpdateKeepsMissingFields(t *testing.T) {
	m := newModel(studio())
	m.apply([]event{{Type: "update", Data: []resource{{ID: "light-1", On: &onState{On: false}}}}})
	ct := m.resources["light-1"].ColorTemperature
	if ct == nil || ct.Mirek == nil || *ct.Mirek != 250 || ct.Schema == nil {
		t.Fatalf("color temperature lost: %+v", ct)
	}
	m.apply([]event{{Type: "update", Data: []resource{{ID: "unknown", On: &onState{}}}}})
	if _, ok := m.resources["unknown"]; ok {
		t.Fatal("update created a resource")
	}
	m.apply([]event{{Type: "delete", Data: []resource{{ID: "zone-1"}}}})
	if len(m.groups()) != 1 {
		t.Fatal("deleted zone still listed")
	}
}

func TestScenesHaveStableRoomPrefixedIDs(t *testing.T) {
	scenes := newModel(studio()).scenes()
	if len(scenes) != 2 {
		t.Fatalf("scenes %+v", scenes)
	}
	focus, bright := scenes[0], scenes[1]
	if focus.ControlID != "hue.scene-desk-8b7e0d21" || focus.Label != "Desk Focus" || focus.ShortLabel != "Focus" || !focus.Active {
		t.Fatalf("focus %+v", focus)
	}
	if bright.ControlID != "hue.scene-studio-3f2a9c10" || bright.Active {
		t.Fatalf("bright %+v", bright)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{"Living Room": "living-room", "  Café #2 ": "caf-2", "": "group", "***": "group"}
	for input, want := range cases {
		if got := slug(input); got != want {
			t.Errorf("slug(%q) = %q, want %q", input, got, want)
		}
	}
}
