package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestStartupSecondQueueDoesNotClaimCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	actions := make(chan UIAction, 1)
	c := startupControl(startupState{}, nil, "Snoofer", true)
	if err := registry.Publish("core", []snoofer.Control{c}, coreHandler(ctx, actions, startupState{}, true)); err != nil {
		t.Fatal(err)
	}
	c = registry.Snapshot()[0]
	results := make(chan snoofer.Outcome, 2)
	if err := registry.DispatchWithResult(ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}, func(result snoofer.Outcome) { results <- result }); err != nil {
		t.Fatal(err)
	}
	action := <-actions
	select {
	case result := <-results:
		t.Fatalf("second enqueue claimed processing: %+v", result)
	default:
	}
	action.reply(applyStartup(ctx, action, true, "Snoofer", func() (startupState, error) { return startupState{}, nil }, func(bool, string) error { return errors.New("registration denied") }))
	if got := <-results; got.Status != snoofer.Failed || got.Detail != "registration denied" {
		t.Fatal(got)
	}
}

func TestStartupChangedTargetDoesNotWrite(t *testing.T) {
	writes := 0
	result := applyStartup(context.Background(), UIAction{Enable: true}, true, "Snoofer", func() (startupState, error) { return startupState{Command: "Other copy"}, nil }, func(bool, string) error { writes++; return nil })
	if result.Status != snoofer.Rejected || writes != 0 {
		t.Fatal(result, writes)
	}
}

func TestCoreQueueRefusalAndStopSettleClaims(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	actions := make(chan UIAction, 1)
	registry := snoofer.NewControls()
	c := startupControl(startupState{}, nil, "Snoofer", true)
	if err := registry.Publish("core", []snoofer.Control{c}, coreHandler(ctx, actions, startupState{}, true)); err != nil {
		t.Fatal(err)
	}
	c = registry.Snapshot()[0]
	request := snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
	first, second := make(chan snoofer.Outcome, 2), make(chan snoofer.Outcome, 2)
	if err := registry.DispatchWithResult(ctx, request, func(r snoofer.Outcome) { first <- r }); err != nil {
		t.Fatal(err)
	}
	if err := registry.DispatchWithResult(ctx, request, func(r snoofer.Outcome) { second <- r }); err == nil {
		t.Fatal("full core queue accepted action")
	}
	if result := <-second; result.Status != snoofer.Rejected {
		t.Fatal(result)
	}
	cancel()
	select {
	case result := <-first:
		if result.Status != snoofer.Unknown {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("stop lost startup claim")
	}
	action := <-actions
	writes := 0
	action.reply(applyStartup(ctx, action, true, "Snoofer", func() (startupState, error) { return startupState{}, nil }, func(bool, string) error { writes++; return nil }))
	if writes != 0 {
		t.Fatal("startup ran after stop")
	}
	for _, results := range []chan snoofer.Outcome{first, second} {
		select {
		case result := <-results:
			t.Fatalf("duplicate result: %+v", result)
		default:
		}
	}
}
