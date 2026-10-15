package snoofer

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func outcomeControl(t *testing.T, handler func(context.Context, Request) error) (*Controls, Request) {
	t.Helper()
	registry := NewControls()
	if err := registry.Publish("test", []Control{{ID: "test.toggle", Available: true, Operations: []string{"press"}}}, handler); err != nil {
		t.Fatal(err)
	}
	return registry, Request{ID: "test.toggle", Revision: registry.Snapshot()[0].Revision, Operation: "press"}
}

func receiveOutcome(t *testing.T, results <-chan Outcome) Outcome {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(time.Second):
		t.Fatal("missing action result")
		return Outcome{}
	}
}

func TestDispatchSubmissionAndDelayedCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var complete Completion
	tracked := false
	registry, request := outcomeControl(t, func(_ context.Context, r Request) error {
		if tracked {
			complete = r.ClaimCompletion(ctx)
		}
		return nil
	})
	results := make(chan Outcome, 4)
	reply := func(outcome Outcome) { results <- outcome }
	if err := registry.DispatchWithResult(ctx, request, reply); err != nil {
		t.Fatal(err)
	}
	if got := receiveOutcome(t, results); got.Status != Submitted {
		t.Fatal("enqueue claimed processing", got)
	}
	tracked = true
	if err := registry.DispatchWithResult(ctx, request, reply); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-results:
		t.Fatal("claimed result returned before processing", got)
	default:
	}
	complete(Outcome{Status: Failed, Detail: "save refused"})
	complete(Outcome{Status: Processed})
	cancel()
	if got := receiveOutcome(t, results); got.Status != Failed || got.Detail != "save refused" {
		t.Fatal("delayed failure lost", got)
	}
	select {
	case got := <-results:
		t.Fatal("completion was delivered twice", got)
	default:
	}
}

func TestDispatchCompletionOutsideRegistryLock(t *testing.T) {
	registry, request := outcomeControl(t, func(ctx context.Context, r Request) error {
		r.ClaimCompletion(ctx)(Outcome{Status: Processed})
		return nil
	})
	results := make(chan Outcome, 1)
	done := make(chan error, 1)
	go func() {
		done <- registry.DispatchWithResult(context.Background(), request, func(outcome Outcome) {
			registry.Snapshot() // Reentry would deadlock if the callback ran in invoke.
			results <- outcome
		})
	}()
	if got := receiveOutcome(t, results); got.Status != Processed {
		t.Fatal(got)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestDispatchEnqueueErrorWinsEarlyCompletion(t *testing.T) {
	queueFull := errors.New("provider queue full")
	registry, request := outcomeControl(t, func(ctx context.Context, r Request) error {
		r.ClaimCompletion(ctx)(Outcome{Status: Unknown, Detail: "cancelled concurrently"})
		return queueFull
	})
	results := make(chan Outcome, 1)
	if err := registry.DispatchWithResult(context.Background(), request, func(result Outcome) { results <- result }); !errors.Is(err, queueFull) {
		t.Fatal(err)
	}
	if got := receiveOutcome(t, results); got.Status != Rejected || got.Detail != queueFull.Error() {
		t.Fatal("enqueue refusal was hidden", got)
	}
}

func TestDispatchValidationAndOrdinaryCallers(t *testing.T) {
	calls := 0
	registry, request := outcomeControl(t, func(ctx context.Context, r Request) error {
		calls++
		r.ClaimCompletion(ctx)(Outcome{Status: Processed})
		data, err := json.Marshal(r)
		if err != nil || strings.Contains(string(data), "completion") {
			t.Fatalf("completion leaked onto wire: %s, %v", data, err)
		}
		return nil
	})
	if err := registry.Dispatch(context.Background(), request); err != nil || calls != 1 {
		t.Fatal("ordinary caller changed", err, calls)
	}
	for _, invalid := range []Request{
		{ID: request.ID, Revision: request.Revision - 1, Operation: "press"},
		{ID: request.ID, Revision: request.Revision, Operation: "set"},
		{ID: "missing", Revision: request.Revision, Operation: "press"},
	} {
		results := make(chan Outcome, 1)
		if err := registry.DispatchWithResult(context.Background(), invalid, func(result Outcome) { results <- result }); err == nil {
			t.Fatal("invalid request accepted", invalid)
		}
		if got := receiveOutcome(t, results); got.Status != Rejected || got.Detail == "" {
			t.Fatal("validation result missing", got)
		}
	}
	if calls != 1 {
		t.Fatal("invalid request reached provider", calls)
	}
}

func TestClaimedCompletionSettlesWhenProviderStops(t *testing.T) {
	providerCtx, stop := context.WithCancel(context.Background())
	defer stop()
	var complete Completion
	registry, request := outcomeControl(t, func(_ context.Context, r Request) error {
		complete = r.ClaimCompletion(providerCtx)
		return nil
	})
	results := make(chan Outcome, 2)
	if err := registry.DispatchWithResult(context.Background(), request, func(result Outcome) { results <- result }); err != nil {
		t.Fatal(err)
	}
	stop()
	if got := receiveOutcome(t, results); got.Status != Unknown {
		t.Fatal("provider stop claimed a known failure", got)
	}
	complete(Outcome{Status: Processed, Detail: "late operation"})
	select {
	case got := <-results:
		t.Fatal("late provider callback changed the result", got)
	default:
	}
}

// Context.AfterFunc supports contexts that own their cancellation registrations.
// Counting live registrations verifies completed actions do not accumulate until
// a long-lived provider stops.
type registrationContext struct {
	context.Context
	done   chan struct{}
	active atomic.Int32
}

func (c *registrationContext) Done() <-chan struct{} { return c.done }
func (c *registrationContext) AfterFunc(func()) func() bool {
	c.active.Add(1)
	var once sync.Once
	return func() bool {
		stopped := false
		once.Do(func() {
			c.active.Add(-1)
			stopped = true
		})
		return stopped
	}
}

func TestCompletionDetachesProviderCancellation(t *testing.T) {
	ctx := &registrationContext{Context: context.Background(), done: make(chan struct{})}
	var complete Completion
	var enqueueErr error
	registry, request := outcomeControl(t, func(_ context.Context, r Request) error {
		complete = r.ClaimCompletion(ctx)
		return enqueueErr
	})
	results := make(chan Outcome, 1)
	reply := func(result Outcome) { results <- result }
	if err := registry.DispatchWithResult(context.Background(), request, reply); err != nil {
		t.Fatal(err)
	}
	if ctx.active.Load() != 1 {
		t.Fatal("provider lifetime was not registered")
	}
	complete(Outcome{Status: Processed})
	receiveOutcome(t, results)
	if ctx.active.Load() != 0 {
		t.Fatal("completed action retained its cancellation hook")
	}
	enqueueErr = errors.New("queue full")
	if err := registry.DispatchWithResult(context.Background(), request, reply); !errors.Is(err, enqueueErr) {
		t.Fatal(err)
	}
	receiveOutcome(t, results)
	if ctx.active.Load() != 0 {
		t.Fatal("refused action retained its cancellation hook")
	}
}
