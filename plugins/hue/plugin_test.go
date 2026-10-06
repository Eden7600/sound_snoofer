package hue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

	syncDial:          time.Second,
	syncWrite:         time.Second,
	syncBrightnessGap: 60 * time.Millisecond,
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
	return Settings{Address: b.address(), BridgeID: b.id, AppKey: b.key, CertificateSHA256: b.fingerprint(), Group: group}
}

func startHarness(t *testing.T, bridge *fakeBridge, settings Settings, live bool) *harness {
	t.Helper()
	if settings.SyncPort == nil {
		// Never reach a real Hue Sync app installed on the test machine.
		port := closedPort(t)
		settings.SyncPort = &port
	}
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

// mustDispatch retries with a fresh revision, as an input surface does when
// the worker publishes between reading a control and dispatching to it.
func (h *harness) mustDispatch(id, operation string, delta int, value string) {
	h.t.Helper()
	var err error
	for range 20 {
		if err = h.dispatch(id, operation, delta, value); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.t.Fatalf("%s %s: %v", id, operation, err)
}

func (h *harness) settled(id string) snoofer.Control {
	h.t.Helper()
	return h.waitControl(id, func(c snoofer.Control) bool { return !c.Subdued && c.Status == "" })
}

func TestValidateSettings(t *testing.T) {
	good := `{"address":"","bridge_id":"","app_key":"","certificate_sha256":"","group":""}`
	if err := validate(json.RawMessage(good)); err != nil {
		t.Fatal(err)
	}
	if err := validate(Plugin().Defaults); err != nil {
		t.Fatalf("defaults: %v", err)
	}
	bad := []string{
		`{"neutral_kelvin":4000}`, // Removed setting; personal configs are converted manually.
		strings.Replace(good, `"app_key":""`, `"app_key":"k"`, 1),
		strings.Replace(good, `"address":""`, `"address":"a/b"`, 1),
		`{"extra":1}`,
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
	h := startHarness(t, bridge, Settings{Group: "stale"}, true)
	unpaired := h.value("hue.status", "Not paired")
	if !strings.Contains(string(unpaired.ViewData), `"bridge":"`+bridge.address()+`"`) {
		t.Fatalf("found bridge not reported: %s", unpaired.ViewData)
	}
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
	h := startHarness(t, bridge, Settings{}, true)
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
	h.mustDispatch("hue.brightness", "adjust", -1, "")
	time.Sleep(3 * testTiming.writeGap)
	if puts := bridge.putLog(); len(puts) != 1 {
		t.Fatalf("rotation down while off wrote %v", puts)
	}
	h.mustDispatch("hue.brightness", "adjust", 1, "")
	h.value("hue.brightness", "52%")
	h.settled("hue.brightness")
	puts := bridge.putLog()
	if last := puts[len(puts)-1]; last != `grouped_light/gl-1:{"dimming":{"brightness":52},"on":{"on":true}}` {
		t.Fatalf("turn up while off wrote %s", last)
	}
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
	for _, id := range []string{"hue.brightness", "hue.pair", "hue.group", "hue.scene-studio-3f2a9c10"} {
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

func TestRoomSceneSlots(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	first := h.waitControl("hue.room-scene-1", func(c snoofer.Control) bool { return c.Label == "Bright" })
	if !first.Available || first.Icon != "hue-scene" || first.Value != "Ready" {
		t.Fatalf("slot 1 %+v", first)
	}
	for _, id := range []string{"hue.room-scene-2", "hue.room-scene-12"} {
		if c, ok := h.find(id); !ok || c.Available || !c.Hidden || c.Label != "" || c.ShortLabel != "" || c.Icon != "" {
			t.Fatalf("%s should be blank: %+v", id, c)
		}
	}
	if first.Collection != "hue.room-scenes" || first.CollectionLabel != "Scenes · selected room" {
		t.Fatalf("slot collection %+v", first)
	}
	if scene, _ := h.find("hue.scene-studio-3f2a9c10"); scene.Collection != "hue.scenes.room-1" || scene.CollectionLabel != "Scenes · Studio" {
		t.Fatalf("room scene collection %+v", scene)
	}
	if _, ok := h.find("hue.room-scene-13"); ok {
		t.Fatal("more than twelve slots")
	}

	h.mustDispatch("hue.group", "set", 0, "zone-1")
	h.waitControl("hue.room-scene-1", func(c snoofer.Control) bool { return c.Label == "Focus" })
	stale := snoofer.Request{ID: first.ID, Revision: first.Revision, Operation: "press"}
	if err := h.controls.Dispatch(context.Background(), stale); err == nil {
		t.Fatal("press for the previous room's scene accepted")
	}
	bridge.update(resource{ID: "8b7e0d21-aaaa-bbbb-cccc-000000000002", Status: &sceneStatus{Active: "inactive"}})
	h.value("hue.room-scene-1", "Ready")
	h.mustDispatch("hue.room-scene-1", "press", 0, "")
	h.value("hue.room-scene-1", "Active")
	if puts := bridge.putLog(); len(puts) != 1 || !strings.HasPrefix(puts[0], "scene/8b7e0d21-aaaa-bbbb-cccc-000000000002:") {
		t.Fatalf("writes %v", puts)
	}
}

func TestRoomSceneSlotsGrowWithScenes(t *testing.T) {
	items := studio()
	for n := 0; n < 13; n++ {
		items = append(items, resource{ID: fmt.Sprintf("scene-%02d", n), Type: "scene", Metadata: &metadata{Name: fmt.Sprintf("Scene %02d", n)},
			Group: &reference{RID: "room-1", RType: "room"}})
	}
	bridge := newFakeBridge(t, "b1", items...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	last := h.waitControl("hue.room-scene-14", func(c snoofer.Control) bool { return c.Label != "" })
	if !last.Available || last.Hidden {
		t.Fatalf("fourteenth slot %+v", last)
	}
	if _, ok := h.find("hue.room-scene-15"); ok {
		t.Fatal("slot beyond the room's scenes")
	}
	h.mustDispatch("hue.room-scene-14", "press", 0, "")
	h.waitControl("hue.room-scene-14", func(c snoofer.Control) bool { return c.Value == "Active" })
}

func TestRoomSceneSlotsBlankWithoutRoom(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, ""), true)
	h.value("hue.status", "Connected")
	if c, _ := h.find("hue.room-scene-1"); c.Available || c.Label != "" {
		t.Fatalf("slot without room %+v", c)
	}
}

func TestRealBridgeShapesLoadAndStream(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	bridge.set(func(b *fakeBridge) {
		b.extra = []json.RawMessage{
			json.RawMessage(`{"id":"zc-1","type":"zigbee_connectivity","status":"connected"}`),
			json.RawMessage(`{"id":"ec-1","type":"entertainment_configuration","status":"inactive"}`),
		}
	})
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.status", "Connected")
	waitFor(t, func() bool { return bridge.streamCount() == 1 })
	bridge.pushRaw(`[{"type":"update","data":[{"id":"zc-1","type":"zigbee_connectivity","status":"connectivity_issue"}]}]`)
	bridge.update(resource{ID: "gl-1", Type: "grouped_light", Dimming: &dimming{Brightness: 30}})
	h.value("hue.brightness", "30%")
	if c, _ := h.find("hue.status"); c.Value != "Connected" || bridge.streamCount() != 1 {
		t.Fatalf("stream did not survive an unrelated status update: %+v", c)
	}
}

// TestLiveResources loads the paired bridge named by a Snoofer config file
// (SNOOFER_HUE_CONFIG). It prints resource type counts only, never settings.
func TestLiveResources(t *testing.T) {
	path := os.Getenv("SNOOFER_HUE_CONFIG")
	if path == "" {
		t.Skip("set SNOOFER_HUE_CONFIG to a paired snoofer.json")
	}
	data, err := os.ReadFile(path)
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
	if settings.AppKey == "" {
		t.Skip("hue is not paired in that config")
	}
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
	counts := map[string]int{}
	for _, item := range items {
		counts[item.Type]++
	}
	m := newModel(items)
	t.Logf("bridge %s: %v; rooms/zones %d; scenes %d", target.Address, counts, len(m.groups()), len(m.scenes()))
}

func TestNoTemperatureControl(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	h.value("hue.brightness", "50%")
	if _, ok := h.find("hue.temperature"); ok {
		t.Fatal("temperature control published")
	}
}

func TestPairedBridgeReconnectsWithoutDiscovery(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	settings := paired(bridge, "room-1")
	settings.Address = "" // Use discovery, whose first reply is lost.
	port := closedPort(t)
	settings.SyncPort = &port
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	calls := 0
	discover := func(context.Context) ([]string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls != 2 {
			return nil, nil // Only the second query is answered.
		}
		return []string{bridge.address()}, nil
	}
	controls := snoofer.NewControls()
	instance, err := start(context.Background(), snoofer.Services{Controls: controls, Live: true}, raw, discover, testTiming)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := instance.Stop(ctx); err != nil {
			t.Error(err)
		}
	}()
	h := &harness{t: t, controls: controls}
	// testTiming.rediscover is an hour, so only the paired backoff can reconnect in time.
	h.value("hue.status", "Connected")

	// Discovery no longer answers; a stream drop must reconnect through the last address.
	waitFor(t, func() bool { return bridge.streamCount() == 1 })
	bridge.closeStreams()
	waitFor(t, func() bool { return bridge.streamCount() == 1 })
	h.value("hue.status", "Connected")
}
