package snoofer

import (
	"context"
	"testing"
	"time"
)

func actionPeer(id string) MeetingPeer {
	p := peer(id, false, false)
	p.Label = "Call " + id
	p.Deafen = MeetingValue{Known: true, Available: true, Revision: 1}
	p.Share = MeetingValue{Available: true, Revision: 1}
	p.Leave = MeetingValue{Known: true, Value: true, Available: true, Revision: 1}
	return p
}

func focusedControl(t *testing.T, m *Meetings, id string) (Control, func(context.Context, Request) error) {
	t.Helper()
	controls, handle := m.focusedControls()
	for _, control := range controls {
		if control.ID == id {
			return control, handle
		}
	}
	t.Fatalf("missing control %s", id)
	return Control{}, nil
}

func focusedLabel(t *testing.T, m *Meetings) string {
	t.Helper()
	controls, _ := m.focusedControls()
	label := ""
	for _, c := range controls {
		if c.Collection == "meetings.focus" && c.Value == "On" {
			if label != "" {
				t.Fatal("multiple focused calls")
			}
			label = c.Label
		}
	}
	return label
}

func chooseFocus(t *testing.T, m *Meetings, label string) {
	t.Helper()
	controls, handle := m.focusedControls()
	for _, c := range controls {
		if c.Collection == "meetings.focus" && c.Label == label {
			if err := handle(context.Background(), Request{ID: c.ID, Operation: "press"}); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("missing focus %s", label)
}

func queueAction(t *testing.T, m *Meetings, n int) <-chan Outcome {
	t.Helper()
	m.mu.Lock()
	p := m.focus
	capture := meetingActionCapture{p, m.focusEpoch, p.focusOrder, p.state.Lifetime, meetingActionObserved(p.state, n)}
	m.mu.Unlock()
	results := make(chan Outcome, 2)
	if err := m.requestAction(context.Background(), n, capture, func(out Outcome) { results <- out }); err != nil {
		t.Fatal(err)
	}
	return results
}

func nextAction(t *testing.T, a *MeetingAdapter) MeetingActionCommand {
	t.Helper()
	c, ok := a.NextAction()
	if !ok {
		t.Fatal("missing focused action")
	}
	return c
}

func noAction(t *testing.T, a *MeetingAdapter) {
	t.Helper()
	if c, ok := a.NextAction(); ok {
		t.Fatalf("unexpected focused action %+v", c)
	}
}

func actionResult(t *testing.T, results <-chan Outcome, status OutcomeStatus) {
	t.Helper()
	select {
	case got := <-results:
		if got.Status != status {
			t.Fatalf("result %s, want %s: %s", got.Status, status, got.Detail)
		}
	case <-time.After(time.Second):
		t.Fatal("missing focused action completion")
	}
}

func TestMeetingFocusNewestPinnedAndDeparture(t *testing.T) {
	m, a := meetingRig(t, true)
	one, two, three := actionPeer("a"), actionPeer("b"), actionPeer("c")
	reportMeeting(t, a, one)
	if got := focusedLabel(t, m); got != one.Label {
		t.Fatal(got)
	}
	reportMeeting(t, a, two, one)
	if got := focusedLabel(t, m); got != two.Label {
		t.Fatal(got)
	}
	one.Mute.Revision++
	reportMeeting(t, a, one, two)
	if got := focusedLabel(t, m); got != two.Label {
		t.Fatal("heartbeat stole focus", got)
	}
	chooseFocus(t, m, one.Label)
	reportMeeting(t, a, one, two, three)
	if got := focusedLabel(t, m); got != one.Label {
		t.Fatal("new call replaced explicit focus", got)
	}
	reportMeeting(t, a, two, three)
	if got := focusedLabel(t, m); got != three.Label {
		t.Fatal("departed focus did not fall back to newest", got)
	}
	a.Close()
	if got := focusedLabel(t, m); got != "" {
		t.Fatal(got)
	}
}

func TestMeetingFocusLifetimeAndSlotABA(t *testing.T) {
	m, a := meetingRig(t, true)
	one := actionPeer("a")
	reportMeeting(t, a, one)
	old, oldHandler := focusedControl(t, m, "meetings.focus-1")
	chooseFocus(t, m, one.Label)
	one.Lifetime = "call-2"
	reportMeeting(t, a, one)
	fresh, _ := focusedControl(t, m, old.ID)
	if fresh.Epoch == old.Epoch {
		t.Fatal("replacement reused epoch")
	}
	if err := oldHandler(context.Background(), Request{ID: old.ID, Operation: "press"}); err == nil {
		t.Fatal("old lifetime selected replacement")
	}
	reportMeeting(t, a)
	reportMeeting(t, a, actionPeer("b"))
	if err := oldHandler(context.Background(), Request{ID: old.ID, Operation: "press"}); err == nil {
		t.Fatal("old slot selected unrelated call")
	}
}

func TestMeetingFocusedActionsCaptureAndIndependentReservations(t *testing.T) {
	m, a := meetingRig(t, true)
	p := actionPeer("a")
	reportMeeting(t, a, p)
	results := []<-chan Outcome{queueAction(t, m, 0), queueAction(t, m, 1), queueAction(t, m, 2)}
	for n := range 3 {
		c := nextAction(t, a)
		if c.Action != meetingActionName(n) || c.Peer != p.ID || c.Lifetime != p.Lifetime || c.Revision != 1 || !c.Value {
			t.Fatalf("wrong capture %+v", c)
		}
		control, handle := focusedControl(t, m, "meetings."+string(c.Action))
		if control.Available || control.Status != "Pending" {
			t.Fatal(control)
		}
		if err := handle(context.Background(), Request{ID: control.ID, Operation: "press"}); err == nil {
			t.Fatal("duplicate pending action accepted")
		}
		c.Complete(Outcome{Status: Processed})
		actionResult(t, results[n], Processed)
	}
	noAction(t, a)
	reportMeeting(t, a, p)
	noAction(t, a)
}

func TestMeetingFocusedActionRejectsStaleFocusAndObservation(t *testing.T) {
	m, a := meetingRig(t, true)
	one, two := actionPeer("a"), actionPeer("b")
	reportMeeting(t, a, one, two)
	old, handle := focusedControl(t, m, "meetings.share")
	chooseFocus(t, m, one.Label)
	chooseFocus(t, m, two.Label)
	if err := handle(context.Background(), Request{ID: old.ID, Operation: "press"}); err == nil {
		t.Fatal("focus ABA accepted")
	}
	_, handle = focusedControl(t, m, "meetings.share")
	two.Share.Revision++
	reportMeeting(t, a, one, two)
	if err := handle(context.Background(), Request{ID: old.ID, Operation: "press"}); err == nil {
		t.Fatal("stale action observation accepted")
	}
	results := queueAction(t, m, 1)
	two.Share.Identity++
	reportMeeting(t, a, one, two)
	noAction(t, a)
	actionResult(t, results, Rejected)
}

func TestMeetingFocusChangeCancelsUnsentAndRetainsClaimedReservation(t *testing.T) {
	m, a := meetingRig(t, true)
	one, two := actionPeer("a"), actionPeer("b")
	reportMeeting(t, a, one, two)
	unsent := queueAction(t, m, 0)
	claimed := queueAction(t, m, 1)
	deafen, share := nextAction(t, a), nextAction(t, a)
	deafen.Complete(Outcome{Status: Processed})
	actionResult(t, unsent, Processed)
	unsent = queueAction(t, m, 0)
	chooseFocus(t, m, one.Label)
	actionResult(t, unsent, Rejected)
	select {
	case <-share.Context.Done():
	default:
		t.Fatal("claimed obsolete action not canceled")
	}
	control, handle := focusedControl(t, m, "meetings.share")
	if control.Available {
		t.Fatal("claimed reservation released early")
	}
	if err := handle(context.Background(), Request{ID: control.ID, Operation: "press"}); err == nil {
		t.Fatal("second same-kind action overlapped old claim")
	}
	share.Complete(Outcome{Status: Unknown})
	actionResult(t, claimed, Unknown)
	noAction(t, a)
	queueAction(t, m, 1)
	if c := nextAction(t, a); c.Peer != one.ID {
		t.Fatal("new explicit action retargeted", c)
	}
}

func TestMeetingActionTimeoutNoReplayAndLateCompletion(t *testing.T) {
	m, a := meetingRig(t, true)
	p := actionPeer("a")
	reportMeeting(t, a, p)
	results := queueAction(t, m, 1)
	c := nextAction(t, a)
	m.tick(time.Now().Add(6 * time.Second))
	actionResult(t, results, Unknown)
	control, _ := focusedControl(t, m, "meetings.share")
	if control.Status != "Not confirmed" {
		t.Fatal(control)
	}
	c.Complete(Outcome{Status: Processed})
	m.tick(time.Now())
	select {
	case out := <-results:
		t.Fatal("completed twice", out)
	default:
	}
	for range 3 {
		p.Share.Revision++
		reportMeeting(t, a, p)
		noAction(t, a)
	}
	a.Close()
	a, err := m.Attach(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	p.Lifetime = "reconnected"
	reportMeeting(t, a, p)
	noAction(t, a)
}

func TestMeetingLeaveDeparturePreservesExactCompletion(t *testing.T) {
	m, a := meetingRig(t, true)
	p := actionPeer("a")
	reportMeeting(t, a, p)
	results := queueAction(t, m, 2)
	c := nextAction(t, a)
	reportMeeting(t, a)
	if c.Context.Err() != nil {
		t.Fatal("departure canceled its own Leave result", c.Context.Err())
	}
	c.Complete(Outcome{Status: Processed, Detail: "Call left"})
	actionResult(t, results, Processed)
	noAction(t, a)
}

func TestMeetingActionCapabilitiesPreviewAndNoCall(t *testing.T) {
	for _, live := range []bool{true, false} {
		m, a := meetingRig(t, live)
		for _, id := range []string{"meetings.deafen", "meetings.share", "meetings.leave"} {
			c, handle := focusedControl(t, m, id)
			if c.Available || c.Value != "N/A" {
				t.Fatal(c)
			}
			if err := handle(context.Background(), Request{ID: id, Operation: "press"}); err == nil {
				t.Fatal("action without call accepted")
			}
		}
		p := actionPeer("meet")
		p.Deafen = MeetingValue{}
		reportMeeting(t, a, p)
		c, handle := focusedControl(t, m, "meetings.deafen")
		if c.Available || c.Value != "N/A" {
			t.Fatal("Meet invented deafen", c)
		}
		if err := handle(context.Background(), Request{ID: c.ID, Operation: "press"}); err == nil {
			t.Fatal("unsupported action accepted")
		}
		if !live {
			c, handle = focusedControl(t, m, "meetings.share")
			if c.Available || handle(context.Background(), Request{ID: c.ID, Operation: "press"}) == nil {
				t.Fatal("preview accepted action")
			}
			c, handle = focusedControl(t, m, "meetings.focus-1")
			if c.Available || handle(context.Background(), Request{ID: c.ID, Operation: "press"}) == nil {
				t.Fatal("preview changed focus")
			}
		}
		noAction(t, a)
	}
}

func TestMeetingActionValidationAndSubmittedBoundary(t *testing.T) {
	m, a := meetingRig(t, true)
	p := actionPeer("a")
	p.Share.Revision = 0
	if err := a.Report([]MeetingPeer{p}); err == nil {
		t.Fatal("available action without revision accepted")
	}
	p.Share.Revision = 1
	p.Deafen.Known = false
	if err := a.Report([]MeetingPeer{p}); err == nil {
		t.Fatal("unknown deafen accepted")
	}
	p.Deafen.Known = true
	reportMeeting(t, a, p)
	results := queueAction(t, m, 1)
	c := nextAction(t, a)
	c.Complete(Outcome{Status: Submitted, Detail: "Choose a source in Meet"})
	actionResult(t, results, Submitted)
	control, _ := focusedControl(t, m, "meetings.share")
	if control.Value == "On" || control.Status != "Choose a source in Meet" {
		t.Fatal("menu opening became sharing", control)
	}
	noAction(t, a)
}

func TestMeetingActionShutdownCompletionOutsideLock(t *testing.T) {
	m, a := meetingRig(t, true)
	reportMeeting(t, a, actionPeer("a"))
	results := queueAction(t, m, 0)
	c := nextAction(t, a)
	m.mu.Lock()
	old := m.actions[0].complete
	m.actions[0].complete = func(out Outcome) {
		m.Snapshot()
		old(out)
	}
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Close(ctx); err != nil {
		t.Fatal("completion blocked under core lock", err)
	}
	actionResult(t, results, Unknown)
	if c.Context.Err() == nil {
		t.Fatal("shutdown left command alive")
	}
	noAction(t, a)
}

func TestMeetingActionObservationEpochInvalidatesUnchangedDisplay(t *testing.T) {
	m, a := meetingRig(t, true)
	p := actionPeer("a")
	reportMeeting(t, a, p)
	old, _ := focusedControl(t, m, "meetings.share")
	p.Share.Revision++
	reportMeeting(t, a, p)
	fresh, _ := focusedControl(t, m, old.ID)
	if fresh.Epoch == old.Epoch || fresh.Value != old.Value {
		t.Fatal("invisible action identity did not invalidate registry capture", old, fresh)
	}
}

func TestMeetingLeaveReplacementPreservesOriginalResultOnly(t *testing.T) {
	m, a := meetingRig(t, true)
	p := actionPeer("a")
	reportMeeting(t, a, p)
	results := queueAction(t, m, 2)
	c := nextAction(t, a)
	p.Lifetime = "replacement"
	reportMeeting(t, a, p)
	if c.Context.Err() != nil || c.Lifetime != "call-1" {
		t.Fatal("replacement lost original claimed leave", c.Lifetime, c.Context.Err())
	}
	noAction(t, a)
	c.Complete(Outcome{Status: Processed})
	actionResult(t, results, Processed)
	if focusedLabel(t, m) != p.Label {
		t.Fatal("old completion changed new call focus")
	}
	noAction(t, a)
}
