package googlemeet

import (
	"context"
	"errors"
	"fmt"
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
	t       *testing.T
	w       *worker
	lease   *fakeLease
	adapter *testAdapter
	ctx     context.Context
	cancel  context.CancelFunc
	now     time.Time
}

func newRig(t *testing.T, live bool) *rig {
	t.Helper()
	connection := &fakeLease{updates: make(chan browserbridge.MeetUpdate, 16), generations: map[string]uint64{"Brave": 1, "Firefox": 1}, status: browserbridge.Status{Port: 47815}}
	r := &rig{t: t, lease: connection, now: time.Now()}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	controls := snoofer.NewControls()
	meetings := snoofer.NewMeetings(controls, live)
	if err := meetings.Start(r.ctx); err != nil {
		t.Fatal(err)
	}
	adapter, err := meetings.Attach(r.ctx, "googlemeet")
	if err != nil {
		t.Fatal(err)
	}
	r.adapter = &testAdapter{real: adapter}
	r.w = newWorker(snoofer.Services{Controls: controls, Meetings: meetings, Live: live}, connection, r.adapter)
	t.Cleanup(func() {
		r.cancel()
		r.adapter.Close()
		r.w.helpers.Wait()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := meetings.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	return r
}
func boolValue(value bool) *bool { return &value }
func joined(tab int64) browserbridge.MeetCall {
	return browserbridge.MeetCall{Tab: tab, Instance: "1", Call: "1", Sample: 1, State: "joined",
		Mic:    browserbridge.MeetControl{Value: boolValue(true), Identity: 1, Revision: 1, Available: true},
		Camera: browserbridge.MeetControl{Value: boolValue(false), Identity: 1, Revision: 1, Available: true}}
}
func (r *rig) report(calls ...browserbridge.MeetCall) {
	if calls == nil {
		calls = []browserbridge.MeetCall{}
	}
	r.w.receive(browserbridge.MeetUpdate{Browser: "Brave", Generation: r.lease.Generation("Brave"), Connected: true, Supported: true, Received: r.now, Calls: calls})
	r.refresh()
}
func (r *rig) refresh() {
	r.w.reconcile(r.now)
	r.w.synchronize(r.ctx, r.now)
	r.w.publish(r.ctx, r.now)
}

// Manual commands isolate provider guards; integration cases use real Next.
type testAdapter struct {
	real       *snoofer.MeetingAdapter
	peers      []snoofer.MeetingPeer
	commands   []snoofer.MeetingCommand
	integrated bool
	reports    int
}

func (a *testAdapter) Report(peers []snoofer.MeetingPeer) error {
	a.reports++
	a.peers = append([]snoofer.MeetingPeer(nil), peers...)
	return a.real.Report(peers)
}
func (a *testAdapter) Changed() <-chan struct{} { return a.real.Changed() }
func (a *testAdapter) Close()                   { a.real.Close() }
func (a *testAdapter) Next() (snoofer.MeetingCommand, bool) {
	if a.integrated {
		return a.real.Next()
	}
	if len(a.commands) == 0 {
		return snoofer.MeetingCommand{}, false
	}
	c := a.commands[0]
	a.commands = a.commands[1:]
	return c, true
}

func (r *rig) capture(index int, property snoofer.MeetingProperty) (snoofer.MeetingCommand, <-chan snoofer.Outcome) {
	t := r.w.choices[index]
	c := r.w.calls[tabKey{t.browser, t.tab}]
	state := c.state.Mic
	if property == snoofer.MeetingCamera {
		state = c.state.Camera
	}
	results := make(chan snoofer.Outcome, 3)
	return snoofer.MeetingCommand{Context: r.ctx, Peer: t.peer(), Lifetime: t.value(), Property: property, Revision: state.Revision, Value: !*state.Value, TargetRevision: 1, Complete: func(out snoofer.Outcome) { results <- out }}, results
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

func TestMeetSharedBindingsAndSeparateObservedDiagnostics(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7))
	if len(Plugin().Requires) != 0 {
		t.Fatal("Meet depends on another provider")
	}
	for _, check := range []struct{ id, value, icon, mirror string }{{"googlemeet.mute", "On", "mic-mute-muted", "meetings.mute"}, {"googlemeet.camera", "Off", "discord-video-off", "meetings.camera"}} {
		c := r.control(check.id)
		if c.Value != check.value || c.Icon != check.icon || !c.Available || c.Mirrors != check.mirror || c.SurfaceOnly {
			t.Fatal(c)
		}
	}
	layout := deck.Layout{Home: "meet", Pages: []deck.Page{{ID: "meet", Name: "Meet"}}}
	for n, id := range []string{"googlemeet.mute", "googlemeet.camera", "googlemeet.call"} {
		layout.Pages[0].Keys[n] = deck.Binding{Control: id}
	}
	if err := layout.Validate(r.w.services.Controls.Snapshot()); err != nil {
		t.Fatal(err)
	}
	c, done := r.queue("googlemeet.mute", "")
	r.w.handle(r.ctx, c, r.now)
	outcome(t, done, snoofer.Processed)
	r.w.publish(r.ctx, r.now)
	if r.control("googlemeet.mute").Value != "Off" || r.control("googlemeet.call-mute").Value != "On" {
		t.Fatal("target confused with app observation")
	}
	actual := r.control("googlemeet.call-mute")
	if !actual.SurfaceOnly || actual.Kind != "status" || len(actual.Operations) != 0 {
		t.Fatal(actual)
	}
	unknown := joined(7)
	unknown.Sample++
	unknown.Mic = browserbridge.MeetControl{Revision: 2, Reason: "unknown"}
	r.report(unknown)
	actual = r.control("googlemeet.call-mute")
	if actual.Value != "N/A" || actual.Status != "Unknown state" || !actual.Available || len(actual.Operations) != 0 {
		t.Fatal("unknown readout was hidden or actionable", actual)
	}
	if r.control("googlemeet.mute").Value != "Off" {
		t.Fatal("unknown observation overwrote shared target")
	}
}

func TestMeetLegacyIdentityRemainsReadableButCannotSynchronize(t *testing.T) {
	r := newRig(t, true)
	a := joined(7)
	a.Mic.Identity, a.Camera.Identity = 0, 0
	r.report(a)
	if r.adapter.peers[0].Mute.Available || r.adapter.peers[0].Camera.Available {
		t.Fatal("legacy identity synchronized")
	}
	if r.control("googlemeet.call-mute").Value != "On" || r.control("googlemeet.call-camera").Value != "Off" {
		t.Fatal("legacy readback lost")
	}
	c, done := r.capture(0, snoofer.MeetingMute)
	r.w.apply(r.ctx, c, r.now)
	outcome(t, done, snoofer.Rejected)
	if len(r.lease.writes()) != 0 {
		t.Fatal("legacy identity wrote")
	}
}

func TestMeetControlReplacementDoesNotBecomeSharedIntent(t *testing.T) {
	r := newRig(t, true)
	a := joined(7)
	r.report(a)
	a.Mic.Value = boolValue(false)
	a.Mic.Revision++
	a.Mic.Identity++
	a.Sample++
	r.report(a)
	if !r.w.services.Meetings.Snapshot().Mute.Value {
		t.Fatal("replacement became a global unmute")
	}
	if r.control("googlemeet.call-mute").Value != "Off" {
		t.Fatal("replacement observation hidden")
	}
}

func TestMeetLateMatchingReadbackClearsActualUncertainty(t *testing.T) {
	r := newRig(t, true)
	r.adapter.integrated = true
	a := joined(7)
	r.report(a)
	c, done := r.queue("googlemeet.camera", "")
	r.w.handle(r.ctx, c, r.now)
	outcome(t, done, snoofer.Processed)
	r.refresh()
	r.settle(0, "unknown")
	if r.control("googlemeet.call-camera").Status != "Not confirmed" {
		t.Fatal("unknown result hidden")
	}
	a.Sample++
	a.Camera.Revision++
	a.Camera.Value = boolValue(true)
	r.report(a)
	if r.control("googlemeet.call-camera").Status != "" {
		t.Fatal("late matching actual readback stayed uncertain")
	}
	if len(r.lease.writes()) != 1 {
		t.Fatal("late readback replayed command")
	}
}

func TestMeetObservedResultCannotReplayBeforeMatchingSnapshot(t *testing.T) {
	for _, newerContradiction := range []bool{false, true} {
		t.Run(fmt.Sprint(newerContradiction), func(t *testing.T) {
			r := newRig(t, true)
			r.adapter.integrated = true
			a := joined(7)
			r.report(a)
			c, done := r.queue("googlemeet.camera", "")
			r.w.handle(r.ctx, c, r.now)
			outcome(t, done, snoofer.Processed)
			r.refresh()
			if newerContradiction {
				a.Sample++
				a.Camera.Revision++
				r.report(a)
			}
			r.settle(0, "observed")
			r.refresh()
			if len(r.lease.writes()) != 1 {
				t.Fatal("observed result replayed before actual readback")
			}
			if r.control("googlemeet.call-camera").Value != "Off" {
				t.Fatal("result overwrote actual snapshot")
			}
			a.Sample++
			a.Camera.Revision++
			a.Camera.Value = boolValue(true)
			r.report(a)
			if len(r.lease.writes()) != 1 || r.control("googlemeet.call-camera").Value != "On" {
				t.Fatal("matching readback replayed or stayed stale")
			}
		})
	}
}

func TestMeetUnchangedSnapshotsDoNotFeedCoordinatorWake(t *testing.T) {
	r := newRig(t, true)
	a := joined(7)
	r.report(a)
	for range 4 {
		a.Sample++
		r.report(a)
		r.refresh()
	}
	if r.adapter.reports != 1 {
		t.Fatal("heartbeat fed coordinator wake", r.adapter.reports)
	}
	a.Mic.Revision++
	a.Mic.Value = boolValue(false)
	a.Sample++
	r.report(a)
	a.Mic.Revision++
	a.Mic.Value = boolValue(true)
	a.Sample++
	r.report(a)
	if r.adapter.reports != 3 {
		t.Fatal("real value transitions were coalesced", r.adapter.reports)
	}
}

func TestMeetKnownUnsentRefusalStaysRejectedAfterTargetLoss(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7))
	c, done := r.capture(0, snoofer.MeetingMute)
	r.w.apply(r.ctx, c, r.now)
	sent := r.lease.writes()[0]
	wire := sent.command
	r.lease.generation("Brave", 2)
	r.w.finish(completed{id: 1, result: browserbridge.MeetResult{Tab: wire.Tab, Instance: wire.Instance, Call: wire.Call, Control: wire.Control, Revision: wire.Revision, Value: wire.Value, Status: "rejected", Reason: "stale"}}, r.now)
	outcome(t, done, snoofer.Rejected)
	r.finishNext() // The canceled waiter cannot complete it a second time.
	select {
	case duplicate := <-done:
		t.Fatal("duplicate completion", duplicate)
	default:
	}
}

func TestMeetAllCallsFanoutWithoutSelection(t *testing.T) {
	r := newRig(t, true)
	r.adapter.integrated = true
	r.report(joined(8), joined(7))
	if r.w.selected != (target{}) || len(r.adapter.peers) != 2 || r.adapter.peers[0].ID != "Brave/7" {
		t.Fatal("selection or nondeterministic report", r.adapter.peers)
	}
	c, done := r.queue("googlemeet.camera", "")
	r.w.handle(r.ctx, c, r.now)
	outcome(t, done, snoofer.Processed)
	r.refresh()
	writes := r.lease.writes()
	if len(writes) != 2 || writes[0].command.Tab != 7 || writes[1].command.Tab != 8 {
		t.Fatal("fanout missing", writes)
	}
	r.selectCall(1)
	for _, sent := range writes {
		if sent.ctx.Err() != nil {
			t.Fatal("diagnostic selection canceled universal command")
		}
	}
	r.settle(0, "observed")
	r.settle(1, "observed")
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
	if r.w.selected != (target{}) || len(r.adapter.peers) != 1 || r.adapter.peers[0].ID != "Brave/8" {
		t.Fatal("cached sample stayed fresh", r.adapter.peers)
	}
	a.Sample = 0
	r.report(a, b)
	if len(r.adapter.peers) != 1 {
		t.Fatal("lower sample revived peer")
	}
	a.Sample = 2
	r.report(a, b)
	if len(r.adapter.peers) != 2 || r.adapter.peers[0].NewCall {
		t.Fatal("fresh recovery mistaken for new call")
	}
}

func TestMeetNewCallNeedsAuthoritativeEvidence(t *testing.T) {
	for _, change := range []string{"prejoin", "call", "document", "reconnect", "omitted", "unknown", "stale"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			a := joined(7)
			r.report(a)
			a.Sample++
			switch change {
			case "prejoin":
				a.State, a.Call = "prejoin", ""
				r.report(a)
				a.State, a.Call = "joined", "2"
				a.Sample++
			case "call":
				a.Call = "2"
			case "document":
				a.Instance = "2"
			case "reconnect":
				r.lease.generation("Brave", 2)
			case "omitted":
				r.report()
			case "unknown":
				a.State = "unknown"
				r.report(a)
				a.State = "joined"
				a.Sample++
			case "stale":
				r.now = r.now.Add(freshness)
			}
			r.report(a)
			if len(r.adapter.peers) != 1 || r.adapter.peers[0].NewCall != (change == "prejoin" || change == "call") {
				t.Fatal(r.adapter.peers)
			}
			r.refresh()
			if r.adapter.peers[0].NewCall {
				t.Fatal("repeated new call edge")
			}
		})
	}
}

func TestMeetClaimedCommandsRejectChangedIdentity(t *testing.T) {
	for _, change := range []string{"connection", "document", "call", "revision", "revision ABA", "unknown", "prejoin", "removed", "expired", "canceled", "property"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			a := joined(7)
			r.report(a)
			c, done := r.capture(0, snoofer.MeetingMute)
			a.Sample++
			switch change {
			case "connection":
				r.lease.generation("Brave", 2)
			case "document":
				a.Instance = "2"
			case "call":
				a.Call = "2"
			case "revision":
				a.Mic.Revision++
			case "revision ABA":
				a.Mic.Value = boolValue(false)
				a.Mic.Revision++
				r.report(a)
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
				r.report()
			case "expired":
				r.now = r.now.Add(freshness)
			case "canceled":
				ctx, cancel := context.WithCancel(r.ctx)
				cancel()
				c.Context = ctx
			case "property":
				c.Property = "leave"
			}
			if change != "removed" && change != "expired" {
				r.report(a)
			}
			r.w.apply(r.ctx, c, r.now)
			outcome(t, done, snoofer.Rejected)
			if len(r.lease.writes()) != 0 {
				t.Fatal("stale command wrote")
			}
		})
	}
}

func TestMeetExplicitIntentAndTerminalResults(t *testing.T) {
	r := newRig(t, true)
	a := joined(7)
	r.report(a)
	mic, micDone := r.capture(0, snoofer.MeetingMute)
	camera, cameraDone := r.capture(0, snoofer.MeetingCamera)
	r.w.apply(r.ctx, mic, r.now)
	r.w.apply(r.ctx, camera, r.now)
	if len(r.lease.writes()) != 2 || r.lease.writes()[0].command.Value || !r.lease.writes()[1].command.Value {
		t.Fatal("explicit intent lost")
	}
	select {
	case <-micDone:
		t.Fatal("write acknowledgement completed")
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
	if len(r.w.pending) != 1 {
		t.Fatal("mic settled camera")
	}
	r.settle(1, "rejected")
	outcome(t, cameraDone, snoofer.Rejected)
	r.w.finish(completed{id: 1}, r.now)
	select {
	case <-micDone:
		t.Fatal("duplicate terminal result")
	default:
	}
	noop, done := r.capture(0, snoofer.MeetingMute)
	noop.Value = false
	r.w.apply(r.ctx, noop, r.now)
	outcome(t, done, snoofer.Processed)
	if len(r.lease.writes()) != 2 {
		t.Fatal("already satisfied wrote")
	}
}

func TestMeetUncertainCommandAndReconnectDoNotReplay(t *testing.T) {
	r := newRig(t, true)
	r.adapter.integrated = true
	a, b := joined(7), joined(8)
	r.report(a, b)
	c, done := r.queue("googlemeet.camera", "")
	r.w.handle(r.ctx, c, r.now)
	outcome(t, done, snoofer.Processed)
	r.refresh()
	r.settle(0, "unknown")
	r.settle(1, "unknown")
	r.lease.generation("Brave", 2)
	a.Sample++
	b.Sample++
	r.report(a, b)
	r.refresh()
	if len(r.lease.writes()) != 2 {
		t.Fatal("uncertain command replayed", r.lease.writes())
	}
}

func TestMeetLossCancelsClaimedCommand(t *testing.T) {
	for _, change := range []string{"disconnect", "call", "timeout", "malformed result"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			a := joined(7)
			r.report(a)
			c, done := r.capture(0, snoofer.MeetingCamera)
			if change == "timeout" {
				ctx, cancel := context.WithTimeout(r.ctx, 20*time.Millisecond)
				defer cancel()
				c.Context = ctx
			}
			r.w.apply(r.ctx, c, r.now)
			switch change {
			case "disconnect":
				r.lease.generation("Brave", 0)
				r.w.reconcile(r.now)
			case "call":
				a.Call = "2"
				a.Sample++
				r.report(a)
			case "malformed result":
				r.lease.writes()[0].result <- browserbridge.MeetResult{Status: "observed"}
			}
			r.finishNext()
			outcome(t, done, snoofer.Unknown)
			if len(r.w.pending) != 0 {
				t.Fatal("pending survived loss")
			}
		})
	}
}

func TestMeetCapacityOnlyClaimsAvailableSlots(t *testing.T) {
	r := newRig(t, true)
	var calls []browserbridge.MeetCall
	for tab := int64(1); tab <= 5; tab++ {
		calls = append(calls, joined(tab))
	}
	r.report(calls...)
	for index := range 5 {
		for _, property := range []snoofer.MeetingProperty{snoofer.MeetingMute, snoofer.MeetingCamera} {
			c, _ := r.capture(index, property)
			r.adapter.commands = append(r.adapter.commands, c)
		}
	}
	r.refresh()
	if len(r.w.pending) != 8 || len(r.adapter.commands) != 2 || len(r.lease.writes()) != 8 {
		t.Fatal("claim capacity bound lost")
	}
	r.settle(0, "observed")
	r.refresh()
	if len(r.w.pending) != 8 || len(r.adapter.commands) != 1 || len(r.lease.writes()) != 9 {
		t.Fatal("free slot did not progress")
	}
}

func TestMeetPreviewAndOldCompanion(t *testing.T) {
	r := newRig(t, false)
	r.report(joined(7))
	c, done := r.capture(0, snoofer.MeetingMute)
	r.w.apply(r.ctx, c, r.now)
	outcome(t, done, snoofer.Rejected)
	if c := r.control("googlemeet.call-mute"); c.Status != "Preview" || c.Value != "On" {
		t.Fatal(c)
	}
	if r.control("googlemeet.mute").Available || len(r.lease.writes()) != 0 {
		t.Fatal("preview writes enabled")
	}
	if err := Plugin().Validate([]byte(`{"saved_camera":true}`)); err == nil {
		t.Fatal("persisted camera accepted")
	}
	r.w.receive(browserbridge.MeetUpdate{Browser: "Brave", Generation: 1, Connected: true, Received: r.now})
	r.refresh()
	if r.control("googlemeet.app-client").Value != "Update companion" {
		t.Fatal("old companion admitted")
	}
}

func TestMeetHelloDoesNotClaimObservedAbsence(t *testing.T) {
	r := newRig(t, true)
	metadata := browserbridge.MeetUpdate{Browser: "Brave", Generation: 1, Connected: true, Supported: true, Received: r.now}
	r.w.receive(metadata)
	r.refresh()
	if r.control("googlemeet.call").Value != "Unknown call state" {
		t.Fatal("hello claimed absence")
	}
	empty := metadata
	empty.Calls = []browserbridge.MeetCall{}
	r.w.receive(empty)
	r.refresh()
	if r.control("googlemeet.call").Value != "Not in a call" {
		t.Fatal("observed absence missing")
	}
	r.w.receive(metadata)
	r.refresh()
	if r.control("googlemeet.call").Value != "Not in a call" {
		t.Fatal("metadata erased absence")
	}
	r.lease.generation("Brave", 2)
	metadata.Generation = 2
	r.w.receive(metadata)
	r.refresh()
	if r.control("googlemeet.call").Value != "Unknown call state" {
		t.Fatal("new generation retained absence")
	}
}

func TestMeetLegacyActionRejectsStaleSharedRevision(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7))
	a, done := r.queue("googlemeet.camera", "")
	b, other := r.queue("googlemeet.camera", "")
	r.w.handle(r.ctx, a, r.now)
	outcome(t, done, snoofer.Processed)
	r.w.handle(r.ctx, b, r.now)
	outcome(t, other, snoofer.Rejected)
	if len(r.lease.writes()) != 0 {
		t.Fatal("legacy action bypassed coordinator")
	}
}

func TestMeetAdmissionFailureAndQueuedDeadline(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7))
	c, done := r.capture(0, snoofer.MeetingMute)
	r.lease.sendErr = browserbridge.ErrBusy
	r.w.apply(r.ctx, c, r.now)
	outcome(t, done, snoofer.Rejected)
	legacy, result := r.queue("googlemeet.mute", "")
	legacy.accepted = r.now.Add(-completionTimeout)
	r.w.handle(r.ctx, legacy, r.now)
	outcome(t, result, snoofer.Rejected)
	if len(r.lease.writes()) != 0 || len(r.w.pending) != 0 {
		t.Fatal("unsent rejection wrote")
	}
}

func TestMeetStopCompletesClaimOnceAndReturnsLeaseError(t *testing.T) {
	r := newRig(t, true)
	r.report(joined(7))
	c, done := r.capture(0, snoofer.MeetingMute)
	r.w.apply(r.ctx, c, r.now)
	r.lease.closeErr = errors.New("close failed")
	i := start(r.ctx, r.cancel, r.w)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := i.Stop(ctx); !errors.Is(err, r.lease.closeErr) {
		t.Fatal(err)
	}
	outcome(t, done, snoofer.Unknown)
	select {
	case extra := <-done:
		t.Fatal("duplicate terminal completion", extra)
	default:
	}
	for _, c := range r.w.services.Controls.Snapshot() {
		if c.ID == "googlemeet.mute" {
			t.Fatal("controls survived stop")
		}
	}
}
