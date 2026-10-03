package controller

import (
	"context"
	"testing"
	"time"
)

func TestRoutingDebounceDeadline(t *testing.T) {
	c, b := voiceController(t)
	clock := c.Clock.(*fakeClock)
	ctx := context.Background()
	if got := c.Step(ctx, true); got != 100*time.Millisecond {
		t.Fatal("initial deadline", got)
	}
	clock.now = clock.now.Add(99 * time.Millisecond)
	if got := c.Step(ctx, true); got != time.Millisecond || b.attempt != 0 {
		t.Fatal("applied early or missed deadline", got)
	}
	clock.now = clock.now.Add(time.Millisecond)
	if got := c.Step(ctx, true); got != c.Config.Poll() || b.attempt == 0 {
		t.Fatal("did not apply at 100 ms", got)
	}
}

func TestDeviceDebounceAndRoutingReset(t *testing.T) {
	ctx := context.Background()
	c, b := voiceController(t)
	clock := c.Clock.(*fakeClock)
	b.s.Assignments["input:3"] = ""
	if got := c.Step(ctx, true); got != c.Config.Poll() {
		t.Fatal(got)
	}
	clock.now = clock.now.Add(100 * time.Millisecond)
	c.Step(ctx, true)
	if b.attempt != 0 || len(b.writes) != 0 {
		t.Fatal("device change used routing debounce")
	}
	clock.now = clock.now.Add(c.Config.Debounce())
	c.Step(ctx, true)
	if b.s.Assignments["input:3"] != "webcam" {
		t.Fatal("device not applied")
	}
	c.Config.Intent = c.Config.VoiceIntent()
	c.Config.Intent.Monitor = "pre"
	if got := c.Step(ctx, true); got != 100*time.Millisecond {
		t.Fatal(got)
	}
	clock.now = clock.now.Add(50 * time.Millisecond)
	c.Config.Intent.Monitor = "post"
	if got := c.Step(ctx, true); got != 100*time.Millisecond {
		t.Fatal("changed goal did not reset debounce", got)
	}
	clock.now = clock.now.Add(50 * time.Millisecond)
	b.s.Devices[0].ID += " changed"
	if got := c.Step(ctx, true); got != 100*time.Millisecond {
		t.Fatal("inventory did not reset debounce", got)
	}
	if got := c.Step(ctx, false); got != c.Config.Poll() {
		t.Fatal("preview poll changed", got)
	}
}
