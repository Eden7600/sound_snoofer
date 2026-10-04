package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

type Backend interface {
	Snapshot() (model.Snapshot, error)
	Set(string, model.Device) error
}
type Clock interface {
	Now() time.Time
	Wait(context.Context, time.Duration) error
}
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
func (RealClock) Wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

var ErrPlanChanged = errors.New("routing changed before application; waiting for a fresh stable decision")

type Event struct {
	Kind    string        `json:"kind"`
	Message string        `json:"message,omitempty"`
	Plan    *routing.Plan `json:"plan,omitempty"`
}
type Controller struct {
	RecorderPrepared bool
	Backend          Backend
	Config           config.Config
	Clock            Clock
	Emit             func(Event)
	pending          string
	since            time.Time
	lastEvent        string
	retry            time.Duration
	retryWrite       bool
}

func (c *Controller) event(e Event) {
	b, _ := json.Marshal(e)
	key := string(b)
	if key == c.lastEvent {
		return
	}
	c.lastEvent = key
	if c.Emit != nil {
		c.Emit(e)
	}
}
func (c *Controller) Plan() (routing.Plan, error) {
	s, e := c.Backend.Snapshot()
	if e != nil {
		return routing.Plan{}, e
	}
	if c.Config.Studio != nil && c.Config.Studio.Recording != nil {
		r := c.readRecorder()
		s.Recorder = &r
	}
	return routing.Build(c.Config, s)
}

// Apply revalidates the complete plan before every write. No setter is called
// for unresolved slots, unconfigured slots, or already matching names.
func (c *Controller) Apply(ctx context.Context, expected routing.Plan) error {
	if expected.Topology != nil {
		return c.applyTopology(ctx, expected)
	}
	verified := 0
	fail := func(err error) error {
		return fmt.Errorf("application stopped (%d assignments verified): %w", verified, err)
	}
	for _, desired := range expected.Decisions {
		if e := ctx.Err(); e != nil {
			return fail(e)
		}
		fresh, e := c.Plan()
		if e != nil {
			return fail(e)
		}
		if fresh.Key() != expected.Key() {
			return fail(ErrPlanChanged)
		}
		var d routing.Decision
		for _, item := range fresh.Decisions {
			if item.Target == desired.Target {
				d = item
				break
			}
		}
		if !d.Change || d.Desired == nil {
			continue
		}
		c.event(Event{Kind: "submitting", Message: fmt.Sprintf("%s: %q -> %q", d.Target, d.Current, d.Desired.Name)})
		if e = c.Backend.Set(d.Target, *d.Desired); e != nil {
			return fail(e)
		}
		deadline := c.Clock.Now().Add(c.Config.Verify())
		for {
			if e = ctx.Err(); e != nil {
				return fail(e)
			}
			snapshot, e := c.Backend.Snapshot()
			if e != nil {
				return fail(e)
			}
			if snapshot.Edition != expected.Edition {
				return fail(ErrPlanChanged)
			}
			if snapshot.Assignments[d.Target] == d.Desired.Name {
				verified++
				c.event(Event{Kind: "verified", Message: fmt.Sprintf("%s now reports %q (device-name readback)", d.Target, d.Desired.Name)})
				break
			}
			remaining := deadline.Sub(c.Clock.Now())
			if remaining <= 0 {
				return fail(fmt.Errorf("%s: timed out verifying %q; last observed %q", d.Target, d.Desired.Name, snapshot.Assignments[d.Target]))
			}
			if e = c.Clock.Wait(ctx, min(100*time.Millisecond, remaining)); e != nil {
				return fail(e)
			}
		}
	}
	final, e := c.Plan()
	if e != nil {
		return fail(e)
	}
	if final.Key() != expected.Key() || final.HasChanges() {
		return fail(ErrPlanChanged)
	}
	if final.HasUnresolved() {
		return fail(errors.New("one or more slots have no eligible candidate; those assignments were left unchanged"))
	}
	return nil
}

// Step performs a single observation. Invalid snapshots discard pending state;
// callers wait its returned delay. Tests drive this without real sleeping.
func (c *Controller) Step(ctx context.Context, live bool) time.Duration {
	if live && c.RecorderPrepared && c.Config.Studio != nil && c.Config.Studio.Recording != nil {
		if e := c.protectRecorder(ctx); e != nil {
			c.event(Event{Kind: "error", Message: e.Error()})
		}
	}
	p, e := c.Plan()
	attempted := false
	delay := c.Config.Poll()
	if e == nil {
		key := p.Key()
		debounce := 100 * time.Millisecond
		if p.Topology != nil {
			for _, op := range p.Topology.Operations {
				if op.Change && op.Device != nil {
					debounce = c.Config.Debounce()
					break
				}
			}
		} else if p.HasChanges() {
			debounce = c.Config.Debounce()
		}
		if p.Topology != nil {
			key += p.Topology.InventoryKey
		}
		key += debounce.String()
		if key != c.pending {
			c.pending = key
			c.since = c.Clock.Now()
		}
		c.event(Event{Kind: "plan", Plan: &p})
		remaining := debounce - c.Clock.Now().Sub(c.since)
		if live && p.HasChanges() && remaining > 0 {
			delay = min(delay, remaining)
		}
		if live && p.HasChanges() && remaining <= 0 {
			attempted = true
			e = c.Apply(ctx, p)
			// A pass always starts a fresh debounce, including partial application.
			c.pending = ""
		}
	}
	if e != nil {
		c.retryWrite = attempted
		c.pending = ""
		c.event(Event{Kind: "error", Message: e.Error()})
		if c.retry == 0 {
			c.retry = time.Second
		} else {
			c.retry = min(c.retry*2, 30*time.Second)
		}
		return max(c.retry, c.Config.Poll())
	}
	// Successful observations alone must not reset repeated setter failures.
	if !c.retryWrite || !live || !p.HasChanges() || attempted {
		c.retry = 0
		c.retryWrite = false
	}
	return delay
}
func (c *Controller) Watch(ctx context.Context, live bool) error {
	for {
		if e := ctx.Err(); e != nil {
			return e
		}
		delay := c.Step(ctx, live)
		if e := c.Clock.Wait(ctx, delay); e != nil {
			return e
		}
	}
}
