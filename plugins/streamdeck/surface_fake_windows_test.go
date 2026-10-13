//go:build windows

package streamdeck

import (
	"context"

	device "sound-snoofer/internal/streamdeck"
)

// testDeck is the + XL that fakeSurface connects.
const testDeck = "TEST1"

// testSurface stands in for the HID surface: every deck's frames go to one
// channel, latest wins, and done closes when the plugin's context ends.
type testSurface struct {
	frames chan device.Frame
	events chan device.Event
	done   chan struct{}
}

func (s *testSurface) Show(_ string, f device.Frame) {
	select {
	case <-s.frames:
	default:
	}
	select {
	case s.frames <- f:
	default:
	}
}
func (s *testSurface) Events() <-chan device.Event { return s.events }
func (s *testSurface) Done() <-chan struct{}       { return s.done }

// fakeSurface connects testDeck before any other event.
func fakeSurface(frames chan device.Frame, events chan device.Event, done chan struct{}) func(context.Context) surface {
	return func(ctx context.Context) surface {
		connect := device.Event{Connected: true, Device: testDeck, Serial: testDeck, Model: "plus-xl"}
		if cap(events) > 0 {
			events <- connect
		} else {
			go func() { events <- connect }()
		}
		return idleSurface(frames, events, done)(ctx)
	}
}

// idleSurface connects no deck; the test sends every event.
func idleSurface(frames chan device.Frame, events chan device.Event, done chan struct{}) func(context.Context) surface {
	return func(ctx context.Context) surface {
		go func() { <-ctx.Done(); close(done) }()
		return &testSurface{frames: frames, events: events, done: done}
	}
}
