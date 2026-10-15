package hue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestOldRoomLightCompletionDoesNotChangeNewQueue(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.WriteHeader(status)
				_, _ = response.Write([]byte(`{"errors":[],"data":[]}`))
			}))
			defer server.Close()
			w := newWorker(snoofer.Services{SaveSettings: func(string, json.RawMessage, json.RawMessage) error { return nil }}, Settings{}, nil, nil, testTiming)
			w.client = &Client{base: server.URL, requests: server.Client()}
			w.selectGroup("old-room")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer func() { cancel(); w.children.Wait() }()
			w.sendLight(ctx, lightWrite{dial: "hue.decor", light: "old-light"})
			var complete func(context.Context)
			select {
			case complete = <-w.results:
			case <-ctx.Done():
				t.Fatal("light write did not finish")
			}
			// The network result was queued while the actor switched to another room.
			w.selectGroup("new-room")
			w.lights.inFlight, w.lights.sending = true, "hue.mixed"
			w.lights.queue = []lightWrite{{dial: "hue.mixed", light: "new-light"}}
			w.lights.gap, w.lights.lastSend = time.Second, time.Now()
			w.lights.errs = map[string]string{"hue.mixed": "current error"}
			before := w.lights
			before.queue = append([]lightWrite(nil), w.lights.queue...)
			before.errs = map[string]string{"hue.mixed": "current error"}
			complete(ctx)
			if !reflect.DeepEqual(w.lights, before) {
				t.Fatalf("old-room result changed current queue: before=%+v after=%+v", before, w.lights)
			}
		})
	}
}
