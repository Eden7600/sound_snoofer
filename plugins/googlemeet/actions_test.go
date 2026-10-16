package googlemeet

import (
	"context"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/browserbridge"
	"sound-snoofer/snoofer"
)

func actionCall(tab int64) browserbridge.MeetCall {
	c := joined(tab)
	c.Share = &browserbridge.MeetControl{Value: boolValue(false), Identity: 1, Revision: 1, Available: true}
	c.Leave = &browserbridge.MeetControl{Value: boolValue(true), Identity: 1, Revision: 1, Available: true}
	return c
}

func (r *rig) captureAction(index int, action snoofer.MeetingAction) (snoofer.MeetingActionCommand, <-chan snoofer.Outcome) {
	t := r.w.choices[index]
	c := r.w.calls[tabKey{t.browser, t.tab}]
	state := controlState(c.state, string(action))
	want := true
	if action == snoofer.MeetingShare && state.Value != nil {
		want = !*state.Value
	}
	results := make(chan snoofer.Outcome, 3)
	return snoofer.MeetingActionCommand{Context: r.ctx, Peer: t.peer(), Lifetime: t.value(), Action: action, Revision: state.Revision, Value: want, Complete: func(out snoofer.Outcome) { results <- out }}, results
}

func (r *rig) waitControl(id string, match func(snoofer.Control) bool) snoofer.Control {
	r.t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, control := range r.w.services.Controls.Snapshot() {
			if control.ID == id && match(control) {
				return control
			}
		}
		time.Sleep(time.Millisecond)
	}
	r.t.Fatal("control did not reach expected state", id)
	return snoofer.Control{}
}

func TestMeetReportsOnlyExplicitActionCapabilities(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7), actionCall(8))
	old, current := r.adapter.peers[0], r.adapter.peers[1]
	if old.Share.Available || old.Leave.Available || old.Deafen.Available || current.Deafen.Available {
		t.Fatal("capability inferred from mic/camera", r.adapter.peers)
	}
	if !current.Share.Available || !current.Share.Known || current.Share.Value || !current.Leave.Available || !current.Leave.Value {
		t.Fatal("action capability missing", current)
	}
	for _, action := range []snoofer.MeetingAction{snoofer.MeetingShare, snoofer.MeetingLeave, snoofer.MeetingDeafen} {
		c, done := r.captureAction(0, action)
		r.w.applyAction(r.ctx, c, r.now)
		outcome(t, done, snoofer.Rejected)
	}
	if len(r.lease.writes()) != 0 {
		t.Fatal("old helper or unsupported deafen wrote")
	}
}

func TestMeetFocusedActionGuards(t *testing.T) {
	for _, action := range []snoofer.MeetingAction{snoofer.MeetingShare, snoofer.MeetingLeave} {
		for _, change := range []string{"connection", "document", "call", "revision", "revision ABA", "missing", "unknown", "identity", "unavailable", "expired", "canceled", "preview"} {
			t.Run(string(action)+"/"+change, func(t *testing.T) {
				r := newRig(t, true)
				a := actionCall(7)
				r.report(a)
				c, done := r.captureAction(0, action)
				state := a.Share
				if action == snoofer.MeetingLeave {
					state = a.Leave
				}
				a.Sample++
				switch change {
				case "connection":
					r.lease.generation("Brave", 2)
				case "document":
					a.Instance = "2"
				case "call":
					a.Call = "2"
				case "revision":
					state.Revision++
				case "revision ABA":
					state.Value = boolValue(!*state.Value)
					state.Revision++
					r.report(a)
					a.Sample++
					state.Value = boolValue(!*state.Value)
					state.Revision++
				case "missing":
					a.Share, a.Leave = nil, nil
				case "unknown":
					state.Value = nil
				case "identity":
					state.Identity = 0
				case "unavailable":
					state.Available = false
				case "expired":
					r.now = r.now.Add(freshness)
				case "canceled":
					ctx, cancel := context.WithCancel(r.ctx)
					cancel()
					c.Context = ctx
				case "preview":
					r.w.services.Live = false
				}
				if change != "expired" {
					r.report(a)
				}
				r.w.applyAction(r.ctx, c, r.now)
				outcome(t, done, snoofer.Rejected)
				if len(r.lease.writes()) != 0 {
					t.Fatal("obsolete focused action wrote")
				}
			})
		}
	}
}

func TestMeetShareSubmissionIsNotObservedSharing(t *testing.T) {
	r := newRig(t, true)
	r.report(actionCall(7), actionCall(8))
	r.selectCall(0)
	c, done := r.captureAction(1, snoofer.MeetingShare)
	r.adapter.actions = append(r.adapter.actions, c)
	r.refresh()
	write := r.lease.writes()[0]
	if write.command.Tab != 8 || write.command.Control != "share" || !write.command.Value {
		t.Fatal("diagnostic selection redirected focused action", write.command)
	}
	wire := write.command
	write.result <- browserbridge.MeetResult{Tab: wire.Tab, Instance: wire.Instance, Call: wire.Call, Control: wire.Control, Revision: wire.Revision, Value: wire.Value, Status: "submitted", Reason: "choose-source"}
	r.finishNext()
	select {
	case result := <-done:
		if result.Status != snoofer.Submitted || !strings.Contains(result.Detail, "Choose a source") {
			t.Fatal(result)
		}
	default:
		t.Fatal("missing submission")
	}
	if r.adapter.peers[1].Share.Value {
		t.Fatal("chooser claimed sharing")
	}
	r.refresh()
	if len(r.lease.writes()) != 1 {
		t.Fatal("submission replayed")
	}
}

func TestMeetLeaveWaitsForExactDepartureResultAfterSnapshot(t *testing.T) {
	r := newRig(t, true)
	r.adapter.integrated = true
	r.report(actionCall(7))
	c := r.waitControl("meetings.leave", func(c snoofer.Control) bool { return c.Available })
	done := make(chan snoofer.Outcome, 2)
	if err := r.w.services.Controls.DispatchWithResult(r.ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}, func(result snoofer.Outcome) { done <- result }); err != nil {
		t.Fatal(err)
	}
	r.refresh()
	writes := r.lease.writes()
	if len(writes) != 1 || writes[0].command.Control != "leave" || !writes[0].command.Value {
		t.Fatal("Leave capability was treated as already left", writes)
	}
	r.report()
	if writes[0].ctx.Err() != nil {
		t.Fatal("departure canceled correlated confirmation")
	}
	select {
	case out := <-done:
		t.Fatal("snapshot alone completed Leave", out)
	default:
	}
	r.settle(0, "observed")
	outcome(t, done, snoofer.Processed)
	r.report(actionCall(8))
	r.refresh()
	if len(r.lease.writes()) != 1 {
		t.Fatal("leave replayed into new call")
	}
}

func TestMeetActionCapacityAndNoReplay(t *testing.T) {
	r := newRig(t, true)
	var calls []browserbridge.MeetCall
	for n := int64(1); n <= 5; n++ {
		calls = append(calls, actionCall(n))
	}
	r.report(calls...)
	for n := range 4 {
		for _, property := range []snoofer.MeetingProperty{snoofer.MeetingMute, snoofer.MeetingCamera} {
			c, _ := r.capture(n, property)
			r.adapter.commands = append(r.adapter.commands, c)
		}
	}
	r.refresh()
	c, done := r.captureAction(4, snoofer.MeetingShare)
	r.adapter.actions = append(r.adapter.actions, c)
	r.refresh()
	if len(r.adapter.actions) != 1 || len(r.lease.writes()) != 8 {
		t.Fatal("focused action exceeded capacity")
	}
	r.settle(0, "observed")
	r.refresh()
	if len(r.adapter.actions) != 0 || len(r.lease.writes()) != 9 || len(r.w.pending) != 8 {
		t.Fatal("available slot did not progress")
	}
	r.settle(8, "unknown")
	outcome(t, done, snoofer.Unknown)
	r.lease.generation("Brave", 2)
	for n := range calls {
		calls[n].Sample++
	}
	r.report(calls...)
	r.refresh()
	if len(r.lease.writes()) != 9 {
		t.Fatal("uncertain action replayed on reconnect")
	}
}
