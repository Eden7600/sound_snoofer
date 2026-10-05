package snoofer

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

type testInstance struct{ stop func() error }

func (i testInstance) Stop(context.Context) error { return i.stop() }
func TestLifecycle(t *testing.T) {
	var log []string
	mk := func(id string, deps ...string) Plugin {
		return Plugin{ID: id, Requires: deps, Start: func(_ context.Context, _ Services, _ json.RawMessage, d map[string]Instance) (Instance, error) {
			log = append(log, "start "+id)
			for _, dep := range deps {
				if d[dep] == nil {
					t.Fatal("missing direct dependency")
				}
			}
			return testInstance{func() error { log = append(log, "stop "+id); return nil }}, nil
		}}
	}
	c := Config{Version: 1, Plugins: map[string]PluginConfig{"audio": {Enabled: true}, "vr": {Enabled: true}, "deck": {}, "bad": {Enabled: true}, "independent": {Enabled: true}}}
	bad := mk("bad")
	bad.Start = func(context.Context, Services, json.RawMessage, map[string]Instance) (Instance, error) {
		return nil, errors.New("failed")
	}
	h := New(c, Services{Controls: NewControls()}, mk("audio"), mk("vr", "audio"), mk("deck"), bad, mk("independent"))
	h.Start(context.Background())
	h.Retry()
	if h.Status()["deck"] != "Disabled" || h.Status()["bad"] != "failed" {
		t.Fatal(h.Status())
	}
	next, err := h.Selection("audio", false)
	if err != nil || next.Plugins["vr"].Enabled {
		t.Fatal(next, err)
	}
	if err := h.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"start audio", "start independent", "start vr", "stop vr", "stop independent", "stop audio"}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("%v", log)
	}
}
func TestInvalidBranches(t *testing.T) {
	p := func(id string, deps ...string) Plugin {
		return Plugin{ID: id, Requires: deps, Start: func(context.Context, Services, json.RawMessage, map[string]Instance) (Instance, error) {
			t.Fatal("invalid branch started")
			return nil, nil
		}}
	}
	c := Config{Plugins: map[string]PluginConfig{"a": {Enabled: true}, "b": {Enabled: true}, "missing": {Enabled: true}, "dup": {Enabled: true}}}
	h := New(c, Services{Controls: NewControls()}, p("a", "b"), p("b", "a"), p("dup"), p("dup"))
	h.Start(context.Background())
	for _, s := range h.Status() {
		if s == "Running" {
			t.Fatal(h.Status())
		}
	}
}
func TestStopDeadline(t *testing.T) {
	release := make(chan struct{})
	p := Plugin{ID: "slow", Start: func(context.Context, Services, json.RawMessage, map[string]Instance) (Instance, error) {
		return testInstance{func() error { <-release; return nil }}, nil
	}}
	h := New(Config{Plugins: map[string]PluginConfig{"slow": {Enabled: true}}}, Services{Controls: NewControls()}, p)
	h.Start(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if !errors.Is(h.Stop(ctx), context.DeadlineExceeded) {
		t.Fatal("missing timeout")
	}
	close(release)
}
func TestControlsStaleAndUnavailable(t *testing.T) {
	c := NewControls()
	calls := 0
	v := Control{ID: "a.toggle", Available: true, Operations: []string{"press"}}
	handler := func(context.Context, Request) error { calls++; return nil }
	if err := c.Publish("a", []Control{v}, handler); err != nil {
		t.Fatal(err)
	}
	r := Request{ID: v.ID, Revision: c.Snapshot()[0].Revision, Operation: "press"}
	if err := c.Dispatch(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	c.Remove("a")
	_ = c.Publish("a", []Control{v}, handler)
	if err := c.Dispatch(context.Background(), r); err == nil {
		t.Fatal("replayed stale input")
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

func TestDisabledDoesNotValidateOrStart(t *testing.T) {
	p := Plugin{ID: "off", Validate: func(json.RawMessage) error { t.Fatal("disabled validator ran"); return nil }, Start: func(context.Context, Services, json.RawMessage, map[string]Instance) (Instance, error) {
		t.Fatal("disabled factory ran")
		return nil, nil
	}}
	cfg := Config{Version: 1, Plugins: map[string]PluginConfig{"off": {Settings: json.RawMessage(`{"not_its_schema":true}`)}}}
	if err := ValidateEnabled(cfg, p); err != nil {
		t.Fatal(err)
	}
	h := New(cfg, Services{}, p)
	h.Start(context.Background())
	if err := h.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}
