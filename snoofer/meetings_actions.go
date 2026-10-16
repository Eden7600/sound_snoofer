package snoofer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// MeetingAction identifies an explicit operation on the focused call.
type MeetingAction string

const (
	MeetingDeafen MeetingAction = "deafen"
	MeetingShare  MeetingAction = "share"
	MeetingLeave  MeetingAction = "leave"
)

// MeetingActionCommand is one captured operation owned by the adapter worker.
// Complete is terminal: Processed means observed, Submitted may open a chooser.
type MeetingActionCommand struct {
	Context        context.Context
	Peer, Lifetime string
	Action         MeetingAction
	Revision       uint64
	Value          bool
	Complete       Completion
}

type meetingActionCapture struct {
	peer          *meetingPeer
	focus, joined uint64
	lifetime      string
	observed      MeetingValue
}

type meetingShareSubmission struct {
	capture meetingActionCapture
	value   bool
}

type meetingActionAttempt struct {
	capture  meetingActionCapture
	action   int
	value    bool
	ctx      context.Context
	cancel   context.CancelFunc
	stop     func() bool
	complete Completion
	deadline time.Time
	claimed  bool
	done     bool
}

func meetingActionName(n int) MeetingAction {
	return [...]MeetingAction{MeetingDeafen, MeetingShare, MeetingLeave}[n]
}

func meetingActionObserved(p MeetingPeer, n int) MeetingValue {
	return [...]MeetingValue{p.Deafen, p.Share, p.Leave}[n]
}

func meetingActionAvailable(p MeetingPeer, n int) bool {
	v := meetingActionObserved(p, n)
	return p.Joined && v.Available && v.Revision != 0 && (n != 0 || v.Known)
}

// refreshFocusLocked preserves slots for active lifetimes and explicit selection.
func (m *Meetings) refreshFocusLocked() {
	used := map[int]bool{}
	for _, p := range m.peers {
		if !p.present || !p.state.Joined || p.focusLifetime != p.state.Lifetime {
			p.focusSlot = 0
		}
		if p.focusSlot != 0 {
			used[p.focusSlot] = true
		}
	}
	keys := make([]string, 0, len(m.peers))
	for key, p := range m.peers {
		if p.present && p.state.Joined && p.focusSlot == 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		p := m.peers[key]
		for p.focusSlot = 1; used[p.focusSlot]; p.focusSlot++ {
		}
		used[p.focusSlot] = true
		m.nextFocus++
		p.focusOrder, p.focusLifetime = m.nextFocus, p.state.Lifetime
	}
	if m.focus != nil && (!m.focus.present || !m.focus.state.Joined || m.focusOrder != m.focus.focusOrder) {
		m.pinnedFocus = false
	}
	if m.pinnedFocus {
		return
	}
	var newest *meetingPeer
	for _, p := range m.peers {
		if p.present && p.state.Joined && (newest == nil || p.focusOrder > newest.focusOrder) {
			newest = p
		}
	}
	m.setFocusLocked(newest)
}

func (m *Meetings) setFocusLocked(p *meetingPeer) {
	order := uint64(0)
	if p != nil {
		order = p.focusOrder
	}
	if m.focus == p && m.focusOrder == order {
		return
	}
	m.focus, m.focusOrder = p, order
	m.focusEpoch++
	m.actionStatus = [3]string{}
	m.shareSubmitted = nil
	for _, attempt := range m.actions {
		if attempt != nil {
			m.cancelActionLocked(attempt, "Focused call changed")
		}
	}
	m.changedLocked()
}

func (m *Meetings) selectFocus(ctx context.Context, p *meetingPeer, order uint64, complete Completion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	if !m.live || m.closed || m.ctx == nil || m.ctx.Err() != nil || p == nil || !p.present || !p.state.Joined || p.focusOrder != order {
		m.mu.Unlock()
		return errors.New("meeting call changed or unavailable")
	}
	m.pinnedFocus = true
	m.setFocusLocked(p)
	m.mu.Unlock()
	if complete != nil {
		complete(Outcome{Status: Processed, Detail: "Call focused"})
	}
	return nil
}

func (m *Meetings) actionCurrentLocked(c meetingActionCapture, n int) bool {
	return m.focus == c.peer && m.focusEpoch == c.focus && c.peer != nil && c.peer.present && c.peer.state.Joined &&
		c.peer.focusOrder == c.joined && c.peer.state.Lifetime == c.lifetime && meetingActionObserved(c.peer.state, n) == c.observed && meetingActionAvailable(c.peer.state, n)
}

func (m *Meetings) requestAction(ctx context.Context, n int, captured meetingActionCapture, complete Completion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.live || m.closed || m.ctx == nil || m.ctx.Err() != nil || !m.actionCurrentLocked(captured, n) {
		return errors.New("meeting action changed or unavailable")
	}
	if m.actions[n] != nil {
		return errors.New("meeting action pending")
	}
	value := true
	if n != 2 && captured.observed.Known {
		value = !captured.observed.Value
	}
	commandCtx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	attempt := &meetingActionAttempt{capture: captured, action: n, value: value, ctx: commandCtx, cancel: cancel, complete: complete, deadline: time.Now().Add(5 * time.Second)}
	attempt.stop = context.AfterFunc(ctx, cancel)
	m.actions[n], m.actionStatus[n] = attempt, ""
	if n == 1 {
		m.shareSubmitted = nil
	}
	m.changedLocked()
	return nil
}

// NextAction claims one explicit action, never automatic reconciliation work.
func (a *MeetingAdapter) NextAction() (MeetingActionCommand, bool) {
	m := a.owner
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.live || a.closed || a.ctx.Err() != nil || m.closed || m.ctx.Err() != nil {
		return MeetingActionCommand{}, false
	}
	for n, attempt := range m.actions {
		if attempt == nil || attempt.claimed || attempt.capture.peer.adapter != a {
			continue
		}
		if attempt.ctx.Err() != nil || !m.actionCurrentLocked(attempt.capture, n) {
			m.finishActionLocked(attempt, Outcome{Status: Rejected, Detail: "Meeting action changed before sending"})
			continue
		}
		attempt.claimed = true
		return MeetingActionCommand{Context: attempt.ctx, Peer: attempt.capture.peer.state.ID, Lifetime: attempt.capture.lifetime, Action: meetingActionName(n), Revision: attempt.capture.observed.Revision, Value: attempt.value,
			Complete: func(outcome Outcome) {
				m.mu.Lock()
				defer m.mu.Unlock()
				m.finishActionLocked(attempt, outcome)
			}}, true
	}
	return MeetingActionCommand{}, false
}

func (m *Meetings) finishActionLocked(a *meetingActionAttempt, outcome Outcome) {
	if a.done {
		return
	}
	a.done = true
	a.cancel()
	if a.stop != nil {
		a.stop()
	}
	if m.actions[a.action] == a {
		m.actions[a.action] = nil
	}
	if a.capture.focus == m.focusEpoch {
		switch outcome.Status {
		case Unknown:
			m.actionStatus[a.action] = "Not confirmed"
		case Failed:
			m.actionStatus[a.action] = "Failed"
		case Submitted:
			m.actionStatus[a.action] = outcome.Detail
			if a.action == 1 {
				m.shareSubmitted = &meetingShareSubmission{a.capture, a.value}
				m.clearSubmittedShareLocked()
			}
		default:
			m.actionStatus[a.action] = ""
		}
	}
	if a.complete != nil {
		m.completions = append(m.completions, meetingCompletion{a.complete, outcome})
	}
	m.changedLocked()
}

func (m *Meetings) clearSubmittedShareLocked() {
	s := m.shareSubmitted
	if s == nil || s.capture.focus != m.focusEpoch {
		return
	}
	p := s.capture.peer
	value := p.state.Share
	if p.present && p.state.Joined && p.state.Lifetime == s.capture.lifetime && value.Known && value.Value == s.value && value.Revision != s.capture.observed.Revision {
		m.actionStatus[1] = ""
		m.shareSubmitted = nil
		m.changedLocked()
	}
}

func (m *Meetings) cancelActionLocked(a *meetingActionAttempt, detail string) {
	// Departure can be observed before its correlated Leave result arrives.
	if a.claimed && a.action == 2 && !a.capture.peer.adapter.closed && (!a.capture.peer.present || !a.capture.peer.state.Joined || a.capture.peer.state.Lifetime != a.capture.lifetime) {
		return
	}
	a.cancel()
	if !a.claimed {
		m.finishActionLocked(a, Outcome{Status: Rejected, Detail: detail})
	}
}

func (m *Meetings) tickActionsLocked(now time.Time) {
	for _, a := range m.actions {
		if a == nil {
			continue
		}
		if !a.claimed && (a.ctx.Err() != nil || !m.actionCurrentLocked(a.capture, a.action)) {
			m.finishActionLocked(a, Outcome{Status: Rejected, Detail: "Meeting action changed before sending"})
		} else if !now.Before(a.deadline) {
			m.finishActionLocked(a, Outcome{Status: Unknown, Detail: "Meeting action was not confirmed"})
		}
	}
}

// Focus and action captures are made with the same snapshot as their controls.
func (m *Meetings) focusedControls() ([]Control, func(context.Context, Request) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	captures := [3]meetingActionCapture{}
	controls := make([]Control, 0, 3+len(m.peers))
	for n, label := range []string{"Deafen", "Share", "Leave"} {
		value, status, available := "N/A", m.actionStatus[n], false
		icon := [...]string{"discord-deafen", "screen-share", "call-leave"}[n]
		kind := "toggle"
		if n == 2 {
			kind = "command"
		}
		if p := m.focus; p != nil {
			v := meetingActionObserved(p.state, n)
			captures[n] = meetingActionCapture{p, m.focusEpoch, p.focusOrder, p.state.Lifetime, v}
			available = meetingActionAvailable(p.state, n)
			if available {
				value = ""
				if n == 1 && !v.Known {
					value = "Ready"
				}
				if n != 2 && v.Known {
					value = OnOff(v.Value)
				}
				if n == 0 && v.Value {
					icon += "-muted"
				}
			}
		}
		if captures[n] != m.publishedActions[n] {
			m.actionEpoch[n]++
			m.publishedActions[n] = captures[n]
		}
		if m.actions[n] != nil {
			status = "Pending"
		}
		controls = append(controls, Control{ID: "meetings." + string(meetingActionName(n)), Label: label, ShortLabel: label, Group: "Meetings", Kind: kind, Icon: icon, Value: value, Status: status, Available: m.live && available, Epoch: m.actionEpoch[n], Operations: []string{"press"}})
	}
	type focusCapture struct {
		peer  *meetingPeer
		order uint64
	}
	focuses := map[string]focusCapture{}
	for _, p := range m.peers {
		if !p.present || !p.state.Joined {
			continue
		}
		id := fmt.Sprintf("meetings.focus-%d", p.focusSlot)
		label := p.state.Label
		if label == "" {
			label = p.adapter.provider
		}
		controls = append(controls, Control{ID: id, Label: label, ShortLabel: label, Group: "Meetings", Kind: "toggle", Icon: "call", Value: OnOff(m.focus == p), Available: m.live, Epoch: p.focusOrder, Operations: []string{"press"}, Collection: "meetings.focus", CollectionLabel: "Calls", Order: p.focusSlot})
		focuses[id] = focusCapture{p, p.focusOrder}
	}
	return controls, func(ctx context.Context, r Request) error {
		if r.Operation != "press" {
			return errors.New("invalid meeting operation")
		}
		if focus, ok := focuses[r.ID]; ok {
			return m.selectFocus(ctx, focus.peer, focus.order, r.ClaimCompletion(m.ctx))
		}
		for n := range captures {
			if r.ID == "meetings."+string(meetingActionName(n)) {
				return m.requestAction(ctx, n, captures[n], r.ClaimCompletion(m.ctx))
			}
		}
		return errors.New("unknown meeting control")
	}
}
