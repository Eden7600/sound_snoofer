package streamdeck

import (
	"fmt"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/controller"
)

type queued struct {
	event    Event
	identity string
}

// Queue serializes this surface's settings against acknowledgments. Relative
// turns retain their original target identity even while other clients edit.
type Queue struct {
	events             []queued
	inflight, sequence uint64
}

func (q *Queue) Push(e Event, s control.State) error {
	id := ""
	if e.Delta != 0 && e.Encoder >= 0 && e.Encoder < 3 {
		id = controller.GainIdentity(s.Plan, s.Snapshot, []string{"A1", "A2", "mic"}[e.Encoder])
	}
	if len(q.events) > 0 && e.Delta != 0 {
		last := &q.events[len(q.events)-1]
		sum := last.event.Delta + e.Delta
		if last.identity == id && last.event.Encoder == e.Encoder && last.event.Delta != 0 && sum >= -127 && sum <= 127 {
			last.event.Delta = sum
			return nil
		}
	}
	if len(q.events) >= 64 {
		return fmt.Errorf("Stream Deck command queue full")
	}
	q.events = append(q.events, queued{e, id})
	return nil
}
func (q *Queue) Next(s control.State) (control.Action, bool) {
	if q.inflight != 0 {
		if s.Acks["streamdeck"].ID < q.inflight {
			return control.Action{}, false
		}
		q.inflight = 0
	}
	for len(q.events) > 0 {
		event := q.events[0]
		q.events = q.events[1:]
		if !event.event.Press && event.event.Delta == 0 {
			continue
		}
		a, command := Action(event.event, s)
		if command != "audio" {
			continue
		}
		if a.Kind == control.Gain {
			a.Identity = event.identity
		}
		q.sequence++
		a.ID = q.sequence
		q.inflight = a.ID
		return a, true
	}
	return control.Action{}, false
}
func (q *Queue) Rejected() { q.inflight = 0 }
