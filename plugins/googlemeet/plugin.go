// Package googlemeet applies shared meeting intent to observed browser calls.
package googlemeet

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"sound-snoofer/internal/browserbridge"
	"sound-snoofer/snoofer"
)

const (
	freshness         = 3 * time.Second
	completionTimeout = 5 * time.Second
)

type lease interface {
	Updates() <-chan browserbridge.MeetUpdate
	Generation(string) uint64
	Send(context.Context, string, browserbridge.MeetCommand) (<-chan browserbridge.MeetResult, error)
	Status() browserbridge.Status
	Close(context.Context) error
}

type meetingAdapter interface {
	Report([]snoofer.MeetingPeer) error
	Changed() <-chan struct{}
	Next() (snoofer.MeetingCommand, bool)
	Close()
}

func validate(raw json.RawMessage) error { return snoofer.DecodeSettings(raw, &struct{}{}) }

// Plugin has no audio, Discord or Now Playing dependency and no saved activation state.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "googlemeet", Label: "Google Meet", Defaults: json.RawMessage(`{}`), Validate: validate,
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
			if err := validate(raw); err != nil {
				return nil, err
			}
			if s.Browser == nil {
				return nil, errors.New("browser bridge unavailable")
			}
			if s.Meetings == nil {
				return nil, errors.New("meeting coordinator unavailable")
			}
			runCtx, cancel := context.WithCancel(ctx)
			adapter, err := s.Meetings.Attach(runCtx, "googlemeet")
			if err != nil {
				cancel()
				return nil, err
			}
			connection, err := s.Browser.AcquireMeet(runCtx)
			if err != nil {
				cancel()
				adapter.Close()
				return nil, err
			}
			return start(runCtx, cancel, newWorker(s, connection, adapter)), nil
		}}
}

type instance struct {
	cancel     context.CancelFunc
	done       chan struct{}
	connection lease
}

func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	return errors.Join(i.connection.Close(ctx), snoofer.CancelAndWait(ctx, i.cancel, i.done))
}

func start(ctx context.Context, cancel context.CancelFunc, w *worker) *instance {
	i := &instance{cancel: cancel, done: make(chan struct{}), connection: w.connection}
	go func() {
		defer close(i.done)
		defer w.services.Controls.Remove("googlemeet")
		w.run(ctx)
	}()
	return i
}

type command struct {
	request                        snoofer.Request
	complete                       snoofer.Completion
	target                         target
	selection, catalogue, revision uint64
	property                       snoofer.MeetingProperty
	want                           bool
	accepted                       time.Time
}

type pending struct {
	command snoofer.MeetingCommand
	target  target
	control string
	wire    browserbridge.MeetCommand
	cancel  context.CancelFunc
}

type completed struct {
	id     uint64
	result browserbridge.MeetResult
}

type worker struct {
	services             snoofer.Services
	connection           lease
	meetings             meetingAdapter
	requests             chan command
	results              chan completed
	helpers              sync.WaitGroup
	browsers             map[string]browserbridge.MeetUpdate
	calls                map[tabKey]call
	selected             target
	discovered           bool
	selection, catalogue uint64
	choices              []target
	lastReport           []snoofer.MeetingPeer
	nextRequest          uint64
	pending              map[uint64]pending
	notes                map[actionKey]string
	link                 snoofer.ConnectionTracker
}

func newWorker(s snoofer.Services, connection lease, meetings meetingAdapter) *worker {
	return &worker{services: s, connection: connection, meetings: meetings, requests: make(chan command, 8), results: make(chan completed, 8),
		browsers: map[string]browserbridge.MeetUpdate{}, calls: map[tabKey]call{}, selection: 1,
		pending: map[uint64]pending{}, notes: map[actionKey]string{}}
}

func (w *worker) run(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	defer w.helpers.Wait()
	defer w.meetings.Close()
	updates := w.connection.Updates()
	for {
		w.reconcile(time.Now())
		w.synchronize(ctx, time.Now())
		w.publish(ctx, time.Now())
		select {
		case <-ctx.Done():
			for _, p := range w.pending {
				p.cancel()
				p.command.Complete(snoofer.Outcome{Status: snoofer.Unknown, Detail: "Meet stopped before confirmation"})
			}
			return
		case update, ok := <-updates:
			if !ok {
				updates = nil
				w.browsers = map[string]browserbridge.MeetUpdate{}
				w.calls = map[tabKey]call{}
				continue
			}
			w.receive(update)
		case c := <-w.requests:
			w.handle(ctx, c, time.Now())
		case result := <-w.results:
			w.finish(result, time.Now())
		case <-w.meetings.Changed():
		case <-ticker.C:
		}
	}
}

func (c command) finish(status snoofer.OutcomeStatus, detail string) {
	if c.complete != nil {
		c.complete(snoofer.Outcome{Status: status, Detail: detail})
	}
}

func (w *worker) handle(ctx context.Context, c command, now time.Time) {
	w.reconcile(now)
	reject := func(detail string) { c.finish(snoofer.Rejected, detail) }
	if ctx.Err() != nil {
		c.finish(snoofer.Unknown, "Meet stopped before processing")
		return
	}
	if c.request.ID == "googlemeet.target" {
		if c.selection != w.selection || c.catalogue != w.catalogue || !w.eligible(c.target, now) {
			reject("Meet calls changed; select again")
			return
		}
		w.selectTarget(c.target)
		c.finish(snoofer.Processed, "Meet call selected")
		return
	}
	if !w.services.Live {
		reject("Meet controls unavailable in preview")
		return
	}
	if !now.Before(c.accepted.Add(completionTimeout)) {
		reject("Meet request expired before sending")
		return
	}
	if err := w.services.Meetings.Request(ctx, c.property, c.revision, c.want, c.complete); err != nil {
		reject(err.Error())
	}
}

func (w *worker) apply(ctx context.Context, c snoofer.MeetingCommand, now time.Time) {
	reject := func(detail string) { c.Complete(snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}) }
	if ctx.Err() != nil || c.Context.Err() != nil || !w.services.Live {
		reject("Meet command canceled or unavailable in preview")
		return
	}
	var captured target
	for _, choice := range w.choices {
		if choice.peer() == c.Peer && choice.value() == c.Lifetime {
			captured = choice
			break
		}
	}
	current, ok := w.current(captured, now)
	if !ok {
		reject("Meet call changed or unavailable")
		return
	}
	control := ""
	switch c.Property {
	case snoofer.MeetingMute:
		control = "mic"
	case snoofer.MeetingCamera:
		control = "camera"
	default:
		reject("Unknown Meet property")
		return
	}
	state := controlState(current.state, control)
	if !state.Available || state.Identity == 0 || state.Value == nil || state.Revision != c.Revision {
		reject("Meet control changed or unavailable; try again")
		return
	}
	key := actionKey{captured, control}
	for _, p := range w.pending {
		if (actionKey{p.target, p.control}) == key {
			reject("Meet control pending")
			return
		}
	}
	if len(w.pending) >= 8 {
		reject("Meet commands busy")
		return
	}
	if *state.Value == c.Value {
		c.Complete(snoofer.Outcome{Status: snoofer.Processed, Detail: "Meet state observed"})
		return
	}
	wire := browserbridge.MeetCommand{Generation: captured.generation, Tab: captured.tab, Instance: captured.instance, Call: captured.call,
		Control: control, Revision: c.Revision, Value: c.Value}
	requestCtx, cancelRequest := context.WithTimeout(c.Context, completionTimeout)
	stop := context.AfterFunc(ctx, cancelRequest)
	cancel := func() { stop(); cancelRequest() }
	results, err := w.connection.Send(requestCtx, captured.browser, wire)
	if err != nil {
		cancel()
		reject(err.Error())
		return
	}
	w.nextRequest++
	id := w.nextRequest
	w.pending[id] = pending{command: c, target: captured, control: control, wire: wire, cancel: cancel}
	delete(w.notes, key)
	w.helpers.Add(1)
	go func() {
		defer w.helpers.Done()
		result := browserbridge.MeetResult{Status: "unknown", Reason: "not-confirmed"}
		select {
		case received, ok := <-results:
			if ok {
				result = received
			}
		case <-requestCtx.Done():
			if errors.Is(requestCtx.Err(), context.Canceled) {
				result.Reason = "stopped"
			}
		}
		select {
		case w.results <- completed{id, result}:
		case <-ctx.Done():
			// The worker owns terminal completion during shutdown.
		}
	}()
}

func (w *worker) finish(done completed, now time.Time) {
	p, ok := w.pending[done.id]
	if !ok {
		return
	}
	delete(w.pending, done.id)
	p.cancel()
	result, wire := done.result, p.wire
	status, detail := snoofer.Unknown, "Meet state not confirmed"
	matched := result.Tab == wire.Tab && result.Instance == wire.Instance && result.Call == wire.Call && result.Control == wire.Control && result.Revision == wire.Revision && result.Value == wire.Value
	if matched {
		switch result.Status {
		case "observed":
			if w.eligible(p.target, now) {
				status, detail = snoofer.Processed, "Meet state observed"
			}
		case "rejected":
			status, detail = snoofer.Rejected, "Meet request rejected: "+result.Reason
		}
	}
	if w.eligible(p.target, now) && status != snoofer.Processed {
		w.notes[actionKey{p.target, p.control}] = "Not confirmed"
	}
	p.command.Complete(snoofer.Outcome{Status: status, Detail: detail})
}
