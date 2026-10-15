//go:build windows

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func publishedFrame(t *testing.T, c *controls, state ViewState) map[string]json.RawMessage {
	t.Helper()
	if err := c.publish(state); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-c.updates:
		var result map[string]json.RawMessage
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		return result
	default:
		t.Fatal("expected a changed frame")
		return nil
	}
}

func largeArtworkState() ViewState {
	state := ViewState{}
	for i := range 60 {
		state.Controls = append(state.Controls, snoofer.Control{ID: fmt.Sprintf("clip-%d", i), Revision: 1, Artwork: strings.Repeat("A", 64*1024)})
	}
	return state
}

func transportPayload(t testing.TB, sender *viewSender, desktop *Desktop, state ViewState, since uint64) ([]byte, []byte, uint64) {
	t.Helper()
	pipe, err := sender.prepare(state, false)
	if err != nil {
		t.Fatal(err)
	}
	if pipe != nil {
		desktop.applyFrame(decodeFrame(t, pipe))
	}
	update := desktop.State(since)
	ipc, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	if update != nil {
		since = update.Version
	}
	return pipe, ipc, since
}

func TestViewTransportPayload(t *testing.T) {
	state := largeArtworkState()
	legacyPipe, err := json.Marshal(frame{State: &state})
	if err != nil {
		t.Fatal(err)
	}
	legacyIPC, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var sender viewSender
	desktop := &Desktop{}
	pipe, ipc, cursor := transportPayload(t, &sender, desktop, state, 0)
	t.Logf("full-state baseline: pipe=%d IPC=%d bytes; separated initial: pipe=%d IPC=%d bytes", len(legacyPipe), len(legacyIPC), len(pipe), len(ipc))
	pipe, ipc, cursor = transportPayload(t, &sender, desktop, state, cursor)
	if len(pipe) != 0 || string(ipc) != "null" {
		t.Fatal("unchanged publication transferred a snapshot")
	}
	state.Controls[0].Progress.PositionMs++
	pipe, ipc, cursor = transportPayload(t, &sender, desktop, state, cursor)
	if bytes.Contains(pipe, []byte(state.Controls[0].Artwork)) || bytes.Contains(ipc, []byte(state.Controls[0].Artwork)) {
		t.Fatal("telemetry repeated artwork bytes")
	}
	t.Logf("unchanged: pipe=0 IPC=4 bytes; telemetry: pipe=%d IPC=%d bytes", len(pipe), len(ipc))
	state.Controls[0].Artwork = strings.Repeat("B", 64*1024)
	pipe, ipc, _ = transportPayload(t, &sender, desktop, state, cursor)
	if !bytes.Contains(pipe, []byte(state.Controls[0].Artwork)) || !bytes.Contains(ipc, []byte(state.Controls[0].Artwork)) {
		t.Fatal("replacement artwork was not delivered")
	}
	t.Logf("artwork replacement: pipe=%d IPC=%d bytes", len(pipe), len(ipc))
}

func BenchmarkViewTransport(b *testing.B) {
	for _, mode := range []string{"full-state-baseline", "initial", "unchanged", "telemetry", "artwork"} {
		b.Run(mode, func(b *testing.B) {
			state := largeArtworkState()
			var sender viewSender
			desktop := &Desktop{}
			_, _, cursor := transportPayload(b, &sender, desktop, state, 0)
			var pipeBytes, ipcBytes int
			firstArt, secondArt := state.Controls[0].Artwork, strings.Repeat("B", 64*1024)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if mode == "initial" {
					sender, desktop, cursor = viewSender{}, &Desktop{}, 0
				}
				if mode == "telemetry" {
					state.Controls[0].Progress.PositionMs++
				}
				if mode == "artwork" {
					state.Controls[0].Artwork = secondArt
					if i%2 != 0 {
						state.Controls[0].Artwork = firstArt
					}
				}
				if mode == "full-state-baseline" {
					pipe, err := json.Marshal(frame{State: &state})
					if err != nil {
						b.Fatal(err)
					}
					decoded := decodeFrame(b, pipe)
					ipc, err := json.Marshal(decoded.State)
					if err != nil {
						b.Fatal(err)
					}
					pipeBytes += len(pipe)
					ipcBytes += len(ipc)
					continue
				}
				pipe, ipc, next := transportPayload(b, &sender, desktop, state, cursor)
				cursor = next
				pipeBytes += len(pipe)
				ipcBytes += len(ipc)
			}
			b.StopTimer()
			b.ReportMetric(float64(pipeBytes)/float64(b.N), "pipe-B/op")
			b.ReportMetric(float64(ipcBytes)/float64(b.N), "IPC-B/op")
		})
	}
}

func TestControlsSuppressUnchanged(t *testing.T) {
	c := &controls{updates: make(chan []byte, 1)}
	state := ViewState{Controls: []snoofer.Control{{ID: "clip", Revision: 1, Artwork: "art"}}}
	publishedFrame(t, c, state)
	if err := c.publish(state); err != nil {
		t.Fatal(err)
	}
	if len(c.updates) != 0 {
		t.Fatal("unchanged state was queued again")
	}
}

func TestControlsArtworkSeparated(t *testing.T) {
	c := &controls{updates: make(chan []byte, 1)}
	state := ViewState{Controls: []snoofer.Control{{ID: "clip", Artwork: "art"}}}
	f := publishedFrame(t, c, state)
	var stripped ViewState
	if err := json.Unmarshal(f["State"], &stripped); err != nil {
		t.Fatal(err)
	}
	if stripped.Controls[0].Artwork != "" || string(f["Artwork"]) != `{"clip":"art"}` {
		t.Fatal("still artwork was not separated from polled state")
	}
}

func TestControlsCompleteAnimationChanges(t *testing.T) {
	c := &controls{updates: make(chan []byte, 1)}
	state := ViewState{Controls: []snoofer.Control{{ID: "clip", Artwork: "a", Animation: []snoofer.ArtworkFrame{{Artwork: "a", Delay: 1}, {Artwork: "b", Delay: 1}}}}}
	publishedFrame(t, c, state)
	state.Controls[0].Animation = []snoofer.ArtworkFrame{{Artwork: "a", Delay: 1}, {Artwork: "b", Delay: 2}}
	if f := publishedFrame(t, c, state); f["Animations"] == nil {
		t.Fatal("changed later animation delay was omitted")
	}
	state.Controls[0].Animation = nil
	if f := publishedFrame(t, c, state); string(f["Animations"]) != "{}" {
		t.Fatal("last animation removal did not encode an empty map")
	}
}
