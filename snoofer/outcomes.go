package snoofer

import (
	"context"
	"sync"
)

// OutcomeStatus names the established action boundary, not observed device state.
type OutcomeStatus string

const (
	Submitted OutcomeStatus = "Submitted"
	Processed OutcomeStatus = "Processed"
	Rejected  OutcomeStatus = "Rejected"
	Failed    OutcomeStatus = "Failed"
	Unknown   OutcomeStatus = "Unknown"
)

// Outcome describes one action. A processing error may follow a partial effect.
type Outcome struct {
	Status OutcomeStatus
	Detail string
}

// Completion settles a claimed action once. Callbacks must not block.
type Completion func(Outcome)

type requestCompletion struct {
	mu          sync.Mutex
	reply       Completion
	stop        func() bool
	early       *Outcome
	dispatching bool
	claimed     bool
	done        bool
}

// ClaimCompletion transfers result ownership to the provider before enqueueing.
// The provider must finish every accepted path. Cancelling its lifetime settles
// Unknown; normal completion detaches that cancellation hook. Untracked callers
// receive a no-op callback. Claim synchronously inside the Publish handler.
func (r Request) ClaimCompletion(ctx context.Context) Completion {
	c := r.completion
	if c == nil {
		return func(Outcome) {}
	}
	c.mu.Lock()
	if c.claimed || c.done {
		c.mu.Unlock()
		return c.finish
	}
	c.claimed = true
	c.mu.Unlock()
	stop := context.AfterFunc(ctx, func() {
		c.finish(Outcome{Status: Unknown, Detail: "Provider stopped before completion"})
	})
	c.mu.Lock()
	if c.done {
		c.mu.Unlock()
		stop()
	} else {
		c.stop = stop
		c.mu.Unlock()
	}
	return c.finish
}

// DispatchWithResult retains Dispatch's validation and enqueue error return.
// Unclaimed dispatch is only Submitted. Early completion is held until Dispatch
// releases its lock, and an enqueue refusal takes precedence over that completion.
func (c *Controls) DispatchWithResult(ctx context.Context, r Request, reply Completion) error {
	if reply == nil {
		return c.Dispatch(ctx, r)
	}
	pending := &requestCompletion{reply: reply, dispatching: true}
	r.completion = pending
	err := c.Dispatch(ctx, r)
	pending.dispatched(err)
	return err
}

func (c *requestCompletion) finish(outcome Outcome) {
	c.mu.Lock()
	if c.done {
		c.mu.Unlock()
		return
	}
	if c.dispatching {
		if c.early == nil {
			c.early = &outcome
		}
		c.mu.Unlock()
		return
	}
	deliver := c.settleLocked(outcome)
	c.mu.Unlock()
	deliver()
}

func (c *requestCompletion) dispatched(err error) {
	c.mu.Lock()
	c.dispatching = false
	result := c.early
	if err != nil {
		result = &Outcome{Status: Rejected, Detail: err.Error()}
	} else if result == nil && !c.claimed {
		result = &Outcome{Status: Submitted, Detail: "Submitted"}
	}
	c.early = nil
	if result == nil {
		c.mu.Unlock()
		return
	}
	deliver := c.settleLocked(*result)
	c.mu.Unlock()
	deliver()
}

// The returned function runs after both completion and registry locks are free.
func (c *requestCompletion) settleLocked(outcome Outcome) func() {
	c.done = true
	reply, stop := c.reply, c.stop
	c.reply, c.stop = nil, nil
	return func() {
		if stop != nil {
			stop()
		}
		reply(outcome)
	}
}
