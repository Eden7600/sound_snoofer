package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestGenericPrivateFrames(t *testing.T) {
	state := ViewState{Controls: []snoofer.Control{{ID: "external.light", Revision: 9, Value: "On"}}, Plugins: map[string]string{"external": "Running"}, PluginLabels: map[string]string{"external": "External Lights"}}
	data, err := json.Marshal(frame{State: &state, Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	updates := make(chan []byte, 1)
	offer(updates, data)
	state.Notice = "new"
	next, _ := json.Marshal(frame{State: &state})
	offer(updates, next)
	var f frame
	if err := json.Unmarshal(<-updates, &f); err != nil {
		t.Fatal(err)
	}
	if !f.Focus || !reflect.DeepEqual(*f.State, state) {
		t.Fatal(f)
	}
	action := UIAction{Request: &snoofer.Request{ID: "external.light", Revision: 9, Operation: "press"}}
	data, _ = json.Marshal(action)
	queue := make(chan UIAction, 1)
	if err := readActions(context.Background(), bytes.NewReader(data), queue); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(<-queue, action) {
		t.Fatal("lost semantic action")
	}
}
func TestClosedAndCancelledTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := readActions(ctx, bytes.NewBufferString("{}"), make(chan UIAction)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	r.Close()
	defer w.Close()
	updates := make(chan []byte, 1)
	updates <- []byte("{}")
	if err := writeFrames(context.Background(), w, updates); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

func decodeFrame(t testing.TB, data []byte) frame {
	t.Helper()
	var f frame
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func prepareFrame(t testing.TB, sender *viewSender, state ViewState, focus bool) ([]byte, frame) {
	t.Helper()
	data, err := sender.prepare(state, focus)
	if err != nil {
		t.Fatal(err)
	}
	return data, decodeFrame(t, data)
}

func artworkState() ViewState {
	return ViewState{Controls: []snoofer.Control{{ID: "clip", Revision: 9, Artwork: "first", Animation: []snoofer.ArtworkFrame{{Artwork: "first", Delay: time.Second}, {Artwork: "second", Delay: time.Second}}}}}
}

func TestViewSenderTelemetryAndPresentation(t *testing.T) {
	var sender viewSender
	state := artworkState()
	_, first := prepareFrame(t, &sender, state, false)
	if first.Version != 1 || first.Artwork == nil || first.Animations == nil || first.State.Controls[0].Artwork != "" {
		t.Fatal("incomplete initial frame", first)
	}
	for name, change := range map[string]func(){
		"meter": func() { state.Controls[0].Meter = snoofer.Meter{Present: true, Known: true, DB: -15, At: time.Now()} },
		"progress": func() {
			state.Controls[0].Progress = snoofer.Progress{Known: true, Playing: true, PositionMs: 100, At: time.Now()}
		},
		"view data":    func() { state.Controls[0].ViewData = json.RawMessage(`{"Sessions":[]}`) },
		"plugin label": func() { state.PluginLabels = map[string]string{"custom": "Custom"} },
		"plugin state": func() { state.Plugins = map[string]string{"custom": "Running"} },
		"enablement":   func() { state.Enabled = map[string]bool{"custom": true} },
		"notice":       func() { state.Notice = "notice" },
		"confirmation": func() { state.Confirmation = "confirm" },
	} {
		t.Run(name, func(t *testing.T) {
			change()
			_, next := prepareFrame(t, &sender, state, false)
			if next.Version != first.Version+1 || next.Artwork != nil || next.Animations != nil || next.State.Controls[0].Revision != 9 {
				t.Fatal("presentation change lost or repeated assets", next)
			}
			stripped, _, _ := separateArtwork(state)
			want, _ := json.Marshal(stripped)
			got, _ := json.Marshal(next.State)
			if !bytes.Equal(got, want) {
				t.Fatal("snapshot changed in transit")
			}
			first = next
		})
	}
	if unchanged, err := sender.prepare(state, false); err != nil || unchanged != nil {
		t.Fatal("unchanged state encoded", err)
	}
	_, focused := prepareFrame(t, &sender, state, true)
	if !focused.Focus || focused.Version != first.Version || focused.Artwork != nil || focused.Animations != nil {
		t.Fatal("unchanged focus changed presentation", focused)
	}
	var reopened viewSender
	_, fresh := prepareFrame(t, &reopened, state, false)
	if fresh.Version != 1 || fresh.Artwork == nil || fresh.Animations == nil {
		t.Fatal("new child inherited old baselines")
	}
}

func TestViewSenderAssetsAndFailureRecovery(t *testing.T) {
	var sender viewSender
	state := artworkState()
	before := state.Controls[0]
	before.Animation = append([]snoofer.ArtworkFrame(nil), before.Animation...)
	prepareFrame(t, &sender, state, false)
	if !reflect.DeepEqual(state.Controls[0], before) {
		t.Fatal("provider control was mutated")
	}
	state.Controls[0].Artwork = "replacement"
	state.Controls[0].ViewData = json.RawMessage(`{`)
	if _, err := sender.prepare(state, false); err == nil {
		t.Fatal("invalid provider JSON accepted")
	}
	state.Controls[0].ViewData = nil
	_, recovered := prepareFrame(t, &sender, state, false)
	if recovered.Version != 2 || recovered.Artwork == nil || (*recovered.Artwork)["clip"] != "replacement" {
		t.Fatal("failed preparation advanced the asset baseline", recovered)
	}
	state.Controls = append(state.Controls, snoofer.Control{ID: "still", Artwork: "added"})
	_, added := prepareFrame(t, &sender, state, false)
	if added.Artwork == nil || len(*added.Artwork) != 2 {
		t.Fatal("map replacement omitted an entry")
	}
	state.Controls[0].Artwork = ""
	_, removed := prepareFrame(t, &sender, state, false)
	if removed.Artwork == nil || len(*removed.Artwork) != 1 || (*removed.Artwork)["still"] != "added" {
		t.Fatal("removed art remained cached")
	}
	state.Controls = nil
	_, cleared := prepareFrame(t, &sender, state, false)
	if cleared.Artwork == nil || len(*cleared.Artwork) != 0 || cleared.Animations == nil || len(*cleared.Animations) != 0 {
		t.Fatal("empty replacements were omitted", cleared)
	}
	if data, err := sender.prepare(state, false); err != nil || data != nil {
		t.Fatal("empty maps were repeated", err)
	}
}

func TestViewSenderCompleteAnimations(t *testing.T) {
	var sender viewSender
	state := artworkState()
	prepareFrame(t, &sender, state, false)
	for _, replacement := range [][]snoofer.ArtworkFrame{
		{{Artwork: "first", Delay: time.Second}, {Artwork: "new second", Delay: time.Second}},
		{{Artwork: "first", Delay: time.Second}, {Artwork: "new second", Delay: 2 * time.Second}},
		nil,
	} {
		state.Controls[0].Animation = replacement
		_, next := prepareFrame(t, &sender, state, false)
		if next.Artwork != nil || next.Animations == nil || !reflect.DeepEqual((*next.Animations)["clip"], replacement) {
			t.Fatal("complete animation update was lost", next)
		}
	}
}

func TestOfferRetainsMapsFocusAndReversion(t *testing.T) {
	var sender viewSender
	updates := make(chan []byte, 1)
	state := artworkState()
	data, _ := prepareFrame(t, &sender, state, true)
	offer(updates, data)
	state.Controls[0].Artwork = "changed"
	data, _ = prepareFrame(t, &sender, state, false)
	offer(updates, data)
	state.Controls[0].Artwork = "first"
	data, _ = prepareFrame(t, &sender, state, false)
	offer(updates, data)
	state.Notice = "latest"
	data, last := prepareFrame(t, &sender, state, false)
	offer(updates, data)
	merged := decodeFrame(t, <-updates)
	if merged.Version != last.Version || merged.State.Notice != "latest" || !merged.Focus || merged.Artwork == nil || (*merged.Artwork)["clip"] != "first" || merged.Animations == nil {
		t.Fatal("coalescing lost state, focus, maps or reversion", merged)
	}
	// Newer explicit empty maps replace older populated maps, even before a read.
	var fresh viewSender
	data, _ = prepareFrame(t, &fresh, state, false)
	offer(updates, data)
	state.Controls = nil
	data, _ = prepareFrame(t, &fresh, state, false)
	offer(updates, data)
	data, last = prepareFrame(t, &fresh, state, true)
	offer(updates, data)
	merged = decodeFrame(t, <-updates)
	if merged.Version != last.Version || !merged.Focus || merged.Artwork == nil || len(*merged.Artwork) != 0 || merged.Animations == nil || len(*merged.Animations) != 0 {
		t.Fatal("focus replacement resurrected removed assets", merged)
	}
}
