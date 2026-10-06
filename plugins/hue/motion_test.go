package hue

import (
	"slices"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

// sensorStudio adds two motion sensors to the studio room.
func sensorStudio() []resource {
	on := true
	items := studio()
	items[0].Children = append(items[0].Children, reference{RID: "dev-4", RType: "device"}, reference{RID: "dev-5", RType: "device"})
	return append(items,
		resource{ID: "dev-4", Type: "device", Services: []reference{{RID: "motion-1", RType: "motion"}}},
		resource{ID: "dev-5", Type: "device", Services: []reference{{RID: "motion-2", RType: "motion"}}},
		resource{ID: "motion-1", Type: "motion", Owner: &reference{RID: "dev-4", RType: "device"}, Enabled: &on},
		resource{ID: "motion-2", Type: "motion", Owner: &reference{RID: "dev-5", RType: "device"}, Enabled: &on},
	)
}

func motionPuts(b *fakeBridge) []string {
	var out []string
	for _, put := range b.putLog() {
		if strings.HasPrefix(put, "motion/") {
			out = append(out, put)
		}
	}
	return out
}

func TestMotionToggle(t *testing.T) {
	bridge := newFakeBridge(t, "b1", sensorStudio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	if c := h.value("hue.motion", "On"); c.Hidden || !c.Available || c.Icon != "hue-motion" {
		t.Fatalf("room with sensors: %+v", c)
	}

	// Accepted writes stay Pending until the bridge reports every sensor.
	bridge.set(func(b *fakeBridge) { b.ignorePuts = true })
	h.mustDispatch("hue.motion", "press", 0, "")
	h.waitControl("hue.motion", func(c snoofer.Control) bool { return c.Status == "Pending" && c.Value == "On" })
	off := false
	bridge.update(resource{ID: "motion-1", Type: "motion", Enabled: &off})
	h.waitControl("hue.motion", func(c snoofer.Control) bool { return c.Status == "Pending" && c.Value == "Mixed" })
	bridge.update(resource{ID: "motion-2", Type: "motion", Enabled: &off})
	if c := h.waitControl("hue.motion", func(c snoofer.Control) bool { return c.Status == "" && c.Value == "Off" }); c.Icon != "hue-motion-off" {
		t.Fatalf("off icon: %+v", c)
	}
	// Sensors are written concurrently, so the order is not fixed.
	if puts := motionPuts(bridge); len(puts) != 2 || !slices.Contains(puts, `motion/motion-1:{"enabled":false}`) || !slices.Contains(puts, `motion/motion-2:{"enabled":false}`) {
		t.Fatalf("disable writes: %v", puts)
	}

	// Mixed enables every sensor, writing only those that differ.
	bridge.set(func(b *fakeBridge) { b.ignorePuts = false })
	on := true
	bridge.update(resource{ID: "motion-2", Type: "motion", Enabled: &on})
	h.value("hue.motion", "Mixed")
	h.mustDispatch("hue.motion", "press", 0, "")
	h.waitControl("hue.motion", func(c snoofer.Control) bool { return c.Status == "" && c.Value == "On" })
	if puts := motionPuts(bridge); len(puts) != 3 || puts[2] != `motion/motion-1:{"enabled":true}` {
		t.Fatalf("enable writes: %v", puts)
	}

	// A failed write reports its reason and leaves the observed value.
	bridge.set(func(b *fakeBridge) { b.throttle = 2 })
	h.mustDispatch("hue.motion", "press", 0, "")
	h.waitControl("hue.motion", func(c snoofer.Control) bool {
		return strings.Contains(c.Status, "bridge busy") && c.Value == "On"
	})

	// A zone has no sensors: blank on the deck, input ignored.
	h.mustDispatch("hue.group", "set", 0, "zone-1")
	if c := h.waitControl("hue.motion", func(c snoofer.Control) bool { return c.Hidden }); c.Available {
		t.Fatalf("zone motion: %+v", c)
	}
}

func TestMotionSensorsByRoom(t *testing.T) {
	m := newModel(sensorStudio())
	if got := m.motionSensors("room-1"); len(got) != 2 || !got[0].Known || !got[0].Enabled {
		t.Fatalf("room sensors: %+v", got)
	}
	if got := m.motionSensors("zone-1"); len(got) != 0 {
		t.Fatalf("zone sensors: %+v", got)
	}
	if got := motionValue([]motionSensor{{ID: "a", Known: true, Enabled: true}, {ID: "b"}}); got != "N/A" {
		t.Fatalf("unknown flag: %s", got)
	}
}
