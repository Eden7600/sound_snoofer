package hue

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

var testTiming = timing{
	writeGap:    40 * time.Millisecond,
	maxWriteGap: 200 * time.Millisecond,
	confirm:     300 * time.Millisecond,
	pairWindow:  3 * time.Second,
	retryBase:   50 * time.Millisecond,
	retryMax:    200 * time.Millisecond,
	rediscover:  time.Hour,
}

type harness struct {
	t        *testing.T
	controls *snoofer.Controls
	bridge   *fakeBridge
	instance snoofer.Instance

	mu    sync.Mutex
	saved Settings
}

func paired(b *fakeBridge, group string) Settings {
	return Settings{Address: b.address(), BridgeID: b.id, AppKey: b.key, CertificateSHA256: b.fingerprint(), Group: group, NeutralKelvin: 4000}
}

func startHarness(t *testing.T, bridge *fakeBridge, settings Settings, live bool) *harness {
	t.Helper()
	h := &harness{t: t, controls: snoofer.NewControls(), bridge: bridge, saved: settings}
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	services := snoofer.Services{Controls: h.controls, Live: live, SaveSettings: func(id string, expected, next json.RawMessage) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		current, _ := json.Marshal(h.saved)
		if id != "hue" || string(current) != string(expected) {
			t.Errorf("save for %s expected %s, have %s", id, expected, current)
		}
		return json.Unmarshal(next, &h.saved)
	}}
	discover := func(context.Context) ([]string, error) { return []string{bridge.address()}, nil }
	h.instance, err = start(context.Background(), services, raw, discover, testTiming)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.instance.Stop(ctx); err != nil {
			t.Errorf("stop: %v", err)
		}
		if _, ok := h.find("hue.status"); ok {
			t.Error("controls remain after stop")
		}
	})
	return h
}

func (h *harness) find(id string) (snoofer.Control, bool) {
	for _, c := range h.controls.Snapshot() {
		if c.ID == id {
			return c, true
		}
	}
	return snoofer.Control{}, false
}

func (h *harness) waitControl(id string, ok func(snoofer.Control) bool) snoofer.Control {
	h.t.Helper()
	var last snoofer.Control
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c, found := h.find(id); found {
			last = c
			if ok(c) {
				return c
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.t.Fatalf("%s did not reach the expected state; last %+v", id, last)
	return last
}

func (h *harness) value(id, want string) snoofer.Control {
	h.t.Helper()
	return h.waitControl(id, func(c snoofer.Control) bool { return c.Value == want })
}

func (h *harness) dispatch(id, operation string, delta int, value string) error {
	c, _ := h.find(id)
	return h.controls.Dispatch(context.Background(), snoofer.Request{ID: id, Revision: c.Revision, Operation: operation, Delta: delta, Value: value})
}

func (h *harness) mustDispatch(id, operation string, delta int, value string) {
	h.t.Helper()
	if err := h.dispatch(id, operation, delta, value); err != nil {
		h.t.Fatalf("%s %s: %v", id, operation, err)
	}
}

func (h *harness) settled(id string) snoofer.Control {
	h.t.Helper()
	return h.waitControl(id, func(c snoofer.Control) bool { return !c.Subdued && c.Status == "" })
}

func TestValidateSettings(t *testing.T) {
	good := `{"address":"","bridge_id":"","app_key":"","certificate_sha256":"","group":"","neutral_kelvin":4000}`
	if err := validate(json.RawMessage(good)); err != nil {
		t.Fatal(err)
	}
	if err := validate(Plugin().Defaults); err != nil {
		t.Fatalf("defaults: %v", err)
	}
	bad := []string{
		strings.Replace(good, "4000", "9000", 1),
		strings.Replace(good, `"app_key":""`, `"app_key":"k"`, 1),
		strings.Replace(good, `"address":""`, `"address":"a/b"`, 1),
		`{"neutral_kelvin":4000,"extra":1}`,
	}
	for _, raw := range bad {
		if err := validate(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestConnectedControls(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.status", "Connected")
	h.value("hue.brightness", "50%")
	h.value("hue.temperature", "4000K")
	h.value("hue.pair", "Paired")
	group := h.value("hue.group", "room-1")
	if len(group.Options) != 2 || group.OptionLabels["zone-1"] != "Desk (zone)" {
		t.Fatalf("group options %+v", group)
	}
	scene := h.value("hue.scene-desk-8b7e0d21", "Active")
	if scene.ShortLabel != "Focus" || scene.Icon != "hue-scene" {
		t.Fatalf("scene %+v", scene)
	}
	for _, c := range h.controls.Snapshot() {
		if strings.Contains(c.Value+c.Status, bridge.key) {
			t.Fatalf("application key exposed in %s", c.ID)
		}
	}
}

func TestPairingSavesIdentityAndConnects(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, Settings{NeutralKelvin: 4000, Group: "stale"}, true)
	h.value("hue.status", "Not paired")
	h.mustDispatch("hue.pair", "press", 0, "")
	h.value("hue.pair", "Press button")
	bridge.set(func(b *fakeBridge) { b.linkPressed = true })
	h.value("hue.status", "Connected")
	h.mu.Lock()
	saved := h.saved
	h.mu.Unlock()
	if saved.AppKey != bridge.key || saved.BridgeID != "b1" || saved.CertificateSHA256 != bridge.fingerprint() || saved.Group != "" {
		t.Fatalf("saved %+v", saved)
	}
	h.value("hue.brightness", "N/A")
	if c, _ := h.find("hue.brightness"); c.Available || c.Status != "Choose room" {
		t.Fatalf("brightness without room %+v", c)
	}
	h.mustDispatch("hue.group", "set", 0, "room-1")
	h.value("hue.brightness", "50%")
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.saved.Group != "room-1" {
		t.Fatalf("group not saved: %+v", h.saved)
	}
}

func TestPairingTimeoutKeepsSettings(t *testing.T) {
	bridge := newFakeBridge(t, "b1")
	h := startHarness(t, bridge, Settings{NeutralKelvin: 4000}, true)
	h.value("hue.status", "Not paired")
	h.mustDispatch("hue.pair", "press", 0, "")
	c := h.waitControl("hue.pair", func(c snoofer.Control) bool { return strings.Contains(c.Status, "not pressed") })
	if c.Value != "" {
		t.Fatalf("pair %+v", c)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.saved.AppKey != "" {
		t.Fatal("timeout saved a key")
	}
}

func TestBrightnessCoalescesFastRotation(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.brightness", "50%")
	for range 10 {
		h.mustDispatch("hue.brightness", "adjust", 1, "")
	}
	h.value("hue.brightness", "70%")
	h.settled("hue.brightness")
	puts := bridge.putLog()
	if len(puts) == 0 || len(puts) > 4 {
		t.Fatalf("%d writes for 10 ticks: %v", len(puts), puts)
	}
	if last := puts[len(puts)-1]; last != `grouped_light/gl-1:{"dimming":{"brightness":70}}` {
		t.Fatalf("last write %s", last)
	}
	h.mustDispatch("hue.brightness", "adjust", 60, "")
	h.value("hue.brightness", "100%")
}

func TestBrightnessOffRules(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.brightness", "50%")
	h.mustDispatch("hue.brightness", "press", 0, "")
	h.value("hue.brightness", "Off")
	h.settled("hue.brightness")
	if puts := bridge.putLog(); len(puts) != 1 || puts[0] != `grouped_light/gl-1:{"on":{"on":false}}` {
		t.Fatalf("press wrote %v", puts)
	}
	h.mustDispatch("hue.temperature", "adjust", 1, "")
	h.mustDispatch("hue.brightness", "adjust", -1, "")
	time.Sleep(3 * testTiming.writeGap)
	if puts := bridge.putLog(); len(puts) != 1 {
		t.Fatalf("rotation down or temperature while off wrote %v", puts)
	}
	h.mustDispatch("hue.brightness", "adjust", 1, "")
	h.value("hue.brightness", "52%")
	h.settled("hue.brightness")
	puts := bridge.putLog()
	if last := puts[len(puts)-1]; last != `grouped_light/gl-1:{"dimming":{"brightness":52},"on":{"on":true}}` {
		t.Fatalf("turn up while off wrote %s", last)
	}
}

func TestTemperatureKnob(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	settings := paired(bridge, "room-1")
	settings.NeutralKelvin = 2700
	h := startHarness(t, bridge, settings, true)
	h.value("hue.temperature", "4000K")
	h.mustDispatch("hue.temperature", "adjust", 3, "")
	h.value("hue.temperature", "4300K")
	h.settled("hue.temperature")
	h.mustDispatch("hue.temperature", "press", 0, "")
	h.value("hue.temperature", "2700K")
	h.settled("hue.temperature")
	puts := bridge.putLog()
	if puts[0] != `grouped_light/gl-1:{"color_temperature":{"mirek":233}}` || puts[len(puts)-1] != `grouped_light/gl-1:{"color_temperature":{"mirek":370}}` {
		t.Fatalf("writes %v", puts)
	}
	h.mustDispatch("hue.temperature", "adjust", -40, "")
	h.value("hue.temperature", "2200K") // Clamped to the room's common 454 mirek limit.
}

func TestThrottledWriteIsRetriedWithLatestValue(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.brightness", "50%")
	bridge.set(func(b *fakeBridge) { b.throttle = 2 })
	h.mustDispatch("hue.brightness", "adjust", 5, "")
	h.value("hue.brightness", "60%")
	h.settled("hue.brightness")
	if puts := bridge.putLog(); len(puts) != 3 {
		t.Fatalf("writes %v", puts)
	}
}

func TestUnconfirmedWriteReportsError(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.brightness", "50%")
	bridge.set(func(b *fakeBridge) { b.ignorePuts = true })
	h.mustDispatch("hue.brightness", "adjust", 5, "")
	c := h.waitControl("hue.brightness", func(c snoofer.Control) bool { return c.Status != "" })
	if c.Status != "Not applied: requested 60%, observed 50%" || c.Value != "50%" || c.Subdued {
		t.Fatalf("brightness %+v", c)
	}
	bridge.set(func(b *fakeBridge) { b.ignorePuts = false })
	h.mustDispatch("hue.brightness", "adjust", 1, "")
	h.value("hue.brightness", "52%")
	h.settled("hue.brightness")
}

func TestReconnectReloadsWithoutReplay(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.status", "Connected")
	waitFor(t, func() bool { return bridge.streamCount() == 1 })
	bridge.set(func(b *fakeBridge) { b.ignorePuts = true })
	h.mustDispatch("hue.brightness", "adjust", 5, "")
	waitFor(t, func() bool { return len(bridge.putLog()) == 1 })
	bridge.set(func(b *fakeBridge) {
		b.ignorePuts = false
		gl := b.resources["gl-1"]
		gl.Dimming = &dimming{Brightness: 20}
		b.resources["gl-1"] = gl
	})
	bridge.closeStreams()
	h.value("hue.brightness", "20%") // Reloaded state that changed during the gap.
	h.value("hue.status", "Connected")
	time.Sleep(3 * testTiming.writeGap)
	if puts := bridge.putLog(); len(puts) != 1 {
		t.Fatalf("pending write replayed: %v", puts)
	}
}

func TestSceneRecall(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.scene-studio-3f2a9c10", "Ready")
	h.mustDispatch("hue.scene-studio-3f2a9c10", "press", 0, "")
	h.value("hue.scene-studio-3f2a9c10", "Active")
	h.settled("hue.scene-studio-3f2a9c10")
	if puts := bridge.putLog(); len(puts) != 1 || puts[0] != `scene/3f2a9c10-aaaa-bbbb-cccc-000000000001:{"recall":{"action":"active"}}` {
		t.Fatalf("writes %v", puts)
	}
	bridge.set(func(b *fakeBridge) { b.ignorePuts = true })
	bridge.update(resource{ID: "3f2a9c10-aaaa-bbbb-cccc-000000000001", Status: &sceneStatus{Active: "inactive"}})
	h.value("hue.scene-studio-3f2a9c10", "Ready")
	h.mustDispatch("hue.scene-studio-3f2a9c10", "press", 0, "")
	h.waitControl("hue.scene-studio-3f2a9c10", func(c snoofer.Control) bool { return c.Status == "Pending" })
	h.waitControl("hue.scene-studio-3f2a9c10", func(c snoofer.Control) bool { return c.Status == "Not confirmed by bridge" })
}

func TestSceneListFollowsBridgeEvents(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.status", "Connected")
	waitFor(t, func() bool { return bridge.streamCount() == 1 })
	added := resource{ID: "c0ffee00-aaaa-bbbb-cccc-000000000003", Type: "scene", Metadata: &metadata{Name: "Night"},
		Group: &reference{RID: "room-1", RType: "room"}, Status: &sceneStatus{Active: "inactive"}}
	data, _ := json.Marshal([]event{{Type: "add", Data: []resource{added}}})
	bridge.set(func(b *fakeBridge) { b.streams[0] <- string(data) })
	h.value("hue.scene-studio-c0ffee00", "Ready")
}

func TestRoomMissingAndPreview(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "gone"), true)
	c := h.waitControl("hue.brightness", func(c snoofer.Control) bool { return c.Status == "Room missing" })
	if c.Available {
		t.Fatal("missing room is adjustable")
	}

	preview := startHarness(t, bridge, paired(bridge, "room-1"), false)
	preview.value("hue.brightness", "50%")
	for _, id := range []string{"hue.brightness", "hue.temperature", "hue.pair", "hue.group", "hue.scene-studio-3f2a9c10"} {
		if err := preview.dispatch(id, "press", 0, ""); err == nil {
			t.Errorf("preview accepted %s", id)
		}
	}
	if puts := bridge.putLog(); len(puts) != 0 {
		t.Fatalf("preview wrote %v", puts)
	}
}

func TestStaleRevisionRejected(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	old := h.value("hue.brightness", "50%")
	h.mustDispatch("hue.brightness", "adjust", 1, "")
	h.value("hue.brightness", "52%")
	err := h.controls.Dispatch(context.Background(), snoofer.Request{ID: old.ID, Revision: old.Revision, Operation: "adjust", Delta: 1})
	if err == nil {
		t.Fatal("stale request accepted")
	}
}

func TestCertificateChangeStopsConnection(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	settings := paired(bridge, "room-1")
	settings.CertificateSHA256 = strings.Repeat("ab", 32)
	h := startHarness(t, bridge, settings, true)
	c := h.value("hue.status", "Error")
	if !strings.Contains(c.Status, "certificate changed") {
		t.Fatalf("status %+v", c)
	}
}
