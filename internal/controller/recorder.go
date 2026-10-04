package controller

import (
	"context"
	"fmt"
	"time"

	"sound-snoofer/internal/model"
)

type RecorderBackend interface {
	Recorder() (model.RecorderSnapshot, error)
	SetRecorder(string, int) error
}

func (c *Controller) readRecorder() model.RecorderSnapshot {
	b, ok := c.Backend.(RecorderBackend)
	if !ok {
		return model.RecorderSnapshot{Error: "recorder API unavailable"}
	}
	r, e := b.Recorder()
	if e != nil {
		r.Error = e.Error()
	}
	return r
}

// Record is one-shot; its caller must hold live writer ownership. It never
// enters the watch retry loop, including after an uncertain transport write.
func (c *Controller) Record(ctx context.Context, start, live bool) error {
	if !live {
		return fmt.Errorf("recorder commands require live mode")
	}
	if c.Config.Studio == nil || c.Config.Studio.Recording == nil {
		return fmt.Errorf("recording profile not configured")
	}
	b, ok := c.Backend.(RecorderBackend)
	if !ok {
		return fmt.Errorf("recorder API unavailable")
	}
	r := c.readRecorder()
	if r.State() == "Unknown" {
		return fmt.Errorf("recorder state unknown: %s", r.Error)
	}
	write := func(param string, value int, transport bool) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := b.SetRecorder(param, value); e != nil {
			return fmt.Errorf("%s outcome unknown; not retried: %w", param, e)
		}
		deadline := c.Clock.Now().Add(c.Config.Verify())
		for {
			r = c.readRecorder()
			if r.State() == "Unknown" {
				return fmt.Errorf("%s confirmation unknown; not retried: %s", param, r.Error)
			}
			matched := r.Values[param] == float32(value)
			if transport {
				if start {
					matched = r.State() == "Recording"
				} else {
					matched = r.State() == "Stopped"
				}
			} else if r.State() != "Stopped" {
				return fmt.Errorf("recorder became active during preparation")
			}
			if matched {
				return nil
			}
			if c.Clock.Now().After(deadline) || c.Clock.Now().Equal(deadline) {
				return fmt.Errorf("%s verification timed out; not retried", param)
			}
			if e := c.Clock.Wait(ctx, min(5*time.Millisecond, deadline.Sub(c.Clock.Now()))); e != nil {
				return e
			}
		}
	}
	if start {
		if i := c.Config.VoiceIntent(); i != nil && i.Recording != nil && i.Recording.ToVST {
			return fmt.Errorf("disable Recording to VST before recording")
		}
	}
	if !start {
		if r.State() == "Stopped" {
			return nil
		}
		return write("Recorder.stop", 1, true)
	}
	if r.State() == "Recording" && r.Ready() {
		c.RecorderPrepared = true
		return nil
	}
	if r.State() != "Stopped" {
		return fmt.Errorf("Start requires stopped recorder; current state %s", r.State())
	}
	checkPlan := func() error {
		p, e := c.Plan()
		if e != nil {
			return e
		}
		if p.Topology == nil || p.Topology.Recording == nil || p.Topology.Recording.Blocked != "" {
			return fmt.Errorf("recording routing unavailable")
		}
		if p.Topology.Recording.Sources == 0 {
			return fmt.Errorf("no eligible recording sources enabled")
		}
		if p.HasChanges() {
			return fmt.Errorf("routing transition pending; wait for verified routes")
		}
		return nil
	}
	if e := checkPlan(); e != nil {
		return e
	}
	for _, s := range model.RecorderSetup() {
		r = c.readRecorder()
		if r.State() != "Stopped" {
			return fmt.Errorf("recorder is no longer stopped")
		}
		if r.Values[s.Parameter] != float32(s.Value) {
			if e := write(s.Parameter, s.Value, false); e != nil {
				return e
			}
		}
	}
	if e := checkPlan(); e != nil {
		return e
	}
	r = c.readRecorder()
	if r.State() != "Stopped" || !r.Ready() {
		return fmt.Errorf("recorder preparation changed before Start")
	}
	c.RecorderPrepared = true
	return write("Recorder.record", 1, true)
}

// Protection only repairs playback sends while stopped. Active conflicts are
// reported by the planner and frozen, so external transport is never fought.
func (c *Controller) protectRecorder(ctx context.Context) error {
	b, ok := c.Backend.(RecorderBackend)
	if !ok {
		return nil
	}
	for _, p := range []string{"Recorder.B1", "Recorder.B2", "Recorder.B3"} {
		if i := c.Config.VoiceIntent(); p == "Recorder.B2" && i != nil && i.Recording != nil && i.Recording.ToVST {
			continue
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		r := c.readRecorder()
		if r.State() != "Stopped" {
			return nil
		}
		if r.Values[p] != 0 {
			if e := b.SetRecorder(p, 0); e != nil {
				return e
			}
			r = c.readRecorder()
			if r.State() == "Unknown" || r.Values[p] != 0 {
				return fmt.Errorf("tape playback protection unverified: %s", p)
			}
		}
	}
	return nil
}
