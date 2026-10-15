package googlemeet

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"sound-snoofer/internal/browserbridge"
	deck "sound-snoofer/plugins/streamdeck"
	"sound-snoofer/snoofer"
)

type sent struct {
	ctx     context.Context
	browser string
	command browserbridge.MeetCommand
	result  chan browserbridge.MeetResult
}
type fakeLease struct {
	mu                sync.Mutex
	updates           chan browserbridge.MeetUpdate
	generations       map[string]uint64
	sent              []sent
	sendErr, closeErr error
	closed            bool
	status            browserbridge.Status
}

func (f *fakeLease) Updates() <-chan browserbridge.MeetUpdate { return f.updates }
func (f *fakeLease) Generation(browser string) uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0
	}
	return f.generations[browser]
}
func (f *fakeLease) Status() browserbridge.Status { f.mu.Lock(); defer f.mu.Unlock(); return f.status }
func (f *fakeLease) Close(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return f.closeErr
}
func (f *fakeLease) Send(ctx context.Context, browser string, command browserbridge.MeetCommand) (<-chan browserbridge.MeetResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	result := make(chan browserbridge.MeetResult, 1)
	f.sent = append(f.sent, sent{ctx, browser, command, result})
	return result, nil
}
func (f *fakeLease) writes() []sent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sent(nil), f.sent...)
}
func (f *fakeLease) generation(browser string, value uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.generations[browser] = value
}

type rig struct {
	t      *testing.T
	w      *worker
	lease  *fakeLease
	ctx    context.Context
	cancel context.CancelFunc
	now    time.Time
}

func newRig(t *testing.T, live bool) *rig {
	t.Helper()
	connection := &fakeLease{updates: make(chan browserbridge.MeetUpdate, 16), generations: map[string]uint64{"Brave": 1, "Firefox": 1}, status: browserbridge.Status{Port: 47815}}
	r := &rig{t: t, lease: connection, now: time.Now()}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.w = newWorker(snoofer.Services{Controls: snoofer.NewControls(), Live: live}, connection)
	t.Cleanup(func() { r.cancel(); r.w.helpers.Wait() })
	return r
}
func boolValue(value bool) *bool { return &value }
func joined(tab int64) browserbridge.MeetCall {
	return browserbridge.MeetCall{Tab: tab, Instance: "1", Call: "1", Sample: 1, State: "joined",
		Mic:    browserbridge.MeetControl{Value: boolValue(true), Revision: 1, Available: true},
		Camera: browserbridge.MeetControl{Value: boolValue(false), Revision: 1, Available: true}}
}
func (r *rig) report(calls ...browserbridge.MeetCall) {
	if calls == nil {
		calls = []browserbridge.MeetCall{}
	}
	r.w.receive(browserbridge.MeetUpdate{Browser: "Brave", Generation: r.lease.Generation("Brave"), Connected: true, Supported: true, Received: r.now, Calls: calls})
	r.w.publish(r.ctx, r.now)
}
func (r *rig) control(id string) snoofer.Control {
	r.t.Helper()
	for _, c := range r.w.services.Controls.Snapshot() {
		if c.ID == id {
			return c
		}
	}
	r.t.Fatal("missing control", id)
	return snoofer.Control{}
}
func (r *rig) queue(id, value string) (command, <-chan snoofer.Outcome) {
	r.t.Helper()
	op := "press"
	if id == "googlemeet.target" {
		op = "set"
	}
	outcomes := make(chan snoofer.Outcome, 2)
	err := r.w.services.Controls.DispatchWithResult(r.ctx, snoofer.Request{ID: id, Revision: r.control(id).Revision, Operation: op, Value: value}, func(result snoofer.Outcome) { outcomes <- result })
	if err != nil {
		r.t.Fatal(err)
	}
	select {
	case c := <-r.w.requests:
		c.accepted = r.now
		return c, outcomes
	case <-time.After(time.Second):
		r.t.Fatal("request was not queued")
	}
	return command{}, outcomes
}
func outcome(t *testing.T, results <-chan snoofer.Outcome, want snoofer.OutcomeStatus) {
	t.Helper()
	select {
	case got := <-results:
		if got.Status != want {
			t.Fatalf("outcome %+v, want %s", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("missing outcome")
	}
}
func (r *rig) selectCall(index int) {
	r.t.Helper()
	c, done := r.queue("googlemeet.target", r.w.choices[index].value())
	r.w.handle(r.ctx, c, r.now)
	outcome(r.t, done, snoofer.Processed)
	r.w.publish(r.ctx, r.now)
}
func (r *rig) settle(index int, status string) {
	r.t.Helper()
	sent := r.lease.writes()[index]
	c := sent.command
	sent.result <- browserbridge.MeetResult{ID: "1", Tab: c.Tab, Instance: c.Instance, Call: c.Call, Control: c.Control, Revision: c.Revision, Value: c.Value, Status: status, Reason: "unavailable"}
	r.finishNext()
}
func (r *rig) finishNext() {
	r.t.Helper()
	select {
	case result := <-r.w.results:
		r.w.finish(result, r.now)
	case <-time.After(time.Second):
		r.t.Fatal("missing worker result")
	}
	r.w.publish(r.ctx, r.now)
}

func TestMeetObservedControlsAndBindings(t *testing.T) {
	r := newRig(t, true)
	original := joined(7)
	r.report(original)
	if len(Plugin().Requires) != 0 {
		t.Fatal("Meet depends on another provider")
	}
	for _, check := range []struct{ id, value, icon string }{{"googlemeet.mute", "On", "mic-mute-muted"}, {"googlemeet.camera", "Off", "discord-video-off"}} {
		c := r.control(check.id)
		if c.Value != check.value || c.Icon != check.icon || !c.Available || c.Mirrors != "" || c.SurfaceOnly {
			t.Fatal(c)
		}
	}
	layout := deck.Layout{Home: "meet", Pages: []deck.Page{{ID: "meet", Name: "Meet"}}}
	for n, id := range []string{"googlemeet.mute", "googlemeet.camera", "googlemeet.call"} {
		layout.Pages[0].Keys[n] = deck.Binding{Control: id}
	}
	if err := layout.Validate(r.w.services.Controls.Snapshot()); err != nil {
		t.Fatal("controls are not bindable", err)
	}
	unknown := original
	unknown.Sample++
	unknown.Mic = browserbridge.MeetControl{Revision: 2, Reason: "unknown"}
	r.report(unknown)
	if c := r.control("googlemeet.mute"); c.Value != "N/A" || c.Available || c.Status != "Unknown state" {
		t.Fatal(c)
	}
	if !r.control("googlemeet.camera").Available {
		t.Fatal("mic state disabled independent camera")
	}
	if len(r.lease.writes()) != 0 {
		t.Fatal("observation wrote state")
	}
}

func TestMeetSelectionPinsAndNeverFallsBackAfterLoss(t *testing.T) {
	r := newRig(t, true)
	a, b := joined(7), joined(8)
	r.report(a)
	selected := r.w.selected
	r.report(a, b)
	if r.w.selected != selected {
		t.Fatal("new call changed selection")
	}
	r.report(b)
	if r.w.selected != (target{}) || r.control("googlemeet.mute").Available {
		t.Fatal("lost target fell back")
	}
	if c := r.control("googlemeet.target"); c.Value != "" || c.OptionLabels[""] != "Select a call" || c.Options[0] != "" {
		t.Fatal("selector implies a target", c)
	}
	r.report(a)
	if r.w.selected != (target{}) {
		t.Fatal("same URL/lifetime return reactivated auto-selection")
	}
	r.selectCall(0)
	if r.w.selected.tab != 7 || len(r.lease.writes()) != 0 {
		t.Fatal("selection changed mic/camera")
	}
}

func TestMeetInitiallyAmbiguousCallsRequireSelection(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7), joined(8))
	if r.w.selected != (target{}) || r.control("googlemeet.call").Value != "Select a call" {
		t.Fatal("ambiguous call auto-selected")
	}
	r.report(joined(8))
	if r.w.selected != (target{}) {
		t.Fatal("ambiguity later removal auto-selected")
	}
	r.selectCall(0)
	if r.w.selected.tab != 8 {
		t.Fatal(r.w.selected)
	}
}

func TestMeetFreshnessIsPerTabAndPerSample(t *testing.T) {
	r := newRig(t, true)
	a, b := joined(7), joined(8)
	r.report(a)
	r.report(a, b)
	r.now = r.now.Add(2 * time.Second)
	b.Sample++
	r.report(a, b)
	r.now = r.now.Add(time.Second)
	b.Sample++
	r.report(a, b)
	if r.w.selected != (target{}) || len(r.w.choices) != 1 || r.w.choices[0].tab != 8 {
		t.Fatal("cached sample/other tab kept target fresh", r.w.selected, r.w.choices)
	}
	a.Sample = 0
	r.report(a, b)
	if len(r.w.choices) != 1 {
		t.Fatal("lower sample resurrected target")
	}
	a.Sample = 2
	r.report(a, b)
	if len(r.w.choices) != 2 || r.w.selected != (target{}) {
		t.Fatal("fresh state should need explicit reselection")
	}
}

func TestMeetQueuedActionsRejectChangedIdentity(t *testing.T) {
	for _, change := range []string{"connection", "document", "call", "selection", "selection ABA", "revision", "revision ABA", "unknown", "prejoin", "removed", "expired"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			a, b := joined(7), joined(8)
			r.report(a)
			r.report(a, b)
			c, done := r.queue("googlemeet.mute", "")
			a.Sample++
			switch change {
			case "connection":
				r.lease.generation("Brave", 2)
			case "document":
				a.Instance = "2"
			case "call":
				a.Call = "2"
			case "selection", "selection ABA":
				r.selectCall(1)
				if change == "selection ABA" {
					r.selectCall(0)
				}
			case "revision":
				a.Mic.Revision++
			case "revision ABA":
				a.Mic.Value = boolValue(false)
				a.Mic.Revision++
				r.report(a, b)
				a.Sample++
				a.Mic.Value = boolValue(true)
				a.Mic.Revision++
			case "unknown":
				a.Mic.Value = nil
				a.Mic.Available = false
				a.Mic.Revision++
			case "prejoin":
				a.State, a.Call = "prejoin", ""
			case "removed":
				r.report(b)
			case "expired":
				r.now = r.now.Add(freshness)
			}
			if change != "removed" && change != "expired" {
				r.report(a, b)
			}
			r.w.handle(r.ctx, c, r.now)
			outcome(t, done, snoofer.Rejected)
			if len(r.lease.writes()) != 0 {
				t.Fatal("stale request wrote", r.lease.writes())
			}
		})
	}
}

func TestMeetExplicitIntentPendingAndIndependentResults(t *testing.T) {
	r := newRig(t, true)
	a := joined(7)
	r.report(a)
	mic, micDone := r.queue("googlemeet.mute", "")
	again, againDone := r.queue("googlemeet.mute", "")
	camera, cameraDone := r.queue("googlemeet.camera", "")
	r.w.handle(r.ctx, mic, r.now)
	r.w.handle(r.ctx, again, r.now)
	outcome(t, againDone, snoofer.Rejected)
	r.w.handle(r.ctx, camera, r.now)
	r.w.publish(r.ctx, r.now)
	writes := r.lease.writes()
	if len(writes) != 2 || writes[0].command.Control != "mic" || writes[0].command.Value || writes[1].command.Control != "camera" || !writes[1].command.Value {
		t.Fatal(writes)
	}
	for _, id := range []string{"googlemeet.mute", "googlemeet.camera"} {
		c := r.control(id)
		if c.Status != "Pending" || !c.Available {
			t.Fatal(c)
		}
	}
	if r.control("googlemeet.mute").Value != "On" || r.control("googlemeet.camera").Value != "Off" {
		t.Fatal("optimistic values")
	}
	select {
	case <-micDone:
		t.Fatal("socket write completed outcome")
	default:
	}
	a.Sample++
	a.Mic.Revision++
	a.Mic.Value = boolValue(false)
	r.report(a)
	select {
	case <-micDone:
		t.Fatal("snapshot completed correlated action")
	default:
	}
	r.settle(0, "observed")
	outcome(t, micDone, snoofer.Processed)
	if r.control("googlemeet.camera").Status != "Pending" {
		t.Fatal("mic completed camera")
	}
	r.settle(1, "rejected")
	outcome(t, cameraDone, snoofer.Rejected)
	if r.control("googlemeet.camera").Value != "Off" {
		t.Fatal("result changed actual camera state")
	}
}

func TestMeetNoReplayAfterLostTargetOrUncertainResult(t *testing.T) {
	for _, change := range []string{"disconnect", "call", "timeout", "malformed result"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			a := joined(7)
			r.report(a)
			c, done := r.queue("googlemeet.camera", "")
			if change == "timeout" {
				c.accepted = time.Now().Add(-completionTimeout + 20*time.Millisecond)
			}
			r.w.handle(r.ctx, c, r.now)
			switch change {
			case "disconnect":
				r.lease.generation("Brave", 0)
				r.w.reconcile(r.now)
				r.finishNext()
			case "call":
				a.Call = "2"
				a.Sample++
				r.report(a)
				r.finishNext()
			case "timeout":
				r.finishNext()
			case "malformed result":
				r.lease.writes()[0].result <- browserbridge.MeetResult{Status: "observed"}
				r.finishNext()
			}
			outcome(t, done, snoofer.Unknown)
			r.lease.generation("Brave", 2)
			a.Sample++
			r.report(a)
			if len(r.lease.writes()) != 1 {
				t.Fatal("uncertain operation replayed")
			}
			select {
			case duplicate := <-done:
				t.Fatal("duplicate outcome", duplicate)
			default:
			}
		})
	}
}

func TestMeetPreviewAndOldCompanionRemainUnavailable(t *testing.T) {
	r := newRig(t, false)
	r.report(joined(7))
	if c := r.control("googlemeet.mute"); c.Available || c.Status != "Preview" || c.Value != "On" {
		t.Fatal(c)
	}
	if err := Plugin().Validate([]byte(`{"saved_camera":true}`)); err == nil {
		t.Fatal("accepted persisted activation setting")
	}
	r.w.receive(browserbridge.MeetUpdate{Browser: "Brave", Generation: 1, Connected: true, Received: r.now})
	r.w.publish(r.ctx, r.now)
	if c := r.control("googlemeet.app-client"); c.Connection.State != snoofer.ConnectionAttention || c.Value != "Update companion" {
		t.Fatal(c)
	}
	if len(r.lease.writes()) != 0 {
		t.Fatal("preview/upgrade wrote")
	}
}

func TestMeetLeaseCloseErrorIsReturned(t *testing.T) {
	r := newRig(t, true)
	r.lease.closeErr = errors.New("close failed")
	i := start(r.ctx, r.cancel, r.w)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := i.Stop(ctx); !errors.Is(err, r.lease.closeErr) {
		t.Fatal(err)
	}
	if len(r.w.services.Controls.Snapshot()) != 0 {
		t.Fatal("controls survived stop")
	}
}

func TestMeetHelloDoesNotClaimAnObservedEmptyCallList(t *testing.T) {
	r := newRig(t, true)
	metadata := browserbridge.MeetUpdate{Browser: "Brave", Generation: 1, Connected: true, Supported: true, Received: r.now}
	r.w.receive(metadata)
	r.w.publish(r.ctx, r.now)
	if c := r.control("googlemeet.call"); c.Value != "Unknown call state" {
		t.Fatal("hello claimed call absence", c)
	}
	empty := metadata
	empty.Calls = []browserbridge.MeetCall{}
	r.w.receive(empty)
	r.w.publish(r.ctx, r.now)
	if c := r.control("googlemeet.call"); c.Value != "Not in a call" {
		t.Fatal("observed absence missing", c)
	}
	r.w.receive(metadata)
	r.w.publish(r.ctx, r.now)
	if c := r.control("googlemeet.call"); c.Value != "Not in a call" {
		t.Fatal("metadata erased observed absence", c)
	}
	r.lease.generation("Brave", 2)
	metadata.Generation = 2
	r.w.receive(metadata)
	r.w.publish(r.ctx, r.now)
	if c := r.control("googlemeet.call"); c.Value != "Unknown call state" {
		t.Fatal("new generation reused old absence", c)
	}
}

func TestMeetSelectionCancelsAcceptedWorkBeforeDelayedWrite(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7))
	r.report(joined(7), joined(8))
	c, done := r.queue("googlemeet.mute", "")
	r.w.handle(r.ctx, c, r.now)
	accepted := r.lease.writes()[0]
	r.selectCall(1)
	select {
	case <-accepted.ctx.Done():
	default:
		t.Fatal("selection change left old unmute eligible at the bridge write gate")
	}
	r.finishNext()
	outcome(t, done, snoofer.Unknown)
	if r.control("googlemeet.mute").Status != "" || r.w.selected.tab != 8 {
		t.Fatal("old result changed new target")
	}
}

func TestMeetAdmissionFailureAndQueueDeadlineDoNotWrite(t *testing.T) {
	for _, failure := range []string{"send", "deadline"} {
		t.Run(failure, func(t *testing.T) {
			r := newRig(t, true)
			r.report(joined(7))
			c, done := r.queue("googlemeet.mute", "")
			if failure == "send" {
				r.lease.sendErr = browserbridge.ErrBusy
			} else {
				c.accepted = r.now.Add(-completionTimeout)
			}
			r.w.handle(r.ctx, c, r.now)
			outcome(t, done, snoofer.Rejected)
			if len(r.lease.writes()) != 0 || len(r.w.pending) != 0 {
				t.Fatal("unsent rejection reserved or wrote")
			}
		})
	}
}

func TestMeetQueueFullAndStopSettleClaims(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7))
	active, activeDone := r.queue("googlemeet.mute", "")
	r.w.handle(r.ctx, active, r.now)
	var queued []<-chan snoofer.Outcome
	for range cap(r.w.requests) {
		result := make(chan snoofer.Outcome, 2)
		control := r.control("googlemeet.camera")
		if err := r.w.services.Controls.DispatchWithResult(r.ctx, snoofer.Request{ID: control.ID, Revision: control.Revision, Operation: "press"}, func(out snoofer.Outcome) { result <- out }); err != nil {
			t.Fatal(err)
		}
		queued = append(queued, result)
	}
	overflow := make(chan snoofer.Outcome, 2)
	control := r.control("googlemeet.camera")
	err := r.w.services.Controls.DispatchWithResult(r.ctx, snoofer.Request{ID: control.ID, Revision: control.Revision, Operation: "press"}, func(out snoofer.Outcome) { overflow <- out })
	if err == nil {
		t.Fatal("full queue accepted request")
	}
	outcome(t, overflow, snoofer.Rejected)
	r.cancel()
	outcome(t, activeDone, snoofer.Unknown)
	for _, result := range queued {
		outcome(t, result, snoofer.Unknown)
	}
	r.w.helpers.Wait()
	if len(r.lease.writes()) != 1 {
		t.Fatal("queued commands wrote after stop")
	}
	select {
	case extra := <-activeDone:
		t.Fatal("duplicate active outcome", extra)
	default:
	}
}

func TestMeetPendingLimitKeepsIndependentTargetState(t *testing.T) {
	r := newRig(t, true)
	var calls []browserbridge.MeetCall
	for tab := int64(1); tab <= 5; tab++ {
		calls = append(calls, joined(tab))
	}
	r.report(calls...)
	// Switching targets cancels previous accepted work. Until its waiter settles,
	// the bounded owner retains the reservation instead of growing more waiters.
	for index := range 4 {
		r.selectCall(index)
		for _, id := range []string{"googlemeet.mute", "googlemeet.camera"} {
			c, _ := r.queue(id, "")
			r.w.handle(r.ctx, c, r.now)
		}
	}
	r.selectCall(4)
	c, done := r.queue("googlemeet.mute", "")
	r.w.handle(r.ctx, c, r.now)
	outcome(t, done, snoofer.Rejected)
	if len(r.w.pending) != 8 || len(r.lease.writes()) != 8 {
		t.Fatal("pending bound not enforced")
	}
}

func TestMeetObservationsAndRequestsDoNotWriteOtherProviders(t *testing.T) {
	r := newRig(t, true)
	sideEffects := 0
	for _, owner := range []string{"audio", "discord"} {
		if err := r.w.services.Controls.Publish(owner, []snoofer.Control{{ID: owner + ".mute", Label: owner, Kind: "toggle", Value: "On", Available: true, Operations: []string{"press"}}}, func(context.Context, snoofer.Request) error { sideEffects++; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	r.w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { sideEffects++; return nil }
	r.report(joined(7))
	c, done := r.queue("googlemeet.mute", "")
	r.w.handle(r.ctx, c, r.now)
	r.settle(0, "observed")
	outcome(t, done, snoofer.Processed)
	if sideEffects != 0 || r.control("audio.mute").Value != "On" || r.control("discord.mute").Value != "On" {
		t.Fatal("Meet changed global/Discord state")
	}
}

func TestMeetPrejoinAndObservedEmptyListEndSelectedLifetime(t *testing.T) {
	r := newRig(t, true)
	prejoin := joined(7)
	prejoin.State, prejoin.Call = "prejoin", ""
	prejoin.Mic.Available, prejoin.Camera.Available = false, false
	prejoin.Mic.Reason, prejoin.Camera.Reason = "not-in-call", "not-in-call"
	r.report(prejoin)
	if r.w.selected != (target{}) || r.control("googlemeet.mute").Available || r.w.discovered {
		t.Fatal("prejoin consumed initial selection or enabled actions")
	}
	active := joined(7)
	active.Sample = 2
	r.report(active)
	if r.w.selected == (target{}) {
		t.Fatal("first real call was not selected")
	}
	r.report()
	if r.w.selected != (target{}) || r.control("googlemeet.call").Value != "Not in a call" {
		t.Fatal("observed empty list retained call")
	}
	active.Sample++
	r.report(active)
	if r.w.selected != (target{}) {
		t.Fatal("rejoin automatically restored selection")
	}
}
