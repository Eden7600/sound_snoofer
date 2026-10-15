//go:build windows

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestDesktopWaitsForActionResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := Desktop{ctx: ctx, actions: make(chan UIAction, 1)}
	// Decode the private wire shape so this regression also runs before the new field exists.
	var action UIAction
	if err := json.Unmarshal([]byte(`{"ID":"first","Kind":"retry"}`), &action); err != nil {
		t.Fatal(err)
	}
	result := make(chan any, 1)
	go func() { result <- d.Send(action) }()
	select {
	case <-d.actions:
	case <-time.After(time.Second):
		t.Fatal("action was not queued")
	}
	select {
	case got := <-result:
		t.Fatalf("enqueue was incorrectly returned as an action result: %#v", got)
	case <-time.After(30 * time.Millisecond):
	}
	cancel()
	select {
	case got := <-result:
		data, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(data, &fields); err != nil || fields["ID"] != "first" || fields["Status"] != "Unknown" {
			t.Fatalf("disconnect lost action identity/uncertainty: %s (%v)", data, err)
		}
	case <-time.After(time.Second):
		t.Fatal("disconnect did not settle Send")
	}
}

func startDesktopAction(t *testing.T, d *Desktop, id string) <-chan ActionResult {
	t.Helper()
	result := make(chan ActionResult, 1)
	go func() { result <- d.Send(UIAction{ID: id, Kind: "retry"}) }()
	select {
	case action := <-d.actions:
		if action.ID != id {
			t.Fatal("wrong action queued", action)
		}
	case <-time.After(time.Second):
		t.Fatal("action was not queued", id)
	}
	return result
}

func desktopResult(t *testing.T, result <-chan ActionResult) ActionResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(time.Second):
		t.Fatal("Send did not settle")
		return ActionResult{}
	}
}

func TestDesktopPendingCapacityDoesNotBlockState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &Desktop{ctx: ctx, actions: make(chan UIAction, actionCapacity)}
	state := artworkState()
	d.applyFrame(frame{State: &state})
	results := make([]<-chan ActionResult, 0, actionCapacity)
	for n := range actionCapacity {
		results = append(results, startDesktopAction(t, d, fmt.Sprint(n)))
	}
	if got := d.Send(UIAction{ID: "extra"}); got.Status != snoofer.Rejected {
		t.Fatal("pending result capacity exceeded", got)
	}
	if got := d.Send(UIAction{ID: "0"}); got.Status != snoofer.Rejected || got.Detail != "Action already pending" {
		t.Fatal("duplicate pending action queued", got)
	}
	if d.State(0) == nil || d.State(d.version) != nil {
		t.Fatal("pending Send changed state availability")
	}
	d.Animation("clip")
	if len(d.actions) != 0 {
		t.Fatal("refused actions were queued")
	}
	cancel()
	for n, result := range results {
		if got := desktopResult(t, result); got.ID != fmt.Sprint(n) || got.Status != snoofer.Unknown {
			t.Fatal("disconnect lost an outstanding result", got)
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.pending) != 0 {
		t.Fatal("disconnect leaked pending reservations")
	}
}

func TestDesktopCorrelatesResultsWithoutChangingView(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &Desktop{ctx: ctx, actions: make(chan UIAction, 2)}
	state := artworkState()
	d.applyFrame(frame{State: &state})
	version := d.version
	first, second := startDesktopAction(t, d, "first"), startDesktopAction(t, d, "second")
	d.applyFrame(frame{Result: &ActionResult{ID: "absent", Status: snoofer.Processed}})
	d.applyFrame(frame{Result: &ActionResult{ID: "second", Status: snoofer.Failed, Detail: "save refused"}})
	if got := desktopResult(t, second); got.ID != "second" || got.Status != snoofer.Failed {
		t.Fatal("wrong second result", got)
	}
	state.Notice = "unrelated notice"
	d.applyFrame(frame{Version: version + 1, State: &state})
	select {
	case got := <-first:
		t.Fatal("another result or state change settled the first action", got)
	default:
	}
	d.applyFrame(frame{Result: &ActionResult{ID: "first", Status: snoofer.Processed}})
	d.applyFrame(frame{Result: &ActionResult{ID: "first", Status: snoofer.Failed}})
	if got := desktopResult(t, first); got.ID != "first" || got.Status != snoofer.Processed {
		t.Fatal("duplicate replaced the terminal result", got)
	}
	if d.State(version+1) != nil {
		t.Fatal("result frame invalidated presentation version")
	}
}
