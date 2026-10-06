package hue

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"sound-snoofer/snoofer"
)

// Settings configures both halves of the Hue plugin. Pairing writes the bridge
// identity, key and certificate pin; the Lights screen writes Group. Empty
// strings mean not configured.
type Settings struct {
	Address           string `json:"address"`
	BridgeID          string `json:"bridge_id"`
	AppKey            string `json:"app_key"`
	CertificateSHA256 string `json:"certificate_sha256"`
	Group             string `json:"group"`
	SyncPort          *int   `json:"sync_port,omitempty"` // Hue Sync third-party control port; nil means 24851.
}

// Plugin returns inert metadata; no network activity occurs until Start.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{
		ID:       "hue",
		Label:    "Hue",
		Validate: validate,
		Defaults: snoofer.MarshalSettings(Settings{}),
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
			discover := func(ctx context.Context) ([]string, error) { return discoverMDNS(ctx, 3*time.Second) }
			return start(ctx, s, raw, discover, defaultTiming)
		},
	}
}

func decode(raw json.RawMessage) (Settings, error) {
	var s Settings
	if err := snoofer.DecodeSettings(raw, &s); err != nil {
		return s, err
	}
	if s.SyncPort != nil && (*s.SyncPort < 1 || *s.SyncPort > 65535) {
		return s, fmt.Errorf("hue sync_port must be 1-65535")
	}
	if strings.ContainsAny(s.Address, " /\\\x00") {
		return s, fmt.Errorf("hue address must be a host name or IP address")
	}
	if s.AppKey != "" {
		pin, err := hex.DecodeString(s.CertificateSHA256)
		if err != nil || len(pin) != 32 || s.BridgeID == "" {
			return s, fmt.Errorf("hue pairing is incomplete; pair again")
		}
	}
	return s, nil
}

func validate(raw json.RawMessage) error {
	_, err := decode(raw)
	return err
}

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Stop cancels the worker and waits for it and its requests to finish.
func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, discover discoverFunc, t timing) (snoofer.Instance, error) {
	settings, err := decode(raw)
	if err != nil {
		return nil, err
	}
	w := newWorker(s, settings, raw, discover, t)
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(i.done)
		w.run(runCtx)
	}()
	return i, nil
}

// worker owns all bridge and Hue Sync state. Child goroutines perform network I/O and
// return their results as functions executed on the worker goroutine.
type worker struct {
	services   snoofer.Services
	settings   Settings
	raw        json.RawMessage // Settings payload last saved, for atomic replacement.
	discover   discoverFunc
	timing     timing
	deviceType string

	requests chan snoofer.Request
	results  chan func(context.Context)
	children sync.WaitGroup

	generation    int // Invalidates results from superseded connections.
	connecting    bool
	connected     bool
	client        *Client
	model         model
	status        string
	bridgeAddress string                    // Bridge found while unpaired, shown by the pairing UI.
	lastAddress   string                    // Address of the paired bridge last connected this session.
	bridgeInfo    identity                  // Identity of the connected bridge, for diagnostics.
	bridgeLink    snoofer.ConnectionTracker // Bridge connection report timing.
	connects      int                       // Successful bridge connections this session.
	diagnostic    string
	retryAt       time.Time
	retryDelay    time.Duration

	pairing   bool
	pairValue string
	pairErr   string

	group     groupRequest
	brightErr string
	groupErr  string

	sceneWriting string               // Scene ID with a recall request in flight.
	scenePending map[string]time.Time // Recalled scenes awaiting an active status.
	sceneErr     map[string]string

	sync syncLink // Hue Sync PC app half.

	artwork map[string]cachedArtwork // Scene thumbnails by scene ID.
}

func newWorker(s snoofer.Services, settings Settings, raw json.RawMessage, discover discoverFunc, t timing) *worker {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "pc"
	}
	if len(host) > 19 {
		host = host[:19] // Hue limits the device part of devicetype to 19 characters.
	}
	return &worker{
		services:     s,
		settings:     settings,
		raw:          append(json.RawMessage(nil), raw...),
		discover:     discover,
		timing:       t,
		deviceType:   "snoofer#" + host,
		requests:     make(chan snoofer.Request, 16),
		results:      make(chan func(context.Context), 16),
		model:        newModel(nil),
		status:       "Disconnected",
		scenePending: map[string]time.Time{},
		sceneErr:     map[string]string{},
		sync:         syncLink{url: syncURL(settings)},
		artwork:      map[string]cachedArtwork{},
	}
}

func (w *worker) run(ctx context.Context) {
	defer w.services.Controls.Remove("hue")
	defer w.closeClient()
	defer w.children.Wait() // Children observe ctx, which Stop has canceled.
	defer w.syncClose()     // Runs first, unblocking the sync reader before Wait.
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	w.connect(ctx)
	w.syncConnect(ctx)
	w.publish()
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-w.requests:
			w.handle(ctx, r)
		case apply := <-w.results:
			apply(ctx)
		case now := <-ticker.C:
			if !w.tick(ctx, now) {
				continue
			}
		}
		w.publish()
	}
}

// spawn runs network work off the worker goroutine and delivers its result.
func (w *worker) spawn(ctx context.Context, work func(context.Context) func(context.Context)) {
	w.children.Add(1)
	go func() {
		defer w.children.Done()
		apply := work(ctx)
		select {
		case w.results <- apply:
		case <-ctx.Done():
		}
	}()
}

// enqueue is the control handler. It runs under the registry lock and must not block.
func (w *worker) enqueue(ctx context.Context, r snoofer.Request) error {
	select {
	case w.requests <- r:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.New("hue queue full")
	}
}

func (w *worker) closeClient() {
	if w.client != nil {
		w.client.close()
		w.client = nil
	}
}

// connect resolves the bridge and, when paired, loads resources and follows events.
func (w *worker) connect(ctx context.Context) {
	if w.connecting {
		return
	}
	w.connecting = true
	w.generation++
	generation := w.generation
	settings := w.settings
	discover := w.discover
	hint := w.lastAddress
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		target, err := resolveKnown(ctx, settings, hint, discover)
		if err != nil || settings.AppKey == "" {
			return func(ctx context.Context) { w.onConnected(ctx, generation, target, nil, nil, err) }
		}
		client := newClient(target.Address, settings.CertificateSHA256, settings.AppKey)
		items, err := client.Resources(ctx)
		if err != nil {
			client.close()
			client = nil
		}
		return func(ctx context.Context) { w.onConnected(ctx, generation, target, client, items, err) }
	})
}

// onConnected applies a connection attempt's result on the worker goroutine.
func (w *worker) onConnected(ctx context.Context, generation int, target bridgeTarget, client *Client, items []resource, err error) {
	if generation != w.generation {
		if client != nil {
			client.close()
		}
		return
	}
	w.connecting = false
	if err != nil {
		w.fail(err)
		return
	}
	if client == nil {
		w.status = "Not paired"
		w.bridgeAddress = target.Address
		w.bridgeInfo = target.identity
		w.diagnostic = ""
		w.retryAt = time.Now().Add(w.timing.rediscover)
		return
	}
	w.client = client
	w.lastAddress = target.Address
	w.bridgeInfo = target.identity
	w.connects++
	w.bridgeLink.Activity(time.Now())
	w.connected = true
	w.model = newModel(items)
	w.status = "Connected"
	w.diagnostic = ""
	w.retryDelay = 0
	w.observe()
	w.follow(ctx, generation, client)
}

// follow streams events until the stream ends; any gap forces a full reload.
func (w *worker) follow(ctx context.Context, generation int, client *Client) {
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		err := client.Events(ctx, func(events []event) error {
			apply := func(context.Context) {
				if generation == w.generation && w.connected {
					w.model.apply(events)
					w.bridgeLink.Activity(time.Now())
					w.observe()
				}
			}
			select {
			case w.results <- apply:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		return func(context.Context) {
			if generation == w.generation && w.connected {
				w.disconnect(fmt.Errorf("event stream ended: %w", err))
			}
		}
	})
}

// disconnect drops observed state and pending requests; nothing is replayed.
func (w *worker) disconnect(err error) {
	w.generation++
	w.connected = false
	w.connecting = false
	w.closeClient()
	w.group = groupRequest{}
	w.sceneWriting = ""
	clear(w.scenePending)
	w.fail(err)
}

func (w *worker) fail(err error) {
	w.bridgeLink.Fail(err.Error(), time.Now())
	var multiple *multipleBridgesError
	var mismatch *bridgeMismatchError
	now := time.Now()
	w.diagnostic = err.Error()
	switch {
	case errors.Is(err, errNoBridge):
		w.status = "No bridge"
		w.diagnostic = ""
		w.retryAt = now.Add(w.timing.rediscover)
		if w.settings.AppKey != "" {
			// A paired bridge should exist; a missed reply must not cost a full rediscovery delay.
			w.retryDelay = min(w.timing.retryMax, max(w.timing.retryBase, 2*w.retryDelay))
			w.retryAt = now.Add(w.retryDelay)
		}
	case errors.As(err, &multiple):
		w.status = "Multiple bridges"
		w.retryAt = now.Add(w.timing.rediscover)
	case errors.Is(err, ErrCertificateChanged), errors.As(err, &mismatch):
		w.status = "Error"
		w.retryAt = time.Time{} // Requires pairing or configuration; never retried.
	default:
		w.status = "Disconnected"
		w.retryDelay = min(w.timing.retryMax, max(w.timing.retryBase, 2*w.retryDelay))
		w.retryAt = now.Add(w.retryDelay)
	}
}

// reconnect discards the current connection and connects with current settings.
func (w *worker) reconnect(ctx context.Context) {
	w.generation++
	w.connected = false
	w.connecting = false
	w.closeClient()
	w.group = groupRequest{}
	w.status = "Disconnected"
	w.diagnostic = ""
	w.connect(ctx)
}

// tick schedules retries and writes and expires confirmations. It reports changes.
func (w *worker) tick(ctx context.Context, now time.Time) bool {
	changed := false
	if !w.connected && !w.connecting && !w.retryAt.IsZero() && now.After(w.retryAt) {
		w.retryAt = time.Time{}
		w.connect(ctx)
		changed = true
	}
	if w.connected && w.group.dirty && !w.group.inFlight && now.Sub(w.group.lastSend) >= w.group.gap {
		w.sendGroup(ctx)
		changed = true
	}
	if w.group.pending() && !w.group.dirty && !w.group.inFlight && now.Sub(w.group.lastSend) > w.timing.confirm {
		w.expireGroup()
		changed = true
	}
	for id, sent := range w.scenePending {
		if now.Sub(sent) > w.timing.confirm {
			delete(w.scenePending, id)
			w.sceneErr[id] = "Not confirmed by bridge"
			changed = true
		}
	}
	if w.syncTick(ctx, now) {
		changed = true
	}
	return changed
}

func (w *worker) view() groupView {
	if !w.connected || w.settings.Group == "" {
		return groupView{}
	}
	return w.model.group(w.settings.Group)
}

// observe confirms pending requests against the latest bridge state.
func (w *worker) observe() {
	view := w.view()
	if view.GroupedLight == w.group.target {
		w.group.confirm(view)
	}
	for _, scene := range w.model.scenes() {
		if scene.Active {
			delete(w.scenePending, scene.SceneID)
		}
	}
}

func (w *worker) handle(ctx context.Context, r snoofer.Request) {
	if !w.services.Live {
		return
	}
	switch {
	case r.ID == "hue.pair":
		w.startPairing(ctx)
	case r.ID == "hue.group":
		w.selectGroup(r.Value)
	case r.ID == "hue.brightness":
		w.adjustBrightness(r)
	case strings.HasPrefix(r.ID, "hue.sync"):
		w.handleSync(r)
	case strings.HasPrefix(r.ID, "hue.scene-"):
		w.recallScene(ctx, w.sceneForControl(r.ID))
	case strings.HasPrefix(r.ID, "hue.room-scene-"):
		w.recallScene(ctx, w.sceneForSlot(r.ID))
	}
}

func (w *worker) save(next Settings) error {
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := w.services.SaveSettings("hue", w.raw, raw); err != nil {
		return err
	}
	w.raw = raw
	w.settings = next
	return nil
}

func (w *worker) selectGroup(id string) {
	next := w.settings
	next.Group = id
	if err := w.save(next); err != nil {
		w.groupErr = err.Error()
		return
	}
	w.groupErr = ""
	w.group = groupRequest{}
	w.brightErr = ""
}

// retarget starts a fresh request when the configured group's light service changes.
func (w *worker) retarget(groupedLight string) {
	if w.group.target != groupedLight {
		w.group = groupRequest{target: groupedLight, gap: w.timing.writeGap}
	}
}

// effectiveOn includes a pending on/off request.
func (w *worker) effectiveOn(view groupView) (on, known bool) {
	if w.group.on != nil {
		return *w.group.on, true
	}
	return view.On, view.OnKnown
}

func (w *worker) adjustBrightness(r snoofer.Request) {
	if r.Operation == "adjust" && w.syncing() {
		w.sync.stepsDue += r.Delta * syncBrightnessStep // The sync stream owns the lights.
		return
	}
	view := w.view()
	if view.GroupedLight == "" {
		return
	}
	w.retarget(view.GroupedLight)
	w.brightErr = ""
	on, known := w.effectiveOn(view)
	if !known {
		w.brightErr = "Room state unknown"
		return
	}
	if r.Operation == "press" {
		next := !on
		w.group.on = &next
		w.group.dirty = true
		return
	}
	if r.Operation != "adjust" || r.Delta == 0 {
		return
	}
	if !on && r.Delta < 0 {
		return
	}
	base := 0.0
	switch {
	case w.group.brightness != nil:
		base = *w.group.brightness
	case view.BrightKnown:
		base = view.Brightness
	case on:
		w.brightErr = "Brightness unknown"
		return
	}
	if !on {
		turnOn := true
		w.group.on = &turnOn
	}
	next := nextBrightness(base, r.Delta)
	w.group.brightness = &next
	w.group.dirty = true
}

// sendGroup writes the latest targets; at most one write is outstanding.
func (w *worker) sendGroup(ctx context.Context) {
	client, target, generation := w.client, w.group.target, w.generation
	body := w.group.body()
	w.group.dirty = false
	w.group.inFlight = true
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		err := client.Put(ctx, "grouped_light", target, body)
		return func(context.Context) { w.groupWritten(generation, target, err) }
	})
}

func (w *worker) groupWritten(generation int, target string, err error) {
	if generation != w.generation || target != w.group.target {
		return
	}
	w.group.inFlight = false
	w.group.lastSend = time.Now()
	switch {
	case errors.Is(err, ErrThrottled):
		w.group.gap = min(w.timing.maxWriteGap, 2*w.group.gap)
		w.group.dirty = true // Resend the latest targets after the longer gap.
		return
	case err != nil:
		w.brightErr = err.Error()
		w.group = groupRequest{target: target, gap: w.timing.writeGap}
		return
	}
	w.group.gap = w.timing.writeGap
	w.bridgeLink.Activity(time.Now())
	w.group.confirm(w.view())
}

// expireGroup reports targets the bridge did not confirm and drops them, so
// later ticks start from observed values.
func (w *worker) expireGroup() {
	view := w.view()
	w.brightErr = fmt.Sprintf("Not applied: requested %s, observed %s", w.requestedBrightness(), observedBrightness(view))
	w.group = groupRequest{target: w.group.target, gap: w.timing.writeGap}
}

func (w *worker) requestedBrightness() string {
	if w.group.on != nil && !*w.group.on {
		return "Off"
	}
	if w.group.brightness != nil {
		return percentLabel(*w.group.brightness)
	}
	return "On"
}

func observedBrightness(view groupView) string {
	switch {
	case !view.OnKnown:
		return "N/A"
	case !view.On:
		return "Off"
	case view.BrightKnown:
		return percentLabel(view.Brightness)
	}
	return "N/A"
}

func (w *worker) sceneForControl(controlID string) string {
	for _, s := range w.model.scenes() {
		if s.ControlID == controlID {
			return s.SceneID
		}
	}
	return ""
}

// recallScene recalls a scene. While Hue Sync is syncing, the stream would
// override the scene, so sync is stopped first and the recall waits for the
// app to confirm.
func (w *worker) recallScene(ctx context.Context, sceneID string) {
	if !w.connected || sceneID == "" || w.sceneWriting != "" || w.sync.sceneAfterStop != "" {
		return
	}
	delete(w.sceneErr, sceneID)
	delete(w.scenePending, sceneID)
	if w.syncing() {
		if w.syncSet(false) {
			w.sync.sceneAfterStop = sceneID
			w.sync.sceneStopAt = time.Now()
		}
		return
	}
	w.sendRecall(ctx, sceneID)
}

func (w *worker) sendRecall(ctx context.Context, sceneID string) {
	if !w.connected {
		w.sceneErr[sceneID] = "Bridge disconnected"
		return
	}
	w.sceneWriting = sceneID
	client, generation := w.client, w.generation
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		err := client.Put(ctx, "scene", sceneID, map[string]any{"recall": map[string]string{"action": "active"}})
		return func(context.Context) { w.sceneRecalled(generation, sceneID, err) }
	})
}

func (w *worker) sceneRecalled(generation int, sceneID string, err error) {
	if generation != w.generation {
		return
	}
	w.sceneWriting = ""
	if err != nil {
		w.sceneErr[sceneID] = err.Error()
		return
	}
	if active, ok := w.model.resources[sceneID]; ok && active.Status != nil && active.Status.Active != "inactive" {
		return // Already reported active; the recall needs no further confirmation.
	}
	w.scenePending[sceneID] = time.Now()
}

func (w *worker) startPairing(ctx context.Context) {
	if w.pairing {
		return
	}
	w.pairing = true
	w.pairValue = "Press button"
	w.pairErr = ""
	address, deviceType, discover, window := w.settings.Address, w.deviceType, w.discover, w.timing.pairWindow
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		target, key, err := pair(ctx, address, deviceType, discover, window)
		return func(ctx context.Context) { w.paired(ctx, target, key, err) }
	})
}

// pair resolves any single bridge and polls for a key while the link button window is open.
func pair(ctx context.Context, address, deviceType string, discover discoverFunc, window time.Duration) (bridgeTarget, string, error) {
	ctx, cancel := context.WithTimeout(ctx, window)
	defer cancel()
	target, err := resolve(ctx, address, "", discover)
	if err != nil {
		return target, "", err
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		key, err := requestKey(ctx, target.Address, target.Fingerprint, deviceType)
		if err == nil {
			return target, key, nil
		}
		// A request cut off by the pairing window is a timeout, not a bridge error.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return target, "", fmt.Errorf("bridge button not pressed within %s", window)
		}
		if !errors.Is(err, ErrLinkButton) {
			return target, "", err
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return target, "", fmt.Errorf("bridge button not pressed within %s", window)
			}
			return target, "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *worker) paired(ctx context.Context, target bridgeTarget, key string, err error) {
	w.pairing = false
	w.pairValue = ""
	if err != nil {
		w.pairErr = err.Error()
		return
	}
	next := w.settings
	if next.BridgeID != target.BridgeID {
		next.Group = "" // Rooms belong to the previous bridge.
	}
	next.BridgeID = target.BridgeID
	next.AppKey = key
	next.CertificateSHA256 = target.Fingerprint
	if err := w.save(next); err != nil {
		w.pairErr = err.Error()
		return
	}
	w.reconnect(ctx)
}

func (w *worker) publish() {
	// Publish fails only while the host is stopping, which also cancels this worker.
	_ = w.services.Controls.Publish("hue", w.controls(), w.enqueue)
}

func (w *worker) controls() []snoofer.Control {
	live := w.services.Live
	view := w.view()
	statusNote := ""
	if w.status == "Error" || w.status == "Disconnected" || w.status == "Multiple bridges" {
		statusNote = w.diagnostic
	}
	pairValue := w.pairValue
	if pairValue == "" && w.pairErr == "" {
		pairValue = "Ready"
		if w.settings.AppKey != "" {
			pairValue = "Paired"
		}
	}
	options := []string{}
	labels := map[string]string{}
	for _, g := range w.model.groups() {
		options = append(options, g.ID)
		labels[g.ID] = g.Name
		if g.Kind == "zone" {
			labels[g.ID] = g.Name + " (zone)"
		}
	}
	controls := []snoofer.Control{
		{ID: "hue.status", Label: "Hue", Group: "Hue", Kind: "status", Value: w.status, Status: statusNote, ViewData: w.statusView(), Available: true},
		{ID: "hue.pair", Label: "Pair Hue bridge", ShortLabel: "Pair", Group: "Hue", Kind: "command", Icon: "hue-pair", Value: pairValue, Status: w.pairErr, Operations: []string{"press"}, Available: live},
		{ID: "hue.group", Label: "Hue room", ShortLabel: "Room", Group: "Hue", Kind: "selection", Value: w.settings.Group, Options: options, OptionLabels: labels, Status: w.groupErr, Operations: []string{"set"}, Available: live && w.connected},
	}
	knobNote := ""
	switch {
	case !w.connected:
	case w.settings.Group == "":
		knobNote = "Choose room"
	case !view.Found || view.GroupedLight == "":
		knobNote = "Room missing"
	}
	ready := live && w.connected && view.GroupedLight != ""
	syncing := w.syncing()
	brightness := snoofer.Control{ID: "hue.brightness", Label: "Hue brightness", ShortLabel: "Brightness", Group: "Hue", Kind: "numeric", Icon: "hue-brightness",
		Value: w.brightnessValue(view), Status: firstNonEmpty(knobNote, w.brightErr), Subdued: w.group.on != nil || w.group.brightness != nil,
		Operations: []string{"adjust", "press"}, Available: ready}
	if syncing {
		// While syncing the dial adjusts the stream; a press still toggles the room.
		brightness.Value = w.syncBrightnessLabel()
		brightness.Status = ""
		brightness.Subdued = w.sync.stepsDue != 0
		brightness.Available = live
	}
	controls = append(controls, brightness)
	controls = append(controls, w.syncControls()...)
	now := time.Now()
	controls = append(controls, w.bridgeReport(now), w.syncReport(now))
	seen := map[string]bool{}
	for _, scene := range w.model.scenes() {
		if seen[scene.ControlID] {
			continue // A colliding ID prefix would invalidate the whole snapshot.
		}
		seen[scene.ControlID] = true
		controls = append(controls, w.sceneControl(scene.ControlID, scene.Label, scene))
	}
	return append(controls, w.slotControls()...)
}

// roomSlots is the number of stable slots mirroring the selected room's scenes.
const roomSlots = 12

func (w *worker) sceneControl(id, label string, scene sceneInfo) snoofer.Control {
	value := "Ready"
	if scene.Active {
		value = "Active"
	}
	status := w.sceneErr[scene.SceneID]
	if _, waiting := w.scenePending[scene.SceneID]; waiting || w.sceneWriting == scene.SceneID || w.sync.sceneAfterStop == scene.SceneID {
		status = "Pending"
	}
	return snoofer.Control{ID: id, Label: label, ShortLabel: scene.ShortLabel, Group: "Hue scenes", Kind: "command", Icon: "hue-scene", Artwork: w.sceneArtwork(scene.SceneID),
		Value: value, Status: status, Operations: []string{"press"}, Available: w.services.Live && w.connected}
}

// slotControls maps hue.room-scene-1..12 to the selected room's scenes. The
// scene ID in ViewData changes the slot's revision whenever its mapping
// changes, so input generated for a previous scene is rejected as stale.
// Unused slots have no label or icon, which the deck renders as a blank key.
func (w *worker) slotControls() []snoofer.Control {
	var scenes []sceneInfo
	if w.connected && w.settings.Group != "" {
		scenes = w.model.roomScenes(w.settings.Group)
	}
	controls := make([]snoofer.Control, 0, roomSlots)
	for n := 1; n <= roomSlots; n++ {
		id := "hue.room-scene-" + strconv.Itoa(n)
		if n > len(scenes) {
			controls = append(controls, snoofer.Control{ID: id, Group: "Hue room scenes", Kind: "command", Operations: []string{"press"}})
			continue
		}
		scene := scenes[n-1]
		control := w.sceneControl(id, scene.ShortLabel, scene)
		control.Group = "Hue room scenes"
		control.ViewData = json.RawMessage(strconv.Quote(scene.SceneID))
		controls = append(controls, control)
	}
	return controls
}

// sceneForSlot resolves a slot control to the scene it currently mirrors.
func (w *worker) sceneForSlot(controlID string) string {
	n, err := strconv.Atoi(strings.TrimPrefix(controlID, "hue.room-scene-"))
	if err != nil || !w.connected || w.settings.Group == "" {
		return ""
	}
	scenes := w.model.roomScenes(w.settings.Group)
	if n < 1 || n > len(scenes) || n > roomSlots {
		return ""
	}
	return scenes[n-1].SceneID
}

func (w *worker) brightnessValue(view groupView) string {
	if !w.connected || view.GroupedLight == "" {
		return "N/A"
	}
	if w.group.on != nil && !*w.group.on {
		return "Off"
	}
	if w.group.brightness != nil {
		return percentLabel(*w.group.brightness)
	}
	if w.group.on != nil && view.BrightKnown {
		return percentLabel(view.Brightness)
	}
	return observedBrightness(view)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// statusView gives the Lights screen setup details that do not belong in deck text.
func (w *worker) statusView() json.RawMessage {
	view := map[string]string{}
	if w.status == "Not paired" && w.bridgeAddress != "" {
		view["bridge"] = w.bridgeAddress
	}
	if w.settings.Address != "" {
		view["address"] = w.settings.Address
	}
	if len(view) == 0 {
		return nil
	}
	data, err := json.Marshal(view)
	if err != nil {
		return nil // A map of strings always marshals.
	}
	return data
}

// resolveKnown tries the paired bridge at the address it last answered on
// before discovery, so reconnects do not depend on mDNS replies. A configured
// address still takes precedence.
func resolveKnown(ctx context.Context, settings Settings, hint string, discover discoverFunc) (bridgeTarget, error) {
	if hint != "" && settings.Address == "" && settings.BridgeID != "" {
		if id, err := probe(ctx, hint); err == nil && id.BridgeID == settings.BridgeID {
			return bridgeTarget{Address: hint, identity: id}, nil
		}
	}
	return resolve(ctx, settings.Address, settings.BridgeID, discover)
}
