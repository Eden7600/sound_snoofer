package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"

	"sound-snoofer/snoofer"
)

func TestGenericPrivateFrames(t *testing.T) {
	state := ViewState{Controls: []snoofer.Control{{ID: "external.light", Revision: 9, Value: "On"}}, Plugins: map[string]string{"external": "Running"}}
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
