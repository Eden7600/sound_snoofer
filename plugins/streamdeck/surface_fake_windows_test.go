//go:build windows

package streamdeck

import (
	"context"

	device "sound-snoofer/internal/streamdeck"
)

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

func fakeSurface(frames chan device.Frame, events chan device.Event, done chan struct{}) func(context.Context) surface {
	return func(ctx context.Context) surface {
		go func() { <-ctx.Done(); close(done) }()
		return &testSurface{frames: frames, events: events, done: done}
	}
}
