package hue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"sound-snoofer/snoofer"
)

// The sync half controls the Hue Sync PC app through its loopback
// third-party control WebSocket (Hue Sync Settings > Third-party control).
//
// Protocol, as documented in Elgato's Hue Sync plugin: clients send
// {"command":C,"data":{...}} with start_sync, stop_sync, inc_bri {step},
// set_intensity {intensity} and set_app_mode {mode}; the app pushes
// {"event":"app_state_update","data":{"state","mode","intensity","bri"}}.
// Mode and intensity changes only take effect while syncing.

const (
	defaultSyncPort       = 24851
	syncStateSyncing      = "syncing"
	syncStateDisconnected = "bridge_disconnected"
	syncBrightnessStep    = 2 // inc_bri step per dial tick on the app's 0-100 scale.
)

var (
	syncModes            = []string{"video", "games", "music"}
	syncModeLabels       = map[string]string{"video": "Video", "games": "Games", "music": "Music"}
	syncIntensities      = []string{"subtle", "moderate", "high", "extreme"}
	syncIntensityLabels  = map[string]string{"subtle": "Subtle", "moderate": "Moderate", "high": "High", "extreme": "Extreme"}
	errSyncNotConfirmed  = "Not confirmed by Hue Sync"
	errSceneSyncNotEnded = "Hue Sync did not stop; scene not recalled"
)

// syncState is the last app_state_update. Bri is a pointer so a missing value
// stays distinct from zero brightness.
type syncState struct {
	State     string   `json:"state"`
	Mode      string   `json:"mode"`
	Intensity string   `json:"intensity"`
	Bri       *float64 `json:"bri"`
}

// syncLink is the worker-owned Hue Sync connection state. Only the worker
// writes to conn; one reader goroutine per connection delivers events.
type syncLink struct {
	url        string
	generation int
	connecting bool
	conn       *websocket.Conn
	retryAt    time.Time
	retryDelay time.Duration
	diagnostic string

	known bool // An app_state_update arrived on the current connection.
	state syncState

	wanted   *bool // Requested sync state awaiting confirmation.
	sent     time.Time
	err      string
	stepsDue int // Accumulated inc_bri steps not yet sent.
	stepSent time.Time

	sceneAfterStop string // Scene ID to recall once the app confirms sync stopped.

	link        snoofer.ConnectionTracker // Connection report timing.
	connects    int                       // Successful connections this session.
	sceneStopAt time.Time
}

func syncURL(settings Settings) string {
	port := defaultSyncPort
	if settings.SyncPort != nil {
		port = *settings.SyncPort
	}
	return "ws://127.0.0.1:" + strconv.Itoa(port) + "/"
}

func (w *worker) syncing() bool {
	return w.sync.conn != nil && w.sync.known && w.sync.state.State == syncStateSyncing
}

// syncReady reports a connection to an app that has a bridge.
func (w *worker) syncReady() bool {
	return w.sync.conn != nil && w.sync.known && w.sync.state.State != syncStateDisconnected
}

func (w *worker) syncConnect(ctx context.Context) {
	if w.sync.connecting || w.sync.conn != nil {
		return
	}
	w.sync.connecting = true
	w.sync.generation++
	generation := w.sync.generation
	url, timeout := w.sync.url, w.timing.syncDial
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		dialer := websocket.Dialer{HandshakeTimeout: timeout}
		dialCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		conn, response, err := dialer.DialContext(dialCtx, url, nil)
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return func(ctx context.Context) { w.syncConnected(ctx, generation, conn, err) }
	})
}

func (w *worker) syncConnected(ctx context.Context, generation int, conn *websocket.Conn, err error) {
	if generation != w.sync.generation {
		if conn != nil {
			conn.Close()
		}
		return
	}
	w.sync.connecting = false
	if err != nil {
		w.syncLost(fmt.Errorf("Hue Sync not reachable at %s; open Hue Sync and turn on Settings > Third-party control: %w", w.sync.url, err))
		return
	}
	w.sync.conn = conn
	w.sync.connects++
	w.sync.diagnostic = ""
	w.syncRead(ctx, generation, conn)
}

// syncRead delivers app_state_update events until the connection closes.
func (w *worker) syncRead(ctx context.Context, generation int, conn *websocket.Conn) {
	conn.SetReadLimit(1 << 16)
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return func(context.Context) {
					if generation == w.sync.generation {
						w.syncLost(fmt.Errorf("Hue Sync connection closed: %w", err))
					}
				}
			}
			var message struct {
				Event string    `json:"event"`
				Data  syncState `json:"data"`
			}
			if json.Unmarshal(data, &message) != nil || message.Event != "app_state_update" {
				continue // Unknown events and malformed messages are ignored.
			}
			apply := func(ctx context.Context) {
				if generation == w.sync.generation {
					w.sync.retryDelay = 0 // Reset only once the connection delivers.
					w.syncObserve(ctx, message.Data)
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

func (w *worker) syncClose() {
	if w.sync.conn != nil {
		w.sync.conn.Close()
		w.sync.conn = nil
	}
}

// syncLost forgets observed state and pending commands; nothing is replayed.
func (w *worker) syncLost(err error) {
	w.sync.generation++
	w.syncClose()
	w.sync.connecting = false
	w.sync.known = false
	w.sync.state = syncState{}
	w.sync.wanted = nil
	w.sync.stepsDue = 0
	if w.sync.sceneAfterStop != "" {
		w.sceneErr[w.sync.sceneAfterStop] = errSceneSyncNotEnded
		w.finishScene(snoofer.Outcome{Status: snoofer.Unknown, Detail: errSceneSyncNotEnded})
	}
	w.sync.diagnostic = err.Error()
	w.sync.link.Fail(err.Error(), time.Now())
	w.sync.retryDelay = min(w.timing.retryMax, max(w.timing.retryBase, 2*w.sync.retryDelay))
	w.sync.retryAt = time.Now().Add(w.sync.retryDelay)
}

func (w *worker) syncObserve(ctx context.Context, update syncState) {
	w.sync.link.Activity(time.Now())
	w.sync.known = true
	w.sync.state.State = update.State
	if update.Mode != "" {
		w.sync.state.Mode = update.Mode
	}
	if update.Intensity != "" {
		w.sync.state.Intensity = update.Intensity
	}
	if update.Bri != nil {
		w.sync.state.Bri = update.Bri
	}
	if w.sync.wanted != nil && w.syncing() == *w.sync.wanted {
		w.sync.wanted = nil
	}
	if w.sync.sceneAfterStop != "" && !w.syncing() {
		scene := w.sync.sceneAfterStop
		if !w.syncReady() {
			w.finishScene(snoofer.Outcome{Status: snoofer.Failed, Detail: errSceneSyncNotEnded})
		} else {
			w.sync.sceneAfterStop = ""
			w.sendRecall(ctx, scene)
		}
	}
}

// syncTick reconnects, flushes accumulated brightness steps and expires confirmations.
func (w *worker) syncTick(ctx context.Context, now time.Time) bool {
	changed := false
	if w.sync.conn == nil && !w.sync.connecting && !w.sync.retryAt.IsZero() && now.After(w.sync.retryAt) {
		w.sync.retryAt = time.Time{}
		w.syncConnect(ctx)
		changed = true
	}
	if w.sync.stepsDue != 0 && now.Sub(w.sync.stepSent) >= w.timing.syncBrightnessGap {
		step := w.sync.stepsDue
		w.sync.stepsDue = 0
		w.sync.stepSent = now
		if w.syncing() {
			w.syncSend("inc_bri", map[string]any{"step": step})
		}
		changed = true
	}
	if w.sync.wanted != nil && now.Sub(w.sync.sent) > w.timing.confirm {
		w.sync.wanted = nil
		w.sync.err = errSyncNotConfirmed
		changed = true
	}
	if w.sync.sceneAfterStop != "" && now.Sub(w.sync.sceneStopAt) > w.timing.confirm {
		w.sceneErr[w.sync.sceneAfterStop] = errSceneSyncNotEnded
		w.finishScene(snoofer.Outcome{Status: snoofer.Unknown, Detail: errSceneSyncNotEnded})
		changed = true
	}
	return changed
}

var errSyncBusy = errors.New("Hue Sync request already in progress")
var errSyncUnavailable = errors.New("Hue Sync unavailable")

func syncOutcome(err error) snoofer.Outcome {
	if errors.Is(err, errSyncBusy) || errors.Is(err, errSyncUnavailable) {
		return rejected(err.Error())
	}
	if err != nil {
		return snoofer.Outcome{Status: snoofer.Unknown, Detail: err.Error()}
	}
	return submitted("Hue Sync command sent")
}

func (w *worker) handleSync(r snoofer.Request, want bool) snoofer.Outcome {
	if !w.syncReady() {
		return rejected(errSyncUnavailable.Error())
	}
	switch r.ID {
	case "hue.sync":
		return syncOutcome(w.syncSet(want))
	case "hue.sync-mode":
		if !w.syncing() || !slices.Contains(syncModes, r.Value) {
			return rejected("Hue Sync mode unavailable")
		}
		return syncOutcome(w.syncSend("set_app_mode", map[string]any{"mode": r.Value}))
	case "hue.sync-intensity":
		if !w.syncing() || !slices.Contains(syncIntensities, r.Value) {
			return rejected("Hue Sync intensity unavailable")
		}
		return syncOutcome(w.syncSend("set_intensity", map[string]any{"intensity": r.Value}))
	}
	return rejected("Hue Sync control unavailable")
}

func (w *worker) syncSet(want bool) error {
	if w.sync.wanted != nil {
		return errSyncBusy
	}
	if !w.syncReady() {
		return errSyncUnavailable
	}
	command := "stop_sync"
	if want {
		command = "start_sync"
	}
	w.sync.err = ""
	if err := w.syncSend(command, nil); err != nil {
		return err
	}
	w.sync.wanted = &want
	w.sync.sent = time.Now()
	return nil
}

// syncSend writes one command. A write failure drops the connection without retrying.
func (w *worker) syncSend(command string, data map[string]any) error {
	if w.sync.conn == nil {
		return errSyncUnavailable
	}
	if data == nil {
		data = map[string]any{}
	}
	payload, err := json.Marshal(map[string]any{"command": command, "data": data})
	if err != nil {
		w.sync.err = err.Error()
		return err
	}
	if err := w.sync.conn.SetWriteDeadline(time.Now().Add(w.timing.syncWrite)); err != nil {
		w.syncLost(err)
		return err
	}
	if err := w.sync.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		w.syncLost(fmt.Errorf("Hue Sync write failed: %w", err))
		return err
	}
	w.sync.link.Activity(time.Now())
	return nil
}

func (w *worker) syncBrightnessLabel() string {
	if w.sync.state.Bri == nil {
		return "Sync"
	}
	return fmt.Sprintf("Sync %.0f%%", math.Round(*w.sync.state.Bri))
}

// syncControls publishes the sync half. Sync-only controls are Hidden while
// they cannot apply: Mode and Intensity unless syncing, Sync while the app
// is not connected.
func (w *worker) syncControls() []snoofer.Control {
	live, ready, syncing := w.services.Live, w.syncReady(), w.syncing()
	connected := w.sync.conn != nil && w.sync.known
	status := "N/A"
	switch {
	case !connected:
	case w.sync.state.State == syncStateDisconnected:
		status = "No bridge"
	case syncing:
		status = "Syncing"
	default:
		status = "Ready"
	}
	syncValue := "N/A"
	if connected {
		syncValue = "Off"
		if syncing {
			syncValue = "On"
		}
	}
	syncStatus := w.sync.err
	if w.sync.wanted != nil {
		syncStatus = "Pending"
	}
	return []snoofer.Control{
		{ID: "hue.sync-status", Label: "Hue Sync", Group: "Hue", Kind: "status", Value: status, Status: w.sync.diagnostic, Available: true},
		{ID: "hue.sync", Label: "Hue Sync", ShortLabel: "Sync", Group: "Hue", Kind: "toggle", Icon: "huesync-sync", Value: syncValue, Status: syncStatus,
			Operations: []string{"press"}, Available: live && ready, Hidden: !connected},
		{ID: "hue.sync-mode", Label: "Hue Sync Mode", ShortLabel: "Mode", Group: "Hue", Kind: "selection", Icon: "huesync-mode", Value: w.sync.state.Mode,
			Options: syncModes, OptionLabels: syncModeLabels, Operations: []string{"set"}, Available: live && ready && syncing, Hidden: !syncing},
		{ID: "hue.sync-intensity", Label: "Hue Sync Intensity", ShortLabel: "Intensity", Group: "Hue", Kind: "selection", Icon: "huesync-intensity", Value: w.sync.state.Intensity,
			Options: syncIntensities, OptionLabels: syncIntensityLabels, Operations: []string{"set"}, Available: live && ready && syncing, Hidden: !syncing},
	}
}
