package snoofer

import (
	"encoding/json"
	"testing"
)

func TestViewDataSnapshotIsolation(t *testing.T) {
	registry := NewControls()
	data := json.RawMessage("{\"Dirty\":true}")
	if err := registry.Publish("test", []Control{{ID: "test.preview", ViewData: data}}, nil); err != nil {
		t.Fatal(err)
	}
	data[0] = '!'
	snapshot := registry.Snapshot()
	if snapshot[0].ViewData[0] != '{' {
		t.Fatal("publish retained mutable data")
	}
	snapshot[0].ViewData[0] = '!'
	if registry.Snapshot()[0].ViewData[0] != '{' {
		t.Fatal("snapshot exposed mutable data")
	}
}
