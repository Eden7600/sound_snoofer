package aec

import (
	"context"
	"errors"
	"testing"

	engine "sound-snoofer/internal/aec"
	"sound-snoofer/snoofer"
)

type refusingEngine struct {
	fakeEngine
	refuse bool
}

func (e *refusingEngine) Configure(c engine.Config) error {
	if e.refuse {
		return errors.New("configure failed")
	}
	return e.fakeEngine.Configure(c)
}

func TestAECChangedTargetRejectsQueuedRetryEvenWhenConfigureFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mixer := &fakeMixer{targets: speakers}
	e := &refusingEngine{fakeEngine: fakeEngine{stats: engine.Stats{Failed: true}}}
	w := newWorker(Settings{}, mixer, &e.fakeEngine)
	w.open = func(string) (canceller, error) { return e, nil }
	w.step(ctx)
	results := queueAEC(t, ctx, w, "aec.retry", "")
	queued := <-w.requests
	e.refuse = true
	mixer.targets.Mic = [2]int{4, 5}
	w.step(ctx)
	w.process(ctx, queued)
	if got := aecOutcome(t, results); got.Status != snoofer.Rejected || w.retryRequested {
		t.Fatal("queued retry crossed a changed native target", got)
	}
}
