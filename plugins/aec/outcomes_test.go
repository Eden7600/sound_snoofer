package aec

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func queueAEC(t *testing.T, ctx context.Context, w *worker, id, value string) <-chan snoofer.Outcome {
	t.Helper()
	w.publish(ctx)
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID != id {
			continue
		}
		operation := "set"
		if id == "aec.retry" {
			operation = "press"
		}
		results := make(chan snoofer.Outcome, 2)
		if err := w.services.Controls.DispatchWithResult(ctx, snoofer.Request{
			ID: id, Revision: c.Revision, Operation: operation, Value: value,
		}, func(result snoofer.Outcome) { results <- result }); err != nil {
			t.Fatal(err)
		}
		return results
	}
	t.Fatal("missing AEC control", id)
	return nil
}

func aecOutcome(t *testing.T, results <-chan snoofer.Outcome) snoofer.Outcome {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(time.Second):
		t.Fatal("missing AEC outcome")
		return snoofer.Outcome{}
	}
}

func TestAECWaitsForSettingsSave(t *testing.T) {
	for _, saveErr := range []error{nil, errors.New("settings disk unavailable")} {
		w := newWorker(Settings{}, &fakeMixer{targets: speakers}, &fakeEngine{})
		saves := 0
		w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { saves++; return saveErr }
		ctx, cancel := context.WithCancel(context.Background())
		results := queueAEC(t, ctx, w, "aec.mode", "off")
		select {
		case result := <-results:
			t.Fatal("queued settings reported an outcome before save", result)
		default:
		}
		if saves != 0 {
			t.Fatal("save happened in registry handler")
		}
		w.process(ctx, <-w.requests)
		got := aecOutcome(t, results)
		cancel()
		if saves != 1 {
			t.Fatal("settings save count", saves)
		}
		if saveErr == nil && (got.Status != snoofer.Processed || w.settings.mode() != "off") {
			t.Fatal(got, w.settings)
		}
		if saveErr != nil && (got.Status != snoofer.Failed || got.Detail != saveErr.Error() || w.settings.mode() == "off") {
			t.Fatal("save failure lost or unsaved setting applied", got, w.settings)
		}
	}
}

func TestAECQueuedSettingsDoNotRebaseAfterABA(t *testing.T) {
	w := newWorker(Settings{}, &fakeMixer{targets: speakers}, &fakeEngine{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := queueAEC(t, ctx, w, "aec.strength", "gentle")
	queued := <-w.requests
	original := w.settings.mode()
	w.handle(snoofer.Request{ID: "aec.mode", Value: "off"})
	w.handle(snoofer.Request{ID: "aec.mode", Value: original})
	w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { t.Fatal("stale edit reached save"); return nil }
	w.process(ctx, queued)
	if got := aecOutcome(t, results); got.Status != snoofer.Rejected {
		t.Fatal(got)
	}
}

func TestAECRetryIsSubmissionAndRechecksContext(t *testing.T) {
	for _, changed := range []bool{false, true} {
		w := newWorker(Settings{}, &fakeMixer{targets: speakers}, &fakeEngine{})
		w.active, w.engineErr = true, "engine missing"
		ctx, cancel := context.WithCancel(context.Background())
		results := queueAEC(t, ctx, w, "aec.retry", "")
		queued := <-w.requests
		if changed {
			w.active = false
		}
		w.process(ctx, queued)
		got := aecOutcome(t, results)
		cancel()
		if changed {
			if got.Status != snoofer.Rejected || w.retryRequested {
				t.Fatal("unavailable retry accepted", got)
			}
		} else if got.Status != snoofer.Submitted || !w.retryRequested || w.engine != nil {
			t.Fatal("retry claimed engine processing", got)
		}
	}
}

func TestAECEngineResetInvalidatesQueuedRetry(t *testing.T) {
	w := newWorker(Settings{}, &fakeMixer{targets: speakers}, &fakeEngine{})
	w.active, w.engineErr = true, "engine missing"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := queueAEC(t, ctx, w, "aec.retry", "")
	queued := <-w.requests
	w.retry(0, time.Now())
	w.engineErr = "engine missing again"
	w.process(ctx, queued)
	if got := aecOutcome(t, results); got.Status != snoofer.Rejected || w.retryRequested {
		t.Fatal(got)
	}
}

func TestAECQueueFullAndStopSettleClaims(t *testing.T) {
	w := newWorker(Settings{}, &fakeMixer{targets: speakers}, &fakeEngine{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := make([]<-chan snoofer.Outcome, 0, 8)
	for range 8 {
		results = append(results, queueAEC(t, ctx, w, "aec.mode", "off"))
	}
	c := control(t, w, "aec.mode")
	rejected := make(chan snoofer.Outcome, 1)
	if err := w.services.Controls.DispatchWithResult(ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "set", Value: "off"}, func(result snoofer.Outcome) { rejected <- result }); err == nil {
		t.Fatal("full AEC queue accepted a request")
	}
	if got := aecOutcome(t, rejected); got.Status != snoofer.Rejected {
		t.Fatal(got)
	}
	cancel()
	w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { t.Fatal("cancelled edit saved"); return nil }
	for _, result := range results {
		if got := aecOutcome(t, result); got.Status != snoofer.Unknown {
			t.Fatal(got)
		}
		w.process(ctx, <-w.requests)
		select {
		case duplicate := <-result:
			t.Fatal("duplicate result", duplicate)
		default:
		}
	}
}
