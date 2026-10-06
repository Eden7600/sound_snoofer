// Package huesync controls the Hue Sync PC app through its loopback
// third-party control WebSocket (Hue Sync Settings > Third-party control).
//
// Protocol, as documented in Elgato's Hue Sync plugin: clients send
// {"command":C,"data":{...}} with start_sync, stop_sync, inc_bri {step},
// set_intensity {intensity} and set_app_mode {mode}; the app pushes
// {"event":"app_state_update","data":{"state","mode","intensity","bri"}}.
// Mode and intensity changes only take effect while syncing.
package huesync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"sound-snoofer/snoofer"
)

// Settings configures the Hue Sync connection.
type Settings struct {
	Port int `json:"port"`
}

const (
	stateSyncing      = "syncing"
	stateConnected    = "bridge_connected"
	stateDisconnected = "bridge_disconnected"
	brightnessStep    = 2 // inc_bri step per dial tick on the app's 0-100 scale.
)

var (
	modes       = []string{"video", "games", "music"}
	intensities = []string{"subtle", "moderate", "high", "extreme"}
)

// Plugin returns inert metadata; no connection is attempted until Start.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{
		ID:       "huesync",
		Label:    "Hue Sync",
		Validate: validate,
		Defaults: snoofer.MarshalSettings(Settings{Port: 24851}),
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
			return start(ctx, s, raw, defaultTiming)
		},
	}
}

func decode(raw json.RawMessage) (Settings, error) {
	var s Settings
	if err := snoofer.DecodeSettings(raw, &s); err != nil {
		return s, err
	}
	if s.Port < 1 || s.Port > 65535 {
		return s, fmt.Errorf("huesync port must be 1-65535")
	}
	return s, nil
}

func validate(raw json.RawMessage) error {
	_, err := decode(raw)
	return err
}

// timing bounds reconnection, brightness coalescing and confirmation. Tests shorten it.
type timing struct {
	dial          time.Duration
	write         time.Duration
	retryBase     time.Duration
	retryMax      time.Duration
	brightnessGap time.Duration
	confirm       time.Duration
}

var defaultTiming = timing{
	dial:          3 * time.Second,
	write:         time.Second,
	retryBase:     time.Second,
	retryMax:      30 * time.Second,
	brightnessGap: 100 * time.Millisecond,
	confirm:       3 * time.Second,
}

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Stop cancels the worker and waits for its connection to close.
func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, t timing) (snoofer.Instance, error) {
	settings, err := decode(raw)
	if err != nil {
		return nil, err
	}
	w := &worker{
		services: s,
		url:      "ws://127.0.0.1:" + strconv.Itoa(settings.Port) + "/",
		timing:   t,
		requests: make(chan snoofer.Request, 16),
		results:  make(chan func(context.Context), 16),
	}
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(i.done)
		w.run(runCtx)
	}()
	return i, nil
}

// appState is the last app_state_update. Bri is a pointer so a missing value
// stays distinct from zero brightness.
type appState struct {
	State     string   `json:"state"`
	Mode      string   `json:"mode"`
	Intensity string   `json:"intensity"`
	Bri       *float64 `json:"bri"`
}

// worker owns the connection. Only the worker writes; one reader goroutine
// per connection returns messages as functions executed on the worker.
type worker struct {
	services snoofer.Services
	url      string
	timing   timing

	requests chan snoofer.Request
	results  chan func(context.Context)
	children sync.WaitGroup

	generation int
	connecting bool
	conn       *websocket.Conn
	retryAt    time.Time
	retryDelay time.Duration
	diagnostic string

	known bool // An app_state_update arrived on the current connection.
	state appState

	syncWanted *bool // Requested sync state awaiting confirmation.
	syncSent   time.Time
	syncErr    string
	stepsDue   int // Accumulated inc_bri steps not yet sent.
	stepSent   time.Time
	writeErr   string
}

func (w *worker) run(ctx context.Context) {
	defer w.services.Controls.Remove("huesync")
	defer w.children.Wait()
	defer w.closeConn() // Runs first, unblocking the reader before Wait.
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	w.connect(ctx)
	w.publish()
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-w.requests:
			w.handle(r)
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
		return errors.New("hue sync queue full")
	}
}

func (w *worker) connect(ctx context.Context) {
	if w.connecting || w.conn != nil {
		return
	}
	w.connecting = true
	w.generation++
	generation := w.generation
	url, timeout := w.url, w.timing.dial
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		dialer := websocket.Dialer{HandshakeTimeout: timeout}
		dialCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		conn, response, err := dialer.DialContext(dialCtx, url, nil)
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return func(ctx context.Context) { w.onConnected(ctx, generation, conn, err) }
	})
}

func (w *worker) onConnected(ctx context.Context, generation int, conn *websocket.Conn, err error) {
	if generation != w.generation {
		if conn != nil {
			conn.Close()
		}
		return
	}
	w.connecting = false
	if err != nil {
		w.lost(fmt.Errorf("Hue Sync not reachable at %s; start Hue Sync and enable Settings > Third-party control: %w", w.url, err))
		return
	}
	w.conn = conn
	w.diagnostic = ""
	w.retryDelay = 0
	w.read(ctx, generation, conn)
}

// read delivers app_state_update events until the connection closes.
func (w *worker) read(ctx context.Context, generation int, conn *websocket.Conn) {
	conn.SetReadLimit(1 << 16)
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return func(context.Context) {
					if generation == w.generation {
						w.lost(fmt.Errorf("Hue Sync connection closed: %w", err))
					}
				}
			}
			var message struct {
				Event string   `json:"event"`
				Data  appState `json:"data"`
			}
			if json.Unmarshal(data, &message) != nil || message.Event != "app_state_update" {
				continue // Unknown events and malformed messages are ignored.
			}
			apply := func(context.Context) {
				if generation == w.generation {
					w.observe(message.Data)
				}
			}
			select {
			case w.results <- apply:
			case <-ctx.Done():
				return func(context.Context) {}
			}
		}
	})
}

func (w *worker) closeConn() {
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
}

// lost forgets observed state and pending commands; nothing is replayed.
func (w *worker) lost(err error) {
	w.generation++
	w.closeConn()
	w.connecting = false
	w.known = false
	w.state = appState{}
	w.syncWanted = nil
	w.stepsDue = 0
	w.diagnostic = err.Error()
	w.retryDelay = min(w.timing.retryMax, max(w.timing.retryBase, 2*w.retryDelay))
	w.retryAt = time.Now().Add(w.retryDelay)
}

func (w *worker) observe(update appState) {
	w.known = true
	w.state.State = update.State
	if update.Mode != "" {
		w.state.Mode = update.Mode
	}
	if update.Intensity != "" {
		w.state.Intensity = update.Intensity
	}
	if update.Bri != nil {
		w.state.Bri = update.Bri
	}
	if w.syncWanted != nil && w.syncing() == *w.syncWanted {
		w.syncWanted = nil
	}
}

func (w *worker) syncing() bool { return w.state.State == stateSyncing }

// ready reports a connection to an app that has a bridge.
func (w *worker) ready() bool {
	return w.conn != nil && w.known && w.state.State != stateDisconnected
}

func (w *worker) tick(ctx context.Context, now time.Time) bool {
	changed := false
	if w.conn == nil && !w.connecting && !w.retryAt.IsZero() && now.After(w.retryAt) {
		w.retryAt = time.Time{}
		w.connect(ctx)
		changed = true
	}
	if w.stepsDue != 0 && now.Sub(w.stepSent) >= w.timing.brightnessGap {
		step := w.stepsDue
		w.stepsDue = 0
		w.stepSent = now
		w.send("inc_bri", map[string]any{"step": step})
		changed = true
	}
	if w.syncWanted != nil && now.Sub(w.syncSent) > w.timing.confirm {
		w.syncWanted = nil
		w.syncErr = "Not confirmed by Hue Sync"
		changed = true
	}
	return changed
}

func (w *worker) handle(r snoofer.Request) {
	if !w.services.Live || !w.ready() {
		return
	}
	switch {
	case r.ID == "huesync.sync", r.ID == "huesync.brightness" && r.Operation == "press":
		w.toggleSync()
	case r.ID == "huesync.brightness" && r.Operation == "adjust":
		w.stepsDue += r.Delta * brightnessStep
	case r.ID == "huesync.mode" && w.syncing():
		w.send("set_app_mode", map[string]any{"mode": r.Value})
	case r.ID == "huesync.intensity" && w.syncing():
		w.send("set_intensity", map[string]any{"intensity": r.Value})
	}
}

func (w *worker) toggleSync() {
	if w.syncWanted != nil {
		return // Awaiting the previous command's outcome.
	}
	want := !w.syncing()
	command := "stop_sync"
	if want {
		command = "start_sync" // Without data the app keeps its current mode and intensity.
	}
	w.syncErr = ""
	if !w.send(command, nil) {
		return
	}
	w.syncWanted = &want
	w.syncSent = time.Now()
}

// send writes one command. A write failure drops the connection without retrying.
func (w *worker) send(command string, data map[string]any) bool {
	if data == nil {
		data = map[string]any{}
	}
	payload, err := json.Marshal(map[string]any{"command": command, "data": data})
	if err != nil {
		w.writeErr = err.Error()
		return false
	}
	if err := w.conn.SetWriteDeadline(time.Now().Add(w.timing.write)); err != nil {
		w.lost(err)
		return false
	}
	if err := w.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		w.lost(fmt.Errorf("Hue Sync write failed: %w", err))
		return false
	}
	w.writeErr = ""
	return true
}

func (w *worker) publish() {
	// Publish fails only while the host is stopping, which also cancels this worker.
	_ = w.services.Controls.Publish("huesync", w.controls(), w.enqueue)
}

func (w *worker) controls() []snoofer.Control {
	live, ready, syncing := w.services.Live, w.ready(), w.syncing()
	status := "N/A"
	switch {
	case !w.known || w.conn == nil:
	case w.state.State == stateDisconnected:
		status = "No bridge"
	case syncing:
		status = "Syncing"
	default:
		status = "Ready"
	}
	syncValue, brightness := "N/A", "N/A"
	if w.known && w.conn != nil {
		syncValue = "Off"
		if syncing {
			syncValue = "On"
		}
		if w.state.Bri != nil {
			brightness = fmt.Sprintf("%.0f%%", math.Round(*w.state.Bri))
		}
	}
	syncStatus := w.syncErr
	if w.syncWanted != nil {
		syncStatus = "Pending"
	}
	modeLabels := map[string]string{"video": "Video", "games": "Games", "music": "Music"}
	intensityLabels := map[string]string{"subtle": "Subtle", "moderate": "Moderate", "high": "High", "extreme": "Extreme"}
	return []snoofer.Control{
		{ID: "huesync.status", Label: "Hue Sync", Group: "Hue Sync", Kind: "status", Value: status, Status: firstNonEmpty(w.diagnostic, w.writeErr), Available: true},
		{ID: "huesync.sync", Label: "Hue Sync", ShortLabel: "Sync", Group: "Hue Sync", Kind: "toggle", Icon: "huesync-sync", Value: syncValue, Status: syncStatus,
			Operations: []string{"press"}, Available: live && ready},
		{ID: "huesync.brightness", Label: "Hue Sync brightness", ShortLabel: "Sync bright", Group: "Hue Sync", Kind: "numeric", Icon: "hue-brightness", Value: brightness,
			Subdued: w.stepsDue != 0, Operations: []string{"adjust", "press"}, Available: live && ready},
		{ID: "huesync.mode", Label: "Hue Sync mode", ShortLabel: "Mode", Group: "Hue Sync", Kind: "selection", Icon: "huesync-mode", Value: w.state.Mode,
			Options: modes, OptionLabels: modeLabels, Operations: []string{"set"}, Available: live && ready && syncing},
		{ID: "huesync.intensity", Label: "Hue Sync intensity", ShortLabel: "Intensity", Group: "Hue Sync", Kind: "selection", Icon: "huesync-intensity", Value: w.state.Intensity,
			Options: intensities, OptionLabels: intensityLabels, Operations: []string{"set"}, Available: live && ready && syncing},
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
