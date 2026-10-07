//go:build windows

package streamdeck

import (
	"context"
	"sync"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestTapHoldAndImmediateKeys(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	var mu sync.Mutex
	var got []string
	record := func(_ context.Context, r snoofer.Request) error {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, r.ID+":"+r.Operation)
		return nil
	}
	if err := registry.Publish("x", []snoofer.Control{
		{ID: "x.hold", Label: "Hold", Kind: "command", Operations: []string{"press", "hold"}, Available: true},
		{ID: "x.plain", Label: "Plain", Kind: "command", Operations: []string{"press"}, Available: true},
	}, record); err != nil {
		t.Fatal(err)
	}
	layout := Layout{Home: "p", Pages: []Page{{ID: "p", Name: "P"}}}
	layout.Pages[0].Keys[0] = Binding{Control: "x.hold"}
	layout.Pages[0].Keys[1] = Binding{Control: "x.plain"}
	frames := make(chan device.Frame, 1)
	events := make(chan device.Event, 16)
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return frames, events, done
	}
	instance, err := startWithSurface(ctx, snoofer.Services{Controls: registry}, snoofer.MarshalSettings(Settings{Layout: layout}), surface)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		stop, release := context.WithTimeout(context.Background(), time.Second)
		defer release()
		if err := instance.Stop(stop); err != nil {
			t.Error(err)
		}
	}()
	generation := (<-frames).Generation
	requests := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
	wait := func(what string, n int) []string {
		t.Helper()
		for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
			if r := requests(); len(r) >= n {
				return r
			}
		}
		t.Fatal("timed out waiting for", what, requests())
		return nil
	}
	key := func(n int, down bool, gen uint64) {
		events <- device.Event{Key: n, Encoder: -1, Press: down, Release: !down, Generation: gen}
	}

	// A plain key acts on key-down.
	key(1, true, generation)
	if r := wait("plain press", 1); r[0] != "x.plain:press" {
		t.Fatal(r)
	}
	key(1, false, generation)
	// A hold key waits: a quick tap sends press on release.
	key(0, true, generation)
	time.Sleep(100 * time.Millisecond)
	if r := requests(); len(r) != 1 {
		t.Fatal("hold key acted on key-down", r)
	}
	key(0, false, generation)
	if r := wait("tap", 2); r[1] != "x.hold:press" {
		t.Fatal(r)
	}
	// Held past the threshold it sends hold once; the release adds nothing,
	// even with a newer generation.
	key(0, true, generation)
	if r := wait("hold", 3); r[2] != "x.hold:hold" {
		t.Fatal(r)
	}
	key(0, false, generation+5)
	time.Sleep(200 * time.Millisecond)
	if r := requests(); len(r) != 3 {
		t.Fatal("release after hold sent more", r)
	}
	// A tap whose release arrives after a generation change still counts.
	key(0, true, generation)
	time.Sleep(50 * time.Millisecond)
	key(0, false, generation+9)
	if r := wait("tap across generations", 4); r[3] != "x.hold:press" {
		t.Fatal(r)
	}
}

func TestResetFocusKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	var mu sync.Mutex
	var got []string
	record := func(_ context.Context, r snoofer.Request) error {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, r.ID+":"+r.Operation)
		return nil
	}
	if err := registry.Publish("x", []snoofer.Control{
		{ID: "x.dial", Kind: "numeric", Operations: []string{"adjust", "press", "reset"}, Available: true},
		{ID: "x.focus", Kind: "numeric", Operations: []string{"adjust", "reset"}, Available: true},
		{ID: "x.other", Kind: "command", Operations: []string{"press"}, Available: true},
	}, record); err != nil {
		t.Fatal(err)
	}
	layout := Layout{Home: "p", Pages: []Page{{ID: "p", Name: "P"}}}
	layout.Pages[0].Keys[26] = Binding{Control: resetFocusID}
	frames := make(chan device.Frame, 1)
	events := make(chan device.Event, 16)
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return frames, events, done
	}
	instance, err := startWithSurface(ctx, snoofer.Services{Controls: registry}, snoofer.MarshalSettings(Settings{Layout: layout}), surface)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		stop, release := context.WithTimeout(context.Background(), time.Second)
		defer release()
		if err := instance.Stop(stop); err != nil {
			t.Error(err)
		}
	}()
	frame := <-frames
	if frame.Keys[26].Label != "Auto" || frame.Keys[26].Icon != "focus-reset" {
		t.Fatalf("reset key %+v", frame.Keys[26])
	}
	events <- device.Event{Key: 26, Encoder: -1, Press: true, Generation: frame.Generation}
	for deadline := time.Now().Add(2 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("reset not sent", got)
		}
	}
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "x.dial:reset" && got[1] != "x.dial:reset" || got[0] != "x.focus:reset" && got[1] != "x.focus:reset" {
		t.Fatal("reset sent to", got)
	}
}
