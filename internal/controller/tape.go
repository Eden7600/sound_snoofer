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

// Tape plays, pauses, stops or winds the recorder's loaded file. Like Record,
// it is explicit, never retried, and needs live writer ownership. It never
// stops or disturbs a recording.
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
	if state == "Recording" || (state == "Paused" && !r.TapePlaying()) {
		return fmt.Errorf("recording in progress")
	}
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
	switch command {
	case TapeStop:
		if !r.TapePlaying() {
			return fmt.Errorf("not playing")
		}
		return send("Recorder.stop", func(r model.RecorderSnapshot) bool { return r.State() == "Stopped" })
	case TapeRewind:
		return send("Recorder.rew", nil)
	case TapeForward:
		return send("Recorder.ff", nil)
	}
	if state == "Playing" {
		return send("Recorder.pause", func(r model.RecorderSnapshot) bool { return r.State() == "Paused" })
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
