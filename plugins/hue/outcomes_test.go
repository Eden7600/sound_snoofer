package hue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func outcomeWorker(t *testing.T, items []resource) (*worker, *fakeBridge, context.Context) {
	t.Helper()
	b := newFakeBridge(t, "b1", items...)
	settings := paired(b, "room-1")
	raw, _ := json.Marshal(settings)
	ctx, cancel := context.WithCancel(context.Background())
	w := newWorker(snoofer.Services{Controls: snoofer.NewControls(), Live: true, SaveSettings: func(string, json.RawMessage, json.RawMessage) error { return nil }}, settings, raw, nil, testTiming)
	w.connected, w.generation, w.client, w.model = true, 1, newClient(b.address(), b.fingerprint(), b.key), newModel(items)
	t.Cleanup(func() { cancel(); w.syncClose(); w.closeClient(); w.children.Wait() })
	return w, b, ctx
}

func queueOutcome(t *testing.T, w *worker, ctx context.Context, id, op, value string, delta int) <-chan snoofer.Outcome {
	t.Helper()
	w.publish(ctx)
	results := make(chan snoofer.Outcome, 4)
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID == id {
			if err := w.services.Controls.DispatchWithResult(ctx, snoofer.Request{ID: id, Revision: c.Revision, Operation: op, Value: value, Delta: delta}, func(o snoofer.Outcome) { results <- o }); err != nil {
				t.Fatal(err)
			}
			return results
		}
	}
	t.Fatal("missing", id)
	return results
}

func TestHueSettingsOutcomeWaitsForSave(t *testing.T) {
	w, _, ctx := outcomeWorker(t, studio())
	results := queueOutcome(t, w, ctx, "hue.group", "set", "zone-1", 0)
	select {
	case o := <-results:
		t.Fatalf("enqueue completed settings save: %+v", o)
	default:
	}
	w.handle(ctx, <-w.requests)
	select {
	case o := <-results:
		if o.Status != snoofer.Processed {
			t.Fatal(o)
		}
	case <-time.After(time.Second):
		t.Fatal("save outcome missing")
	}
}

func TestHueQueuedBrightnessCannotChangeRooms(t *testing.T) {
	w, _, ctx := outcomeWorker(t, studio())
	queueOutcome(t, w, ctx, "hue.brightness", "adjust", "", 1)
	w.selectGroup("zone-1")
	w.handle(ctx, <-w.requests)
	if w.group.dirty {
		t.Fatal("old brightness input targeted replacement room", w.group.target)
	}
}

func awaitHueOutcome(t *testing.T, results <-chan snoofer.Outcome, status snoofer.OutcomeStatus) snoofer.Outcome {
	t.Helper()
	select {
	case o := <-results:
		if o.Status != status {
			t.Fatalf("outcome %+v, want %s", o, status)
		}
		return o
	case <-time.After(time.Second):
		t.Fatal("outcome missing")
		return snoofer.Outcome{}
	}
}
func applyHueResult(t *testing.T, w *worker, ctx context.Context) {
	t.Helper()
	select {
	case apply := <-w.results:
		apply(ctx)
	case <-time.After(time.Second):
		t.Fatal("network callback missing")
	}
}
func TestHueSaveFailuresAndRequiredRoomSelection(t *testing.T) {
	for _, id := range []string{"hue.group", "hue.rooms", "hue.sort"} {
		t.Run(id, func(t *testing.T) {
			w, _, ctx := outcomeWorker(t, studio())
			calls := 0
			w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error {
				calls++
				if id == "hue.rooms" && calls == 1 {
					return nil
				}
				return errors.New("save refused")
			}
			value := "zone-1"
			if id == "hue.sort" {
				value = "hue"
			}
			out := queueOutcome(t, w, ctx, id, "set", value, 0)
			w.handle(ctx, <-w.requests)
			if got := awaitHueOutcome(t, out, snoofer.Failed); !strings.Contains(got.Detail, "save refused") {
				t.Fatal(got)
			}
			if id == "hue.rooms" && calls != 2 {
				t.Fatalf("required room save count %d", calls)
			}
		})
	}
}

func TestHuePairOutcomeWaitsForCredentialsSave(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			w, b, ctx := outcomeWorker(t, studio())
			b.set(func(b *fakeBridge) { b.linkPressed = true })
			saved := false
			w.services.SaveSettings = func(_ string, _ json.RawMessage, next json.RawMessage) error {
				var got Settings
				if err := json.Unmarshal(next, &got); err != nil {
					t.Fatal(err)
				}
				if got.AppKey != b.key || got.BridgeID != b.id || got.CertificateSHA256 != b.fingerprint() {
					t.Fatalf("credentials incomplete: %+v", got)
				}
				saved = true
				if fail {
					return errors.New("credentials save refused")
				}
				return nil
			}
			out := queueOutcome(t, w, ctx, "hue.pair", "press", "", 0)
			w.handle(ctx, <-w.requests)
			select {
			case got := <-out:
				t.Fatalf("completed before save: %+v", got)
			default:
			}
			applyHueResult(t, w, ctx)
			status := snoofer.Processed
			if fail {
				status = snoofer.Failed
			}
			awaitHueOutcome(t, out, status)
			if !saved {
				t.Fatal("credentials not saved")
			}
		})
	}
}

func TestHueMotionJoinsAllWritesAndReportsPartialFailure(t *testing.T) {
	w, b, ctx := outcomeWorker(t, sensorStudio())
	b.set(func(b *fakeBridge) { b.throttle = 1; b.ignorePuts = true })
	out := queueOutcome(t, w, ctx, "hue.motion", "press", "", 0)
	w.handle(ctx, <-w.requests)
	applyHueResult(t, w, ctx)
	select {
	case got := <-out:
		t.Fatalf("completed before all sensors: %+v", got)
	default:
	}
	applyHueResult(t, w, ctx)
	awaitHueOutcome(t, out, snoofer.Failed)
	if puts := motionPuts(b); len(puts) != 2 {
		t.Fatal(puts)
	}
	if motionValue(w.roomSensors()) != "On" {
		t.Fatal("request fabricated observed state")
	}
}

func TestHueMotionCapturedDesiredValueUsesCurrentSkipState(t *testing.T) {
	w, b, ctx := outcomeWorker(t, sensorStudio())
	off := false
	w.model.apply([]event{{Type: "update", Data: []resource{{ID: "motion-1", Enabled: &off}}}})
	out := queueOutcome(t, w, ctx, "hue.motion", "press", "", 0)
	w.model.apply([]event{{Type: "update", Data: []resource{{ID: "motion-2", Enabled: &off}}}})
	w.handle(ctx, <-w.requests)
	applyHueResult(t, w, ctx)
	applyHueResult(t, w, ctx)
	awaitHueOutcome(t, out, snoofer.Processed)
	puts := motionPuts(b)
	if len(puts) != 2 {
		t.Fatalf("captured desired On omitted a sensor: %v", puts)
	}
	for _, put := range puts {
		if !strings.Contains(put, `"enabled":true`) {
			t.Fatal(put)
		}
	}
}

func TestHueBrightnessToggleDoesNotRecomputeDesiredValue(t *testing.T) {
	w, _, ctx := outcomeWorker(t, studio())
	out := queueOutcome(t, w, ctx, "hue.brightness", "press", "", 0)
	off := &onState{On: false}
	w.model.apply([]event{{Type: "update", Data: []resource{{ID: "gl-1", On: off}}}})
	w.handle(ctx, <-w.requests)
	awaitHueOutcome(t, out, snoofer.Submitted)
	if w.group.on == nil || *w.group.on {
		t.Fatal("captured turn-off became turn-on")
	}
}

func TestHueSceneWaitsForHTTPAndDoesNotRetry(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			w, b, ctx := outcomeWorker(t, studio())
			if fail {
				b.set(func(b *fakeBridge) { b.throttle = 1 })
			}
			out := queueOutcome(t, w, ctx, "hue.scene-studio-3f2a9c10", "press", "", 0)
			w.handle(ctx, <-w.requests)
			select {
			case got := <-out:
				t.Fatalf("completed before HTTP: %+v", got)
			default:
			}
			applyHueResult(t, w, ctx)
			status := snoofer.Processed
			if fail {
				status = snoofer.Failed
			}
			awaitHueOutcome(t, out, status)
			if len(b.putLog()) != 1 {
				t.Fatal(b.putLog())
			}
		})
	}
}

func TestHueSceneSyncPreparationOutcomes(t *testing.T) {
	for _, mode := range []string{"stopped", "busy", "lost", "timeout", "bridge missing"} {
		t.Run(mode, func(t *testing.T) {
			w, b, ctx := outcomeWorker(t, studio())
			app := newFakeSyncApp(t, syncStateSyncing)
			app.set(func(a *fakeSyncApp) { a.ignore = true })
			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(app.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			w.sync.conn = conn
			w.sync.known = true
			w.sync.generation = 1
			w.sync.state.State = syncStateSyncing
			if mode == "busy" {
				wanted := true
				w.sync.wanted = &wanted
			}
			out := queueOutcome(t, w, ctx, "hue.scene-studio-3f2a9c10", "press", "", 0)
			w.handle(ctx, <-w.requests)
			if mode == "busy" {
				awaitHueOutcome(t, out, snoofer.Rejected)
				return
			}
			select {
			case got := <-out:
				t.Fatalf("completed during Sync preparation: %+v", got)
			default:
			}
			if len(b.putLog()) != 0 {
				t.Fatal("scene sent before stop")
			}
			switch mode {
			case "stopped":
				w.syncObserve(ctx, syncState{State: "bridge_connected"})
				applyHueResult(t, w, ctx)
				awaitHueOutcome(t, out, snoofer.Processed)
			case "lost":
				w.syncLost(errors.New("socket lost"))
				awaitHueOutcome(t, out, snoofer.Unknown)
			case "timeout":
				w.syncTick(ctx, time.Now().Add(w.timing.confirm+time.Second))
				awaitHueOutcome(t, out, snoofer.Unknown)
			case "bridge missing":
				w.syncObserve(ctx, syncState{State: syncStateDisconnected})
				awaitHueOutcome(t, out, snoofer.Failed)
			}
			if mode != "stopped" && len(b.putLog()) != 0 {
				t.Fatal("scene sent after failed stop")
			}
		})
	}
}

func TestHueQueuedStagePressCannotApplyLaterStage(t *testing.T) {
	w, _, ctx := outcomeWorker(t, lounge())
	d, _ := findGroupDial("hue.decor")
	if err := w.adjustGroupDial(d, snoofer.Request{ID: d.id, Operation: "adjust", Delta: 1}, false); err != nil {
		t.Fatal(err)
	}
	out := queueOutcome(t, w, ctx, d.id, "press", "", 0)
	if err := w.adjustGroupDial(d, snoofer.Request{ID: d.id, Operation: "adjust", Delta: 1}, false); err != nil {
		t.Fatal(err)
	}
	w.handle(ctx, <-w.requests)
	awaitHueOutcome(t, out, snoofer.Rejected)
	if len(w.lights.queue) != 0 {
		t.Fatal("queued press applied a later stage")
	}
}

func TestHuePairIgnoresDisconnectedDiagnosticIdentity(t *testing.T) {
	w, b, ctx := outcomeWorker(t, studio())
	w.connected = false
	w.status = "Error"
	w.bridgeInfo = identity{BridgeID: "retired-bridge"}
	b.set(func(b *fakeBridge) { b.linkPressed = true })
	out := queueOutcome(t, w, ctx, "hue.pair", "press", "", 0)
	w.handle(ctx, <-w.requests)
	applyHueResult(t, w, ctx)
	awaitHueOutcome(t, out, snoofer.Processed)
}

func TestHuePairSettingsChangeReportsUnsavedCredentials(t *testing.T) {
	w, b, ctx := outcomeWorker(t, studio())
	b.set(func(b *fakeBridge) { b.linkPressed = true })
	out := queueOutcome(t, w, ctx, "hue.pair", "press", "", 0)
	w.handle(ctx, <-w.requests)
	w.settingsEpoch++
	applyHueResult(t, w, ctx)
	got := awaitHueOutcome(t, out, snoofer.Failed)
	if !strings.Contains(got.Detail, "not saved") {
		t.Fatal(got)
	}
}

func TestHueSyncReportsAcceptanceAndAmbiguousWriteFailure(t *testing.T) {
	for _, closing := range []bool{false, true} {
		t.Run(fmt.Sprint(closing), func(t *testing.T) {
			w, _, ctx := outcomeWorker(t, studio())
			app := newFakeSyncApp(t, "bridge_connected")
			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(app.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			w.sync.conn = conn
			w.sync.known = true
			w.sync.generation = 1
			w.sync.state.State = "bridge_connected"
			out := queueOutcome(t, w, ctx, "hue.sync", "press", "", 0)
			select {
			case got := <-out:
				t.Fatalf("enqueue claimed socket acceptance: %+v", got)
			default:
			}
			if closing {
				if err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
			}
			w.handle(ctx, <-w.requests)
			want := snoofer.Submitted
			if closing {
				want = snoofer.Unknown
			}
			awaitHueOutcome(t, out, want)
		})
	}
}

func TestHuePairResponseTimeoutIsUnknown(t *testing.T) {
	w, b, ctx := outcomeWorker(t, studio())
	w.timing.pairWindow = 100 * time.Millisecond
	previous := b.Config.Handler
	started := make(chan struct{})
	b.Config.Handler = http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api" {
			close(started)
			time.Sleep(200 * time.Millisecond)
			return
		}
		previous.ServeHTTP(writer, r)
	})
	out := queueOutcome(t, w, ctx, "hue.pair", "press", "", 0)
	w.handle(ctx, <-w.requests)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("pair request not sent")
	}
	applyHueResult(t, w, ctx)
	awaitHueOutcome(t, out, snoofer.Unknown)
}
