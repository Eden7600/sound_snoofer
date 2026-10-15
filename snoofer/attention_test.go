package snoofer

import (
	"context"
	"encoding/json"
	"testing"
)

func TestAttentionIsPresentationOnly(t *testing.T) {
	registry := NewControls()
	calls := 0
	item := Control{ID: "test.setting", Value: "Off", Available: true, Operations: []string{"press"}}
	handler := func(context.Context, Request) error { calls++; return nil }
	if err := registry.Publish("test", []Control{item}, handler); err != nil {
		t.Fatal(err)
	}
	revision := registry.Snapshot()[0].Revision
	for _, attention := range []AttentionLevel{AttentionWarning, AttentionCritical, ""} {
		item.Attention = attention
		if err := registry.Publish("test", []Control{item}, handler); err != nil {
			t.Fatal(err)
		}
		current := registry.Snapshot()[0]
		if current.Attention != attention || current.Revision != revision {
			t.Fatalf("marker changed identity or was lost: %+v", current)
		}
		if err := registry.Dispatch(context.Background(), Request{ID: item.ID, Revision: revision, Operation: "press"}); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(current)
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(raw, &wire); err != nil {
			t.Fatal(err)
		}
		value, present := wire["Attention"]
		if present != (attention != "") || present && value != string(attention) {
			t.Fatalf("unexpected wire marker: %s", raw)
		}
	}
	if calls != 3 {
		t.Fatalf("metadata changes lost input: %d", calls)
	}
	item.Value = "On"
	if err := registry.Publish("test", []Control{item}, handler); err != nil {
		t.Fatal(err)
	}
	if err := registry.Dispatch(context.Background(), Request{ID: item.ID, Revision: revision, Operation: "press"}); err == nil {
		t.Fatal("an actual value change accepted stale input")
	}
}
