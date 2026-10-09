package controller

import (
	"context"
	"fmt"
	"slices"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// TapeCommand is a one-shot recorder playback command.
type TapeCommand int

const (
	TapePlayPause TapeCommand = iota
	TapeStop
	TapeRewind
	TapeForward
)

var tapeBuses = []string{"A1", "A2", "A3", "A4", "A5"}

// Tape runs the tape-deck transport by recorder state
// (openspec/changes/recorder-transport). Play/Pause plays, pauses the tape
// with one stop, or pauses and resumes a recording; Stop returns the tape to
// the start with a second stop, or ends a recording; Rewind and Fast-forward
// never act on a recording. Like Record, every write is explicit, never
// retried, and needs live writer ownership.
func (c *Controller) Tape(ctx context.Context, command TapeCommand, live bool) error {
	if !live {
		return fmt.Errorf("tape commands require live mode")
	}
	i := c.Config.VoiceIntent()
	if i == nil || i.Recording == nil {
		return fmt.Errorf("recording profile not configured")
	}
	b, ok := c.Backend.(RecorderBackend)
	if !ok {
		return fmt.Errorf("recorder API unavailable")
	}
	r := c.readRecorder()
	state := r.State()
	if state == "Unknown" {
		return fmt.Errorf("recorder state unknown: %s", r.Error)
	}
	recording := state == "Recording" || (state == "Paused" && !r.TapePlaying())
	// send writes one command and waits for want (nil: no readback exists).
	send := func(param string, want func(model.RecorderSnapshot) bool) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := b.SetRecorder(param, 1); e != nil {
			return fmt.Errorf("%s outcome unknown; not retried: %w", param, e)
		}
		if want == nil {
			return nil
		}
		return c.awaitRecorder(ctx, param, want)
	}
	stopped := func(r model.RecorderSnapshot) bool { return r.State() == "Stopped" }
	switch command {
	case TapeStop:
		switch {
		case recording:
			return c.Record(ctx, false, live)
		case r.TapePlaying():
			// The first stop pauses playback; the second returns to the start.
			if e := send("Recorder.stop", stopped); e != nil {
				return e
			}
			c.TapePaused = true
			if e := send("Recorder.stop", nil); e != nil {
				return e
			}
			c.TapePaused = false
			return nil
		case c.TapePaused:
			if e := send("Recorder.stop", nil); e != nil {
				return e
			}
			c.TapePaused = false
			return nil
		}
		return fmt.Errorf("already at start")
	case TapeRewind, TapeForward:
		if recording {
			return fmt.Errorf("recording in progress")
		}
		if command == TapeRewind {
			return send("Recorder.rew", nil)
		}
		return send("Recorder.ff", nil)
	}
	switch {
	case state == "Recording":
		return send("Recorder.pause", func(r model.RecorderSnapshot) bool {
			return r.State() == "Paused" && r.Values["Recorder.record"] != 0
		})
	case recording:
		return send("Recorder.record", func(r model.RecorderSnapshot) bool { return r.State() == "Recording" })
	case state == "Playing":
		// Recorder.pause does not work in playback; one stop pauses it.
		if e := send("Recorder.stop", stopped); e != nil {
			return e
		}
		c.TapePaused = true
		return nil
	case state == "Paused":
		// Playback paused elsewhere resumes where it stopped.
		return send("Recorder.play", func(r model.RecorderSnapshot) bool { return r.State() == "Playing" })
	}
	if i.Recording.ToVST {
		return fmt.Errorf("Recording to VST is on; use the rehearsal snippet")
	}
	if e := c.routeTape(ctx, b, r); e != nil {
		return e
	}
	if e := send("Recorder.play", func(r model.RecorderSnapshot) bool { return r.State() == "Playing" }); e != nil {
		return e
	}
	c.TapeListening = true
	c.TapePaused = false
	return nil
}

// routeTape sends the tape to the Playback destination and tape-enabled
// output slots before playback starts, so the first second is audible. The planner keeps it
// there while the tape plays and clears it afterwards.
func (c *Controller) routeTape(ctx context.Context, b RecorderBackend, r model.RecorderSnapshot) error {
	p, e := c.Plan()
	if e != nil {
		return e
	}
	if p.Topology == nil || p.Topology.PlaybackTarget == "" {
		return fmt.Errorf("no playback")
	}
	slotTape := p.Topology.OutputBuses(config.SourceTape)
	for _, bus := range tapeBuses {
		param := "Recorder." + bus
		want := 0
		if bus == p.Topology.PlaybackTarget || slices.Contains(slotTape, bus) {
			want = 1
		}
		if r.Values[param] == float32(want) {
			continue
		}
		if c.sendsPaused() {
			return fmt.Errorf("Sends paused: tape needs %s = %d", param, want)
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := b.SetRecorder(param, want); e != nil {
			return fmt.Errorf("%s outcome unknown; not retried: %w", param, e)
		}
		value := float32(want)
		if e := c.awaitRecorder(ctx, param, func(r model.RecorderSnapshot) bool { return r.Values[param] == value }); e != nil {
			return e
		}
	}
	return nil
}

// awaitRecorder polls the recorder until want holds or the verify window ends.
func (c *Controller) awaitRecorder(ctx context.Context, param string, want func(model.RecorderSnapshot) bool) error {
	deadline := c.Clock.Now().Add(c.Config.Verify())
	for {
		r := c.readRecorder()
		if r.State() == "Unknown" {
			return fmt.Errorf("%s confirmation unknown; not retried: %s", param, r.Error)
		}
		if want(r) {
			return nil
		}
		if !c.Clock.Now().Before(deadline) {
			return fmt.Errorf("%s verification timed out; not retried", param)
		}
		if e := c.Clock.Wait(ctx, min(5*time.Millisecond, deadline.Sub(c.Clock.Now()))); e != nil {
			return e
		}
	}
}
