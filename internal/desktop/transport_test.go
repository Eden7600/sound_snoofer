package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"

	"sound-snoofer/internal/model"
	"sound-snoofer/internal/tui"
)

func TestLaunchModes(t *testing.T) {
	for _, tt := range []struct {
		args []string
		tray bool
	}{
		{nil, true}, {[]string{"--dry-run"}, true}, {[]string{"tray"}, true},
		{[]string{"tui"}, false}, {[]string{"watch"}, false}, {[]string{"plan"}, false},
		{[]string{"--help"}, false}, {[]string{"help"}, false},
	} {
		if got := IsTrayLaunch(tt.args); got != tt.tray {
			t.Errorf("%v: tray=%v", tt.args, got)
		}
	}
}
func TestFrameRoundTrip(t *testing.T) {
	initial := tui.State{Live: true, Connected: true, EditAck: 123, Revision: 7, Snapshot: model.Snapshot{Numbers: map[string]float32{"Strip[0].B2": 1}}}
	data, err := json.Marshal(frame{State: &initial, Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	var decoded frame
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Focus || !reflect.DeepEqual(*decoded.State, initial) {
		t.Fatalf("lost state: %+v", decoded)
	}
}
func TestActionsRetainRevisionAndBatch(t *testing.T) {
	// Batched edits have an internal Go type but exported fields on the wire.
	data := []byte(`{"Kind":3,"ID":123,"Revision":7,"Edits":[{"Row":"monitor","Value":"post"}]}`)
	actions := make(chan tui.Action, 1)
	err := readActions(context.Background(), bytes.NewReader(data), actions)
	if !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	a := <-actions
	if a.ID != 123 || a.Revision != 7 || len(a.Edits) != 1 || a.Edits[0].Value != "post" {
		t.Fatalf("lost action: %+v", a)
	}
}
func TestClosedOrStalledControls(t *testing.T) {
	updates := make(chan []byte, 1)
	for i := 0; i < 1000; i++ {
		offer(updates, []byte{byte(i % 256)})
	}
	if got := <-updates; got[0] != byte(999%256) {
		t.Fatal("did not retain latest state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := readActions(ctx, bytes.NewBufferString(`{"Revision":7}`), make(chan tui.Action)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	r.Close()
	updates <- []byte(`{}`)
	if err := writeFrames(context.Background(), w, updates); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	w.Close()
}
func TestTrayHealth(t *testing.T) {
	for _, tt := range []struct {
		s    tui.State
		want string
	}{
		{tui.State{}, "Preview · Connecting"},
		{tui.State{Live: true, Connected: true}, "Live · Connected"},
		{tui.State{Live: true, Error: "offline"}, "Live · Attention — open controls"},
		{tui.State{StateError: "bad choices"}, "Preview · Attention — open controls"},
	} {
		if got := statusText(tt.s); got != tt.want {
			t.Errorf("got %q want %q", got, tt.want)
		}
	}
}
