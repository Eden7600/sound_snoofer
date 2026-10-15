//go:build windows

package media

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func mediaRequest(t *testing.T, registry *snoofer.Controls) snoofer.Request {
	t.Helper()
	for _, c := range registry.Snapshot() {
		if c.ID == "media.play" {
			return snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
		}
	}
	t.Fatal("missing media command")
	return snoofer.Request{}
}

func mediaOutcome(t *testing.T, results <-chan snoofer.Outcome) snoofer.Outcome {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(time.Second):
		t.Fatal("missing command outcome")
		return snoofer.Outcome{}
	}
}

func TestMediaOutcomeWaitsForSendInput(t *testing.T) {
	for _, nativeErr := range []error{nil, errors.New("media dispatch incomplete: 1 of 2 inputs")} {
		t.Run(fmt.Sprint(nativeErr), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			registry := snoofer.NewControls()
			entered, release := make(chan string, 1), make(chan struct{})
			defer close(release)
			p := pluginWithSender(func(id string) error {
				entered <- id
				<-release
				return nativeErr
			})
			instance, err := p.Start(ctx, snoofer.Services{Controls: registry, Live: true}, []byte("{}"), nil)
			if err != nil {
				t.Fatal(err)
			}
			results := make(chan snoofer.Outcome, 2)
			if err := registry.DispatchWithResult(ctx, mediaRequest(t, registry), func(result snoofer.Outcome) { results <- result }); err != nil {
				t.Fatal(err)
			}
			select {
			case id := <-entered:
				if id != "media.play" {
					t.Fatal("wrong command sent", id)
				}
			case <-time.After(time.Second):
				t.Fatal("native call did not start")
			}
			select {
			case result := <-results:
				t.Fatal("queue acceptance claimed a completed native call", result)
			default:
			}
			release <- struct{}{}
			got := mediaOutcome(t, results)
			if nativeErr == nil && got.Status != snoofer.Processed {
				t.Fatal(got)
			}
			if nativeErr != nil && (got.Status != snoofer.Failed || got.Detail != nativeErr.Error()) {
				t.Fatal("native failure lost", got)
			}
			stopCtx, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			if err := instance.Stop(stopCtx); err != nil {
				t.Fatal(err)
			}
			select {
			case duplicate := <-results:
				t.Fatal("duplicate completion", duplicate)
			default:
			}
		})
	}
}

func TestMediaQueueRefusalAndStopSettleEveryClaim(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	entered, release := make(chan struct{}, 1), make(chan struct{})
	defer close(release)
	p := pluginWithSender(func(string) error {
		entered <- struct{}{}
		<-release
		return nil
	})
	instance, err := p.Start(ctx, snoofer.Services{Controls: registry, Live: true}, []byte("{}"), nil)
	if err != nil {
		t.Fatal(err)
	}
	request := mediaRequest(t, registry)
	results := make(chan snoofer.Outcome, 12)
	reply := func(result snoofer.Outcome) { results <- result }
	if err := registry.DispatchWithResult(ctx, request, reply); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("native call did not start")
	}
	for range 8 {
		if err := registry.DispatchWithResult(ctx, request, reply); err != nil {
			t.Fatal("queue rejected valid capacity", err)
		}
	}
	if err := registry.DispatchWithResult(ctx, request, reply); err == nil {
		t.Fatal("full queue accepted another command")
	}
	if got := mediaOutcome(t, results); got.Status != snoofer.Rejected {
		t.Fatal("queue refusal was not explicit", got)
	}
	cancel()
	for range 9 {
		if got := mediaOutcome(t, results); got.Status != snoofer.Unknown {
			t.Fatal("stop lost a claim", got)
		}
	}
	release <- struct{}{}
	stopCtx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := instance.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
	if len(entered) != 0 || len(results) != 0 {
		t.Fatal("cancelled work ran or completed twice")
	}
}
