package snoofer

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"strings"
	"testing"
)

func TestLifecycleStatesIgnoreStatusTextAndRecover(t *testing.T) {
	failing := true
	config := Config{Plugins: map[string]PluginConfig{
		"healthy": {Enabled: true}, "disabled": {}, "absent": {Enabled: true},
		"absent-off": {}, "dependent": {Enabled: true},
	}}
	started := map[string]int{}
	plugin := func(id, failure string, requires ...string) Plugin {
		return Plugin{ID: id, Requires: requires, Start: func(context.Context, Services, json.RawMessage, map[string]Instance) (Instance, error) {
			started[id]++
			if failing && failure != "" {
				return nil, errors.New(failure)
			}
			return testInstance{stop: func() error { return nil }}, nil
		}}
	}
	plugins := []Plugin{plugin("healthy", ""), plugin("disabled", ""), plugin("omitted", ""), plugin("dependent", "", "bad-running")}
	for _, text := range []string{"Running", "Disabled", "Saved"} {
		id := "bad-" + strings.ToLower(text)
		config.Plugins[id] = PluginConfig{Enabled: true}
		plugins = append(plugins, plugin(id, text))
	}
	host := New(config, Services{}, plugins...)
	want := map[string]PluginState{
		"healthy": PluginStarting, "disabled": PluginDisabled, "omitted": PluginDisabled,
		"absent": PluginMissing, "absent-off": PluginMissing, "dependent": PluginStarting,
		"bad-running": PluginStarting, "bad-disabled": PluginStarting, "bad-saved": PluginStarting,
	}
	if got := host.Lifecycle(); !maps.Equal(got, want) {
		t.Fatalf("before Start: %v, want %v", got, want)
	}
	host.Start(context.Background())
	t.Cleanup(func() {
		if err := host.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	want["healthy"] = PluginRunning
	want["dependent"] = PluginFailed
	for _, text := range []string{"Running", "Disabled", "Saved"} {
		id := "bad-" + strings.ToLower(text)
		want[id] = PluginFailed
		if host.Status()[id] != text {
			t.Fatalf("failure detail changed: %s", host.Status()[id])
		}
	}
	if got := host.Lifecycle(); !maps.Equal(got, want) {
		t.Fatalf("after failed Start: %v, want %v", got, want)
	}
	copy := host.Lifecycle()
	copy["healthy"] = PluginFailed
	delete(copy, "dependent")
	if got := host.Lifecycle(); !maps.Equal(got, want) {
		t.Fatalf("returned map changed host state: %v", got)
	}
	failing = false
	host.Retry()
	for _, id := range []string{"dependent", "bad-running", "bad-disabled", "bad-saved"} {
		want[id] = PluginRunning
	}
	if got := host.Lifecycle(); !maps.Equal(got, want) {
		t.Fatalf("after Retry: %v, want %v", got, want)
	}
	if started["healthy"] != 1 || started["disabled"] != 0 || started["omitted"] != 0 {
		t.Fatalf("retry duplicated a healthy plugin or started a disabled plugin: %v", started)
	}
}

func TestLifecycleStatesIncludeInvalidBranches(t *testing.T) {
	start := func(context.Context, Services, json.RawMessage, map[string]Instance) (Instance, error) {
		t.Fatal("invalid branch started")
		return nil, nil
	}
	host := New(Config{Plugins: map[string]PluginConfig{
		"duplicate": {Enabled: true}, "invalid": {Enabled: true}, "dependent": {Enabled: true},
	}}, Services{}, Plugin{ID: "duplicate", Start: start}, Plugin{ID: "duplicate", Start: start},
		Plugin{ID: "invalid", Start: start, Validate: func(json.RawMessage) error { return errors.New("Running") }},
		Plugin{ID: "dependent", Start: start, Requires: []string{"missing-dependency"}})
	host.Start(context.Background())
	t.Cleanup(func() {
		if err := host.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	want := map[string]PluginState{"duplicate": PluginFailed, "invalid": PluginFailed, "dependent": PluginFailed}
	if got := host.Lifecycle(); !maps.Equal(got, want) {
		t.Fatalf("invalid branches: %v, want %v", got, want)
	}
}
