package hue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func hueEdgeResult(t *testing.T, results <-chan snoofer.Outcome, want snoofer.OutcomeStatus) {
	t.Helper()
	select {
	case got := <-results:
		if got.Status != want {
			t.Fatalf("outcome = %+v, want %s", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("missing outcome")
	}
}

func hueEdgeNoResult(t *testing.T, results <-chan snoofer.Outcome) {
	t.Helper()
	select {
	case got := <-results:
		t.Fatalf("duplicate or premature outcome: %+v", got)
	default:
	}
}

func hueEdgeRecreate(t *testing.T, w *worker, id string) {
	t.Helper()
	r, ok := w.model.resources[id]
	if !ok {
		t.Fatal("missing resource", id)
	}
	w.model.apply([]event{{Type: "delete", Data: []resource{{ID: id}}}, {Type: "add", Data: []resource{r}}})
}

func hueEdgeRequest(t *testing.T, w *worker, id string) snoofer.Request {
	t.Helper()
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID == id {
			return snoofer.Request{ID: id, Revision: c.Revision, Operation: "press"}
		}
	}
	t.Fatal("missing control", id)
	return snoofer.Request{}
}

func TestHueGraphLifetimeRejectsDrawnAndQueuedCommands(t *testing.T) {
	for _, target := range []struct{ control, resource string }{
		{"hue.brightness", "room-1"},
		{"hue.brightness", "gl-1"},
		{"hue.motion", "motion-1"},
		{"hue.motion", "dev-4"},
		{"hue.room-scene-1", "3f2a9c10-aaaa-bbbb-cccc-000000000001"},
	} {
		for _, queued := range []bool{false, true} {
			name := target.resource + "/drawn"
			if queued {
				name = target.resource + "/queued"
			}
			t.Run(name, func(t *testing.T) {
				w, bridge, ctx := outcomeWorker(t, sensorStudio())
				w.publish(ctx)
				r := hueEdgeRequest(t, w, target.control)
				results := make(chan snoofer.Outcome, 2)
				dispatch := func() error {
					return w.services.Controls.DispatchWithResult(ctx, r, func(o snoofer.Outcome) { results <- o })
				}
				if queued {
					if err := dispatch(); err != nil {
						t.Fatal(err)
					}
					hueEdgeNoResult(t, results)
				}
				// The ID and all visible values return to their original values.
				hueEdgeRecreate(t, w, target.resource)
				if queued {
					w.handle(ctx, <-w.requests)
				} else {
					w.publish(ctx)
					if err := dispatch(); err == nil {
						t.Fatal("old drawn revision accepted after resource recreation")
					}
					if len(w.requests) != 0 {
						t.Fatal("stale drawn command reached worker queue")
					}
				}
				hueEdgeResult(t, results, snoofer.Rejected)
				w.children.Wait()
				if w.group.dirty || w.sceneAction != nil || w.motionAction != nil || len(bridge.putLog()) != 0 {
					t.Fatal("obsolete command began work after resource recreation")
				}
				hueEdgeNoResult(t, results)
			})
		}
	}
}

func TestHueMembershipABACannotRestoreQueuedTarget(t *testing.T) {
	w, bridge, ctx := outcomeWorker(t, studio())
	results := queueOutcome(t, w, ctx, "hue.brightness", "press", "", 0)
	room := w.model.resources["room-1"]
	w.model.apply([]event{{Type: "update", Data: []resource{{ID: room.ID, Children: []reference{}}}}})
	w.model.apply([]event{{Type: "update", Data: []resource{{ID: room.ID, Children: room.Children}}}})
	w.handle(ctx, <-w.requests)
	hueEdgeResult(t, results, snoofer.Rejected)
	if w.group.dirty || len(bridge.putLog()) != 0 {
		t.Fatal("membership ABA revived an obsolete room command")
	}
}

func TestHueSubmittedAccumulatorsDropRecreatedTargets(t *testing.T) {
	for _, control := range []string{"hue.brightness", "hue.decor"} {
		t.Run(control, func(t *testing.T) {
			w, bridge, ctx := outcomeWorker(t, lounge())
			results := queueOutcome(t, w, ctx, control, "press", "", 0)
			w.handle(ctx, <-w.requests)
			hueEdgeResult(t, results, snoofer.Submitted)
			resource := "gl-1"
			if control == "hue.decor" {
				resource = "strip"
				if len(w.lights.queue) != 1 {
					t.Fatal("light write was not queued")
				}
			} else if !w.group.dirty {
				t.Fatal("brightness write was not queued")
			}
			hueEdgeRecreate(t, w, resource)
			if control == "hue.brightness" {
				w.sendGroup(ctx)
			} else {
				w.tickLights(ctx, time.Now().Add(time.Second))
			}
			w.children.Wait()
			if len(bridge.putLog()) != 0 || w.group.dirty || len(w.lights.queue) != 0 {
				t.Fatal("deferred write reached a recreated resource")
			}
			hueEdgeNoResult(t, results) // Submission is the established boundary.
		})
	}
}

func TestHueQueuedClaimsSettleOnceOnFullQueueAndStop(t *testing.T) {
	w, bridge, parent := outcomeWorker(t, studio())
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var accepted []<-chan snoofer.Outcome
	for range cap(w.requests) {
		accepted = append(accepted, queueOutcome(t, w, ctx, "hue.brightness", "press", "", 0))
	}
	refused := make(chan snoofer.Outcome, 2)
	if err := w.services.Controls.DispatchWithResult(ctx, hueEdgeRequest(t, w, "hue.brightness"), func(o snoofer.Outcome) { refused <- o }); err == nil {
		t.Fatal("full queue accepted another command")
	}
	hueEdgeResult(t, refused, snoofer.Rejected)
	cancel()
	for _, results := range accepted {
		hueEdgeResult(t, results, snoofer.Unknown)
		w.handle(ctx, <-w.requests)
		hueEdgeNoResult(t, results)
	}
	hueEdgeNoResult(t, refused)
	if w.group.dirty || len(bridge.putLog()) != 0 {
		t.Fatal("cancelled queued input performed work")
	}
}

func TestHueAsyncClaimsSettleWhenCallbacksCannotApply(t *testing.T) {
	for _, stopped := range []string{"cancel-before-callback", "discard-callback", "disconnect-before-callback"} {
		t.Run(stopped, func(t *testing.T) {
			w, bridge, parent := outcomeWorker(t, sensorStudio())
			ctx, cancel := context.WithCancel(parent)
			defer cancel()
			if stopped == "discard-callback" {
				// Fill only the actor result queue so network completions cannot post.
				for range cap(w.results) {
					w.results <- func(context.Context) {}
				}
			}
			motion := queueOutcome(t, w, ctx, "hue.motion", "press", "", 0)
			w.handle(ctx, <-w.requests)
			scene := queueOutcome(t, w, ctx, "hue.room-scene-1", "press", "", 0)
			w.handle(ctx, <-w.requests)
			waitFor(t, func() bool { return len(bridge.putLog()) == 3 })
			if stopped != "discard-callback" {
				w.children.Wait() // All three successful HTTP results await the actor.
			}
			hueEdgeNoResult(t, motion)
			hueEdgeNoResult(t, scene)
			if stopped == "disconnect-before-callback" {
				w.disconnect(errors.New("bridge connection lost"))
			} else {
				cancel()
			}
			hueEdgeResult(t, motion, snoofer.Unknown)
			hueEdgeResult(t, scene, snoofer.Unknown)
			w.children.Wait()
			for len(w.results) > 0 {
				(<-w.results)(ctx)
			}
			hueEdgeNoResult(t, motion)
			hueEdgeNoResult(t, scene)
			if len(bridge.putLog()) != 3 {
				t.Fatal("a discarded or late completion replayed an operation")
			}
		})
	}
}

func hueEdgeHTTPStatus(t *testing.T, w *worker, status int) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(status)
		_, _ = response.Write([]byte(`{"errors":[],"data":[]}`))
	}))
	t.Cleanup(server.Close)
	w.client.close()
	done, cancel := context.WithCancel(context.Background())
	w.client = &Client{base: server.URL, requests: server.Client(), done: done, cancel: cancel}
}

func hueEdgeCallback(t *testing.T, w *worker) func(context.Context) {
	t.Helper()
	select {
	case apply := <-w.results:
		return apply
	case <-time.After(time.Second):
		t.Fatal("HTTP completion did not reach actor")
		return nil
	}
}

func TestHueOldBrightnessCallbackCannotChangeReturnedRoom(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			w, _, ctx := outcomeWorker(t, studio())
			hueEdgeHTTPStatus(t, w, status)
			first := queueOutcome(t, w, ctx, "hue.brightness", "adjust", "", 1)
			w.handle(ctx, <-w.requests)
			hueEdgeResult(t, first, snoofer.Submitted)
			w.sendGroup(ctx)
			apply := hueEdgeCallback(t, w)
			if err := w.selectGroup("zone-1"); err != nil {
				t.Fatal(err)
			}
			if err := w.selectGroup("room-1"); err != nil {
				t.Fatal(err)
			}
			second := queueOutcome(t, w, ctx, "hue.brightness", "adjust", "", -1)
			w.handle(ctx, <-w.requests)
			hueEdgeResult(t, second, snoofer.Submitted)
			w.group.inFlight = true // A newer write owns the current accumulator.
			w.group.gap, w.group.lastSend = 2*time.Second, time.Unix(1000, 0)
			w.brightErr = "Current room error"
			before := w.group
			apply(ctx)
			if !reflect.DeepEqual(w.group, before) || w.brightErr != "Current room error" {
				t.Fatalf("old callback changed returned room: before=%+v after=%+v error=%q", before, w.group, w.brightErr)
			}
			hueEdgeNoResult(t, first)
			hueEdgeNoResult(t, second)
		})
	}
}

func TestHueLateSceneResultDoesNotAnnotateRecreatedScene(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			w, _, ctx := outcomeWorker(t, studio())
			hueEdgeHTTPStatus(t, w, status)
			results := queueOutcome(t, w, ctx, "hue.room-scene-1", "press", "", 0)
			w.handle(ctx, <-w.requests)
			apply := hueEdgeCallback(t, w)
			scene := "3f2a9c10-aaaa-bbbb-cccc-000000000001"
			hueEdgeRecreate(t, w, scene)
			apply(ctx)
			want := snoofer.Processed
			if status != http.StatusOK {
				want = snoofer.Failed
			}
			hueEdgeResult(t, results, want) // The old HTTP call still has its own result.
			if w.sceneAction != nil || w.sceneWriting != "" || len(w.scenePending) != 0 || w.sceneErr[scene] != "" {
				t.Fatal("old HTTP result was attributed to a recreated scene")
			}
			hueEdgeNoResult(t, results)
		})
	}
}

func TestHueLateLightResultDoesNotAnnotateRecreatedLight(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			w, _, ctx := outcomeWorker(t, lounge())
			hueEdgeHTTPStatus(t, w, status)
			results := queueOutcome(t, w, ctx, "hue.decor", "press", "", 0)
			w.handle(ctx, <-w.requests)
			hueEdgeResult(t, results, snoofer.Submitted)
			w.tickLights(ctx, time.Now())
			apply := hueEdgeCallback(t, w)
			hueEdgeRecreate(t, w, "strip")
			apply(ctx)
			if w.lights.inFlight || w.lights.sending != "" || len(w.lights.waiting) != 0 || len(w.lights.queue) != 0 || w.lights.errs["hue.decor"] != "" {
				t.Fatalf("old HTTP result was attributed to a recreated light: %+v", w.lights)
			}
			hueEdgeNoResult(t, results)
		})
	}
}
