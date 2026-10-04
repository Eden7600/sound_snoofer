package controller

import (
	"context"
	"fmt"
	"time"
)

// PlaySnippet is an explicit, never-retried restart of the loaded native tape.
func (c *Controller) PlaySnippet(ctx context.Context, live bool) error {
	if !live {
		return fmt.Errorf("playback requires live mode")
	}
	i := c.Config.VoiceIntent()
	if i == nil || i.Recording == nil || !i.Recording.ToVST {
		return fmt.Errorf("enable Recording to VST first")
	}
	b, ok := c.Backend.(RecorderBackend)
	if !ok {
		return fmt.Errorf("recorder API unavailable")
	}
	p, err := c.Plan()
	if err != nil {
		return err
	}
	if p.HasChanges() || p.HasUnresolved() {
		return fmt.Errorf("wait for verified rehearsal routes")
	}
	r := c.readRecorder()
	if r.State() != "Stopped" {
		return fmt.Errorf("Play requires stopped recorder; current state %s", r.State())
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := b.SetRecorder("Recorder.replay", 1); err != nil {
		return fmt.Errorf("play outcome unknown; not retried: %w", err)
	}
	deadline := c.Clock.Now().Add(c.Config.Verify())
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		r = c.readRecorder()
		if r.State() == "Playing" {
			return nil
		}
		if r.State() == "Unknown" || !c.Clock.Now().Before(deadline) {
			return fmt.Errorf("play confirmation unknown; not retried")
		}
		if err := c.Clock.Wait(ctx, 5*time.Millisecond); err != nil {
			return err
		}
	}
}
