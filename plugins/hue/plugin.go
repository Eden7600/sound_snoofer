package hue

import (
	"cmp"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
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
	Address           string   `json:"address"`
	BridgeID          string   `json:"bridge_id"`
	AppKey            string   `json:"app_key"`
	CertificateSHA256 string   `json:"certificate_sha256"`
	Group             string   `json:"group"`
	Rooms             []string `json:"rooms,omitempty"`      // Rooms and zones Snoofer controls; empty means all.
	SyncPort          *int     `json:"sync_port,omitempty"`  // Hue Sync third-party control port; nil means 24851.
	SceneSort         string   `json:"scene_sort,omitempty"` // name, hue or brightness; empty means name.
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
	if s.SceneSort != "" && !slices.Contains(sceneSorts, s.SceneSort) {
		return s, fmt.Errorf("hue scene_sort must be name, hue or brightness")
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
	return snoofer.CancelAndWait(ctx, i.cancel, i.done)
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

	settingsEpoch, roomEpoch, nextContext uint64
	contexts                              map[string]actionContext
	sceneAction                           *command
	motionAction                          *motionOperation
	requests                              chan command
	results                               chan func(context.Context)
	children                              sync.WaitGroup

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
	sortErr   string
	roomsErr  string

	sceneWriting string               // Scene ID with a recall request in flight.
	scenePending map[string]time.Time // Recalled scenes awaiting an active status.
	sceneErr     map[string]string

	motion motionLink // Requested motion sensor states in the selected room.

	stages   map[string]stage // Staged light-group dial targets, by control ID.
	stagedAt time.Time        // Last turn of any light-group dial; stages expire together.
	lights   lightQueue       // Paced per-light writes for the light-group dials.

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
		requests:     make(chan command, 16),
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
	w.publish(ctx)
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
		w.publish(ctx)
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
	w.resetMotion()
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
	w.observe()
	w.follow(ctx, generation, client)
}

// follow streams events until the stream ends; any gap forces a full reload.
func (w *worker) follow(ctx context.Context, generation int, client *Client) {
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		err := client.Events(ctx, func(events []event) error {
			apply := func(context.Context) {
				if generation == w.generation && w.connected {
					// Only a delivering stream proves the link; one that fails at once keeps backing off.
					w.retryDelay = 0
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
	w.resetGroupDials()
	w.finishScene(snoofer.Outcome{Status: snoofer.Unknown, Detail: "Hue bridge disconnected"})
	clear(w.scenePending)
	w.resetMotion()
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
	w.finishScene(snoofer.Outcome{Status: snoofer.Unknown, Detail: "Hue bridge replaced"})
	w.generation++
	w.connected = false
	w.connecting = false
	w.closeClient()
	w.group = groupRequest{}
	w.resetGroupDials()
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
	if w.expireMotion(now) {
		changed = true
	}
	if w.tickLights(ctx, now) {
		changed = true
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
	w.observeMotion()
	w.observeLights()
	w.enforceRoom()
}

func (w *worker) save(next Settings) error {
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if w.services.SaveSettings == nil {
		return errors.New("Settings save unavailable")
	}
	if err := w.services.SaveSettings("hue", w.raw, raw); err != nil {
		return err
	}
	w.raw = raw
	w.settings = next
	w.settingsEpoch++
	return nil
}

func (w *worker) selectGroup(id string) error {
	next := w.settings
	next.Group = id
	if err := w.save(next); err != nil {
		w.groupErr = err.Error()
		return err
	}
	w.roomEpoch++
	w.groupErr = ""
	w.group = groupRequest{}
	w.resetGroupDials()
	w.resetMotion()
	w.brightErr = ""
	return nil
}

func (w *worker) setSceneSort(mode string) error {
	next := w.settings
	next.SceneSort = mode
	if err := w.save(next); err != nil {
		w.sortErr = err.Error()
		return err
	}
	w.sortErr = ""
	return nil
}

// sceneSort is the chosen scene sort, defaulting to name.
func (w *worker) sceneSort() string {
	if w.settings.SceneSort == "" {
		return sortName
	}
	return w.settings.SceneSort
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

func (w *worker) adjustBrightness(r snoofer.Request, want bool) error {
	if r.Operation == "adjust" && w.syncing() {
		w.sync.stepsDue += r.Delta * syncBrightnessStep
		return nil
	}
	if w.group.context != w.capture(r.ID).target {
		w.group = groupRequest{}
	}
	view := w.view()
	if view.GroupedLight == "" {
		return errors.New("Room unavailable")
	}
	w.retarget(view.GroupedLight)
	w.brightErr = ""
	on, known := w.effectiveOn(view)
	if !known {
		w.brightErr = "Room state unknown"
		return errors.New(w.brightErr)
	}
	w.group.context = w.capture(r.ID).target
	if r.Operation == "press" {
		w.group.on = &want
		w.group.dirty = true
		return nil
	}
	if r.Operation != "adjust" || r.Delta == 0 {
		return errors.New("No brightness change")
	}
	if !on && r.Delta < 0 {
		return errors.New("Room already off")
	}
	base := 0.0
	switch {
	case w.group.brightness != nil:
		base = *w.group.brightness
	case view.BrightKnown:
		base = view.Brightness
	case on:
		w.brightErr = "Brightness unknown"
		return errors.New(w.brightErr)
	}
	if !on {
		turnOn := true
		w.group.on = &turnOn
	}
	next := nextBrightness(base, r.Delta)
	w.group.brightness = &next
	w.group.dirty = true
	return nil
}

// sendGroup writes the latest targets; at most one write is outstanding.
func (w *worker) sendGroup(ctx context.Context) {
	if w.group.context != w.capture("hue.brightness").target {
		w.group = groupRequest{}
		return
	}
	client, target, generation := w.client, w.group.target, w.generation
	identity := w.group.context
	body := w.group.body()
	w.group.dirty = false
	w.group.inFlight = true
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		err := client.Put(ctx, "grouped_light", target, body)
		return func(context.Context) { w.groupWritten(generation, target, identity, err) }
	})
}

func (w *worker) groupWritten(generation int, target string, identity actionTarget, err error) {
	if generation != w.generation || target != w.group.target || identity != w.group.context {
		return
	}
	if identity != w.capture("hue.brightness").target {
		w.group = groupRequest{}
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
func (w *worker) recallScene(ctx context.Context, c command) {
	fail := func(detail string) {
		if c.complete != nil {
			c.complete(rejected(detail))
		}
	}
	sceneID := c.target.scene
	if !w.connected || sceneID == "" || !c.target.chosen {
		fail("Scene unavailable")
		return
	}
	if w.sceneAction != nil || w.sceneWriting != "" || w.sync.sceneAfterStop != "" {
		fail("Scene recall already in progress")
		return
	}
	w.sceneAction = &c
	delete(w.sceneErr, sceneID)
	delete(w.scenePending, sceneID)
	if w.syncing() {
		if err := w.syncSet(false); err != nil {
			w.finishScene(syncOutcome(err))
			return
		}
		w.sync.sceneAfterStop = sceneID
		w.sync.sceneStopAt = time.Now()
		return
	}
	w.sendRecall(ctx, sceneID)
}

func (w *worker) finishScene(outcome snoofer.Outcome) {
	c := w.sceneAction
	w.sceneAction = nil
	w.sceneWriting = ""
	w.sync.sceneAfterStop = ""
	if c != nil && c.complete != nil {
		c.complete(outcome)
	}
}

func (w *worker) sendRecall(ctx context.Context, sceneID string) {
	c := w.sceneAction
	if c == nil {
		return
	}
	target := w.capture(c.request.ID).target
	target.syncing = c.target.syncing // A confirmed Sync stop is expected preparation.
	if c.target.syncing {
		target.sync = w.sync.generation
	}
	if !w.connected || target != c.target {
		outcome := rejected("Scene target changed; scene not recalled")
		if c.target.syncing {
			outcome.Status = snoofer.Failed
		}
		w.finishScene(outcome)
		return
	}
	if ctx.Err() != nil {
		w.finishScene(operationOutcome(ctx.Err()))
		return
	}
	w.sceneWriting = sceneID
	client := w.client
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		err := client.Put(ctx, "scene", sceneID, map[string]any{"recall": map[string]string{"action": "active"}})
		return func(context.Context) {
			if w.sceneAction != c {
				return
			}
			current := w.capture(c.request.ID).target
			current.syncing = c.target.syncing
			if c.target.syncing {
				current.sync = w.sync.generation
			}
			if current != c.target {
				w.finishScene(operationOutcome(err))
				return
			}
			w.finishScene(operationOutcome(err))
			if err != nil {
				w.sceneErr[sceneID] = err.Error()
				return
			}
			if active, ok := w.model.resources[sceneID]; ok && active.Status != nil && active.Status.Active != "inactive" {
				return
			}
			w.scenePending[sceneID] = time.Now()
		}
	})
}

func (w *worker) startPairing(ctx context.Context, c command) {
	if w.pairing {
		return
	}
	w.pairing = true
	w.pairValue = "Press button"
	w.pairErr = ""
	address, deviceType, discover, window := c.target.endpoint, w.deviceType, w.discover, w.timing.pairWindow
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		target, key, err := pair(ctx, address, c.target.bridgeID, deviceType, discover, window)
		return func(ctx context.Context) { w.paired(ctx, c, target, key, err) }
	})
}

// pair resolves any single bridge and polls for a key while the link button window is open.
func pair(ctx context.Context, address, bridgeID, deviceType string, discover discoverFunc, window time.Duration) (bridgeTarget, string, error) {
	ctx, cancel := context.WithTimeout(ctx, window)
	defer cancel()
	target, err := resolve(ctx, address, bridgeID, discover)
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
			return target, "", fmt.Errorf("pairing timed out after %s: %w", window, ctx.Err())
		}
		if !errors.Is(err, ErrLinkButton) {
			return target, "", err
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return target, "", fmt.Errorf("pairing timed out after %s: %w", window, ctx.Err())
			}
			return target, "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *worker) paired(ctx context.Context, c command, target bridgeTarget, key string, err error) {
	finish := func(o snoofer.Outcome) {
		if c.complete != nil {
			c.complete(o)
		}
	}
	if ctx.Err() != nil {
		finish(operationOutcome(ctx.Err()))
		return
	}
	if c.target.settings != w.settingsEpoch {
		w.pairing = false
		finish(snoofer.Outcome{Status: snoofer.Failed, Detail: "Hue settings changed; pairing was not saved"})
		return
	}
	w.pairing = false
	w.pairValue = ""
	if err != nil {
		w.pairErr = err.Error()
		finish(operationOutcome(err))
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
		finish(operationOutcome(err))
		return
	}
	finish(processed("Paired and saved"))
	w.reconnect(ctx)
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
	for _, g := range w.chosenGroups() {
		options = append(options, g.ID)
		labels[g.ID] = g.Name
		if g.Kind == "zone" {
			labels[g.ID] = g.Name + " (zone)"
		}
	}
	controls := []snoofer.Control{
		{ID: "hue.status", Label: "Hue", Group: "Hue", Kind: "status", Value: w.status, Status: statusNote, ViewData: w.statusView(), Available: true},
		{ID: "hue.pair", Label: "Pair Hue Bridge", ShortLabel: "Pair", Group: "Hue", Kind: "command", Icon: "hue-pair", Value: pairValue, Status: w.pairErr, Operations: []string{"press"}, Available: live},
		{ID: "hue.group", Label: "Hue Room", ShortLabel: "Room", Group: "Hue", Kind: "selection", Value: w.settings.Group, Options: options, OptionLabels: labels, Status: w.groupErr, Operations: []string{"set"}, Available: live && w.connected,
			Hidden: w.connected && len(options) < 2}, // Nothing to switch between on the deck.
		w.roomsControl(),
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
	brightness := snoofer.Control{ID: "hue.brightness", Label: "Hue Brightness", ShortLabel: "Brightness", Group: "Hue", Kind: "numeric", Icon: "hue-brightness",
		Value: w.brightnessValue(view), Status: cmp.Or(knobNote, w.brightErr), Subdued: w.group.on != nil || w.group.brightness != nil,
		Operations: []string{"adjust", "press"}, Available: ready}
	if syncing {
		// While syncing the dial adjusts the stream; a press still toggles the room.
		brightness.Value = w.syncBrightnessLabel()
		brightness.Status = ""
		brightness.Subdued = w.sync.stepsDue != 0
		brightness.Available = live
	}
	sceneSort := snoofer.Control{ID: "hue.sort", Label: "Hue Scene Sort", ShortLabel: "Sort", Group: "Hue", Kind: "selection", Icon: "hue-sort",
		Value: w.sceneSort(), Options: sceneSorts, OptionLabels: sceneSortLabels, Status: w.sortErr, Operations: []string{"set"}, Available: live}
	controls = append(controls, brightness, w.motionControl(), sceneSort)
	controls = append(controls, w.groupDialControls()...)
	controls = append(controls, w.syncControls()...)
	now := time.Now()
	controls = append(controls, w.bridgeReport(now), w.syncReport(now))
	seen := map[string]bool{}
	order := w.sceneOrder()
	for _, scene := range w.model.scenes() {
		if seen[scene.ControlID] {
			continue // A colliding ID prefix would invalidate the whole snapshot.
		}
		if !w.settings.chosen(scene.GroupID) {
			continue
		}
		seen[scene.ControlID] = true
		control := w.sceneControl(scene.ControlID, scene.Label, scene)
		control.Collection = "hue.scenes." + scene.GroupID
		control.CollectionLabel = "Scenes · " + scene.GroupName
		control.Order = order[scene.SceneID]
		controls = append(controls, control)
	}
	return append(controls, w.slotControls()...)
}

// sceneOrder is each scene's position within its room in the chosen sort.
func (w *worker) sceneOrder() map[string]int {
	rooms := map[string][]sceneInfo{}
	for _, scene := range w.model.scenes() {
		rooms[scene.GroupID] = append(rooms[scene.GroupID], scene)
	}
	order := map[string]int{}
	for _, scenes := range rooms {
		w.model.sortScenes(scenes, w.sceneSort())
		for n, scene := range scenes {
			order[scene.SceneID] = n + 1
		}
	}
	return order
}

// roomSlots is the minimum number of stable slots mirroring the selected
// room's scenes; rooms with more scenes publish one slot per scene.
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
	c := snoofer.Control{ID: id, Label: label, ShortLabel: scene.ShortLabel, Group: "Hue Scenes", Kind: "command", Icon: "hue-scene", Artwork: w.sceneArtwork(scene.SceneID),
		Value: value, Status: status, Operations: []string{"press"}, Available: w.services.Live && w.connected}
	if b, ok := sceneBrightness(w.model.resources[scene.SceneID]); ok {
		c.Brightness = &b
	}
	return c
}

// slotControls maps hue.room-scene-N to the selected room's scenes. The
// scene ID in ViewData changes the slot's revision whenever its mapping
// changes, so input generated for a previous scene is rejected as stale.
// Unused slots are Hidden with no label or icon: blank keys that regions skip.
func (w *worker) slotControls() []snoofer.Control {
	var scenes []sceneInfo
	if w.connected && w.settings.Group != "" {
		scenes = w.model.roomScenes(w.settings.Group, w.sceneSort())
	}
	slots := max(roomSlots, len(scenes))
	controls := make([]snoofer.Control, 0, slots)
	for n := 1; n <= slots; n++ {
		id := "hue.room-scene-" + strconv.Itoa(n)
		if n > len(scenes) {
			controls = append(controls, snoofer.Control{ID: id, Group: "Hue Room Scenes", Kind: "command", Operations: []string{"press"}, Hidden: true,
				Collection: "hue.room-scenes", CollectionLabel: "Scenes · Selected Room", Order: n})
			continue
		}
		scene := scenes[n-1]
		control := w.sceneControl(id, scene.ShortLabel, scene)
		control.Group = "Hue Room Scenes"
		control.Collection = "hue.room-scenes"
		control.CollectionLabel = "Scenes · Selected Room"
		control.Order = n
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
	scenes := w.model.roomScenes(w.settings.Group, w.sceneSort())
	if n < 1 || n > len(scenes) {
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
