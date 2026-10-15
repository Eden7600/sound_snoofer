package snoofer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func meetingRig(t *testing.T, live bool) (*Meetings, *MeetingAdapter) {
	t.Helper()
	m := NewMeetings(NewControls(), live)
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := m.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	a, err := m.Attach(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	return m, a
}

func peer(id string, mute, camera bool) MeetingPeer {
	return MeetingPeer{ID: id, Lifetime: "call-1", Joined: true, Mute: MeetingValue{Known: true, Value: mute, Available: true, Revision: 1}, Camera: MeetingValue{Known: true, Value: camera, Available: true, Revision: 1}}
}
func reportMeeting(t *testing.T, a *MeetingAdapter, peers ...MeetingPeer) {
	t.Helper()
	if err := a.Report(peers); err != nil {
		t.Fatal(err)
	}
}
func requestMeeting(t *testing.T, m *Meetings, property MeetingProperty, want bool) {
	t.Helper()
	s := m.Snapshot().Mute
	if property == MeetingCamera {
		s = m.Snapshot().Camera
	}
	if err := m.Request(context.Background(), property, s.Revision, want, nil); err != nil {
		t.Fatal(err)
	}
}
func nextMeeting(t *testing.T, a *MeetingAdapter) MeetingCommand {
	t.Helper()
	c, ok := a.Next()
	if !ok {
		t.Fatal("no meeting command")
	}
	return c
}
func noMeeting(t *testing.T, a *MeetingAdapter) {
	t.Helper()
	if c, ok := a.Next(); ok {
		t.Fatalf("unexpected command %+v", c)
	}
}
func waitMeeting(t *testing.T, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !f() {
		if time.Now().After(deadline) {
			t.Fatal("meeting condition timed out")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestMeetingsBaselineExternalAndEcho(t *testing.T) {
	m, a := meetingRig(t, true)
	one, two := peer("a", false, false), peer("b", true, true)
	reportMeeting(t, a, two, one) // Stable peer order establishes false before true.
	if s := m.Snapshot(); s.Mute.Value || s.Camera.Value || !s.Camera.Known {
		t.Fatal(s)
	}
	first, second := nextMeeting(t, a), nextMeeting(t, a)
	if first.Peer != "b" || first.Property != MeetingMute || first.Value || second.Property != MeetingCamera || second.Value {
		t.Fatal(first, second)
	}
	two.Mute.Value, two.Camera.Value = false, false
	two.Mute.Revision, two.Camera.Revision = 2, 2
	reportMeeting(t, a, one, two)
	first.Complete(Outcome{Status: Processed})
	second.Complete(Outcome{Status: Processed})
	noMeeting(t, a)
	one.Mute.Value = true
	one.Mute.Revision++
	reportMeeting(t, a, one, two)
	if !m.Snapshot().Mute.Value {
		t.Fatal("external mute not adopted")
	}
	c := nextMeeting(t, a)
	if c.Peer != "b" || !c.Value {
		t.Fatal(c)
	}
	// A revision/availability-only update cannot replace latest received intent.
	two.Camera.Revision++
	two.Camera.Available = false
	reportMeeting(t, a, one, two)
	if m.Snapshot().Camera.Value {
		t.Fatal("capability changed camera intent")
	}
}

func TestMeetingsCameraPeriodAndMuteRetention(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("a", true, true)
	reportMeeting(t, a, one)
	old := m.Snapshot().Camera.Revision
	reportMeeting(t, a)
	if s := m.Snapshot(); s.Camera.Known || !s.Mute.Known || !s.Mute.Value {
		t.Fatal(s)
	}
	one.Lifetime = "call-2"
	one.Camera.Value = false
	reportMeeting(t, a, one)
	if s := m.Snapshot(); s.Camera.Value || s.Camera.Revision == old {
		t.Fatal(s)
	}
	two := peer("b", false, true)
	reportMeeting(t, a, one, two)
	if m.Snapshot().Camera.Value {
		t.Fatal("new peer replaced active camera target")
	}
}

func TestMeetingsUnknownSurvivesReconnectAndLateReadbackResolves(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("slot", false, false)
	reportMeeting(t, a, one)
	requestMeeting(t, m, MeetingMute, true)
	c := nextMeeting(t, a)
	c.Complete(Outcome{Status: Unknown})
	if m.Snapshot().Mute.Status != "Not confirmed" {
		t.Fatal(m.Snapshot())
	}
	for range 3 {
		one.Mute.Revision++
		reportMeeting(t, a, one)
		noMeeting(t, a)
	}
	a.Close()
	a, err := m.Attach(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	one.Lifetime = "reconnected"
	reportMeeting(t, a, one)
	noMeeting(t, a)
	one.Mute.Value = true
	one.Mute.Revision++
	reportMeeting(t, a, one)
	if s := m.Snapshot(); !s.Mute.Value || s.Mute.Status != "" {
		t.Fatalf("matching late readback did not resolve uncertainty: %+v", s)
	}
}

func TestMeetingsSupersededWriteKeepsReservationAndCannotAdoptEcho(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("slot", false, false)
	reportMeeting(t, a, one)
	requestMeeting(t, m, MeetingMute, true)
	c := nextMeeting(t, a)
	requestMeeting(t, m, MeetingMute, false)
	select {
	case <-c.Context.Done():
	default:
		t.Fatal("obsolete command not canceled")
	}
	noMeeting(t, a)
	one.Mute.Value = true
	one.Mute.Revision++
	reportMeeting(t, a, one)
	if m.Snapshot().Mute.Value {
		t.Fatal("own delayed success adopted as intent")
	}
	noMeeting(t, a)
	c.Complete(Outcome{Status: Processed})
	next := nextMeeting(t, a)
	if next.Value {
		t.Fatal(next)
	}
}

func TestMeetingsOldUnknownCannotFenceDefiniteNewCall(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("slot", false, false)
	reportMeeting(t, a, one)
	requestMeeting(t, m, MeetingMute, true)
	old := nextMeeting(t, a)
	one.Lifetime = "new-call"
	one.NewCall = true
	reportMeeting(t, a, one)
	noMeeting(t, a) // Reservation remains until the old transport settles.
	old.Complete(Outcome{Status: Unknown})
	fresh := nextMeeting(t, a)
	if fresh.Lifetime != "new-call" || !fresh.Value {
		t.Fatal(fresh)
	}
}

func TestMeetingsExternalLatestValueAndUnknownCannotSeed(t *testing.T) {
	m, a := meetingRig(t, true)
	one, two := peer("a", false, false), peer("b", false, false)
	one.Camera.Known, one.Camera.Available = false, false
	reportMeeting(t, a, one)
	if m.Snapshot().Camera.Known {
		t.Fatal("unknown seeded camera")
	}
	reportMeeting(t, a, one, two)
	one.Mute.Value = true
	one.Mute.Revision++
	reportMeeting(t, a, one, two)
	// A contradictory initial baseline did not change intent, but a later real
	// value edge does, even while another peer has not converged.
	two.Mute.Value = true
	two.Mute.Revision++
	reportMeeting(t, a, one, two)
	two.Mute.Value = false
	two.Mute.Revision++
	reportMeeting(t, a, one, two)
	if m.Snapshot().Mute.Value {
		t.Fatal("latest received value edge lost")
	}
}

func TestMeetingsPreviewDoesNotWriteAndStaleRequestRejected(t *testing.T) {
	m, a := meetingRig(t, false)
	reportMeeting(t, a, peer("a", false, false), peer("b", true, true))
	noMeeting(t, a)
	if err := m.Request(context.Background(), MeetingMute, m.Snapshot().Mute.Revision, true, nil); err == nil {
		t.Fatal("preview accepted target write")
	}
	live, adapter := meetingRig(t, true)
	reportMeeting(t, adapter, peer("a", false, false))
	old := live.Snapshot().Mute.Revision
	requestMeeting(t, live, MeetingMute, true)
	requestMeeting(t, live, MeetingMute, false)
	if err := live.Request(context.Background(), MeetingMute, old, true, nil); err == nil {
		t.Fatal("ABA request accepted")
	}
}

type micSave struct {
	want     bool
	revision uint64
	complete Completion
}

func audioMeeting(t *testing.T, m *Meetings) (*MeetingAudio, chan micSave) {
	t.Helper()
	writes := make(chan micSave, 8)
	a, err := m.AttachAudio(context.Background(), func(want bool, revision uint64, complete Completion) error {
		writes <- micSave{want, revision, complete}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return a, writes
}
func receiveMicSave(t *testing.T, writes <-chan micSave) micSave {
	t.Helper()
	select {
	case s := <-writes:
		return s
	case <-time.After(time.Second):
		t.Fatal("no mute save")
		return micSave{}
	}
}

func TestMeetingsAudioSaveBeforeFanoutAndLateAuthority(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("a", false, false)
	reportMeeting(t, a, one)
	audio, writes := audioMeeting(t, m)
	audio.Report(MeetingMicPreference{Known: true, Muted: true, Applied: false, Revision: 10})
	if !m.Snapshot().Mute.Value {
		t.Fatal("runtime baseline replaced saved authority")
	}
	c := nextMeeting(t, a)
	one.Mute.Value = true
	one.Mute.Revision++
	reportMeeting(t, a, one)
	c.Complete(Outcome{Status: Processed})
	one.Mute.Value = false
	one.Mute.Revision++
	reportMeeting(t, a, one)
	save := receiveMicSave(t, writes)
	if save.want || save.revision != 10 || !m.Snapshot().Mute.Value {
		t.Fatal("unsaved intent propagated", save, m.Snapshot())
	}
	noMeeting(t, a)
	audio.Report(MeetingMicPreference{Known: true, Muted: false, Applied: true, Revision: 11})
	save.complete(Outcome{Status: Processed})
	waitMeeting(t, func() bool { return m.Snapshot().Mute.Status == "" })
	if m.Snapshot().Mute.Value {
		t.Fatal("saved preference not adopted")
	}
}

func TestMeetingsSaveFailureAndUnknownNeverRetry(t *testing.T) {
	for _, status := range []OutcomeStatus{Failed, Unknown, Rejected} {
		t.Run(string(status), func(t *testing.T) {
			m, a := meetingRig(t, true)
			audio, writes := audioMeeting(t, m)
			audio.Report(MeetingMicPreference{Known: true, Muted: false, Applied: true, Revision: 7})
			one := peer("a", false, false)
			reportMeeting(t, a, one)
			one.Mute.Value = true
			one.Mute.Revision++
			reportMeeting(t, a, one)
			save := receiveMicSave(t, writes)
			save.complete(Outcome{Status: status, Detail: "save failed"})
			waitMeeting(t, func() bool { return m.Snapshot().Mute.Status == "Not confirmed" })
			if m.Snapshot().Mute.Value {
				t.Fatal("failed save changed target")
			}
			for range 3 {
				reportMeeting(t, a, one)
				m.tick(time.Now())
			}
			select {
			case <-writes:
				t.Fatal("failed save retried")
			default:
			}
			noMeeting(t, a)
		})
	}
}

func TestMeetingsCanceledQueuedMuteDoesNotSave(t *testing.T) {
	m, a := meetingRig(t, true)
	audio, writes := audioMeeting(t, m)
	audio.Report(MeetingMicPreference{Known: true, Muted: false, Revision: 1})
	reportMeeting(t, a, peer("a", false, false))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Request(ctx, MeetingMute, m.Snapshot().Mute.Revision, true, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-writes:
		t.Fatal("canceled request saved")
	default:
	}
}

func TestMeetingsAudioCloseDoesNotLoseRuntimeTarget(t *testing.T) {
	m, a := meetingRig(t, true)
	audio, _ := audioMeeting(t, m)
	audio.Report(MeetingMicPreference{Known: true, Muted: true, Revision: 1})
	audio.Close()
	reportMeeting(t, a, peer("a", false, false))
	if !m.Snapshot().Mute.Value {
		t.Fatal("audio disappearance erased known target")
	}
	requestMeeting(t, m, MeetingMute, false)
	if m.Snapshot().Mute.Value {
		t.Fatal("audio absence blocks runtime target")
	}
}

func TestMeetingsIdenticalReportDoesNotWakeItself(t *testing.T) {
	_, a := meetingRig(t, true)
	one := peer("slot", false, false)
	reportMeeting(t, a, one)
	for {
		select {
		case <-a.Changed():
		default:
			goto drained
		}
	}
drained:
	reportMeeting(t, a, one)
	select {
	case <-a.Changed():
		t.Fatal("unchanged report created a self-wake loop")
	default:
	}
}

func TestMeetingsDirectLastCallReplacementSeedsCameraAgain(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("slot", false, true)
	reportMeeting(t, a, one)
	one.Lifetime, one.NewCall, one.Camera.Value = "new-call", true, false
	reportMeeting(t, a, one)
	if m.Snapshot().Camera.Value {
		t.Fatal("last old call's camera target survived a definite new call")
	}
}

func TestMeetingsLatestExternalMuteWaitsForPriorSaveReadback(t *testing.T) {
	m, adapter := meetingRig(t, true)
	audio, writes := audioMeeting(t, m)
	audio.Report(MeetingMicPreference{Known: true, Muted: false, Revision: 1})
	one, two := peer("a", false, false), peer("b", false, false)
	reportMeeting(t, adapter, one, two)
	one.Mute.Value, one.Mute.Revision = true, 2
	reportMeeting(t, adapter, one, two)
	first := receiveMicSave(t, writes)
	two.Mute.Value, two.Mute.Revision = true, 2
	reportMeeting(t, adapter, one, two)
	two.Mute.Value, two.Mute.Revision = false, 3
	reportMeeting(t, adapter, one, two)
	first.complete(Outcome{Status: Processed})
	m.tick(time.Now())
	select {
	case <-writes:
		t.Fatal("next intent saved before prior readback supplied its revision")
	default:
	}
	audio.Report(MeetingMicPreference{Known: true, Muted: true, Revision: 2})
	second := receiveMicSave(t, writes)
	if second.want || second.revision != 2 {
		t.Fatal(second)
	}
	audio.Report(MeetingMicPreference{Known: true, Muted: false, Revision: 3})
	second.complete(Outcome{Status: Processed})
	waitMeeting(t, func() bool { return m.Snapshot().Mute.Status == "" })
	if m.Snapshot().Mute.Value {
		t.Fatal("latest external intent lost")
	}
}

func manualMeetings(t *testing.T) *Meetings {
	t.Helper()
	m := NewMeetings(NewControls(), true)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	t.Cleanup(func() { m.cancel(); m.stop() })
	return m
}

func TestMeetingsQueuedMuteDoesNotRebaseAcrossNewAudioEdit(t *testing.T) {
	m := manualMeetings(t)
	audio, writes := audioMeeting(t, m)
	audio.Report(MeetingMicPreference{Known: true, Muted: false, Revision: 1})
	m.mu.Lock()
	m.externalLocked(0, true)
	m.mu.Unlock()
	// A later audio edit is authoritative even while its boolean stays false.
	audio.Report(MeetingMicPreference{Known: true, Muted: false, Revision: 2})
	m.tick(time.Now())
	select {
	case save := <-writes:
		t.Fatalf("stale external intent rebased onto newer audio revision: %+v", save)
	default:
	}
	if m.Snapshot().Mute.Value {
		t.Fatal("stale external intent overrode Audio")
	}
}

func TestMeetingsStopSettlesAlreadyQueuedCompletions(t *testing.T) {
	m := manualMeetings(t)
	audio, _ := audioMeeting(t, m)
	audio.Report(MeetingMicPreference{Known: true, Muted: false, Revision: 1})
	done := make(chan Outcome, 1)
	if err := m.Request(context.Background(), MeetingMute, m.Snapshot().Mute.Revision, true, func(o Outcome) { done <- o }); err != nil {
		t.Fatal(err)
	}
	audio.Close() // Queues its terminal completion just before owner stop.
	m.stop()
	select {
	case <-done:
	default:
		t.Fatal("stop lost a previously settled callback")
	}
}

func TestMeetingsHistoricalUncertaintyIsNotCurrentConvergence(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("gone", false, false)
	reportMeeting(t, a, one)
	requestMeeting(t, m, MeetingMute, true)
	c := nextMeeting(t, a)
	c.Complete(Outcome{Status: Unknown})
	reportMeeting(t, a)
	if m.Snapshot().Mute.Status != "" {
		t.Fatal("departed call remains current pending/failure status")
	}
}

func TestMeetingsRetiredSlotsArePrunedAndUnknownFencesAreBounded(t *testing.T) {
	m, a := meetingRig(t, true)
	for n := 0; n < 300; n++ {
		one := peer(fmt.Sprint(n), false, false)
		reportMeeting(t, a, one)
		reportMeeting(t, a)
	}
	m.mu.Lock()
	retained := len(m.peers)
	m.mu.Unlock()
	if retained != 0 {
		t.Fatalf("retained %d resolved retired slots", retained)
	}
	one := peer("first", false, false)
	reportMeeting(t, a, one)
	requestMeeting(t, m, MeetingMute, true)
	for n := 0; n < 128; n++ {
		one = peer(fmt.Sprint(n), false, false)
		reportMeeting(t, a, one)
		command := nextMeeting(t, a)
		command.Complete(Outcome{Status: Unknown})
	}
	if err := a.Report([]MeetingPeer{peer("overflow", false, false)}); err == nil {
		t.Fatal("unbounded uncertainty history accepted")
	}
	m.mu.Lock()
	retained = len(m.peers)
	joined := 0
	for _, p := range m.peers {
		if p.present && p.state.Joined {
			joined++
		}
	}
	m.mu.Unlock()
	if retained != 128 || joined != 0 || m.Snapshot().Camera.Known {
		t.Fatal("capacity failure retained vanished membership", retained, joined, m.Snapshot())
	}
	// Existing admitted slots still update while only excess new slots fail.
	if err := a.Report([]MeetingPeer{one, peer("overflow", false, false)}); err == nil {
		t.Fatal("overflow hidden")
	}
	m.mu.Lock()
	current := m.peers["test\x00"+one.ID].present
	m.mu.Unlock()
	if !current {
		t.Fatal("capacity failure prevented admitted source refresh")
	}
	requestMeeting(t, m, MeetingMute, false)
	if err := a.Report([]MeetingPeer{peer("overflow", false, false)}); err == nil {
		t.Fatal("new intent discarded required delayed-echo history")
	}
	// Matching readback resolves one retained uncertain slot and frees capacity
	// once it departs. Other old potential effects remain protected.
	one.Mute.Value, one.Mute.Revision = true, 2
	reportMeeting(t, a, one)
	reportMeeting(t, a)
	reportMeeting(t, a, peer("overflow", false, false))
	m.mu.Lock()
	retained = len(m.peers)
	m.mu.Unlock()
	if retained != 128 {
		t.Fatal("resolved slot did not release capacity", retained)
	}
}

func TestMeetingsControlReplacementBaselinesWithoutAdoptingValue(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("slot", false, false)
	one.Camera.Identity = 1
	reportMeeting(t, a, one)
	one.Camera.Identity = 2
	one.Camera.Revision++
	one.Camera.Value = true
	reportMeeting(t, a, one)
	if m.Snapshot().Camera.Value {
		t.Fatal("replacement button baseline adopted as external Camera On")
	}
	command := nextMeeting(t, a)
	if command.Property != MeetingCamera || command.Value {
		t.Fatal(command)
	}
}

func TestMeetingsOldEchoSurvivesNewIntentAndReconnect(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("slot", false, false)
	reportMeeting(t, a, one)
	requestMeeting(t, m, MeetingMute, true)
	old := nextMeeting(t, a)
	old.Complete(Outcome{Status: Unknown})
	reportMeeting(t, a)
	requestMeeting(t, m, MeetingMute, false)
	one.Lifetime = "reconnect"
	reportMeeting(t, a, one)
	one.Mute.Value = true
	one.Mute.Revision++
	reportMeeting(t, a, one)
	if m.Snapshot().Mute.Value {
		t.Fatal("retired old echo became new external intent")
	}
}

func TestMeetingsConfirmedResultWaitsForActualSnapshot(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			m, a := meetingRig(t, true)
			one := peer("slot", false, false)
			reportMeeting(t, a, one)
			requestMeeting(t, m, MeetingCamera, true)
			command := nextMeeting(t, a)
			command.Complete(Outcome{Status: Processed})
			if expired {
				m.mu.Lock()
				m.peers["test\x00slot"].properties[1].echo[1].deadline = time.Now().Add(-time.Second)
				m.mu.Unlock()
			}
			wantStatus := "Pending"
			if expired {
				wantStatus = "Not confirmed"
			}
			if got := m.Snapshot().Camera.Status; got != wantStatus {
				t.Fatalf("waiting status = %q, want %q", got, wantStatus)
			}
			for range 3 {
				reportMeeting(t, a, one)
				noMeeting(t, a)
			}
			one.Camera.Value = true
			one.Camera.Revision++
			reportMeeting(t, a, one)
			if got := m.Snapshot().Camera.Status; got != "" {
				t.Fatal("actual readback did not clear wait", got)
			}
			noMeeting(t, a)
		})
	}
}

func TestMeetingsControlReplacementCancelsCapturedUnsentWork(t *testing.T) {
	m, a := meetingRig(t, true)
	one := peer("slot", false, false)
	one.Camera.Identity = 1
	reportMeeting(t, a, one)
	requestMeeting(t, m, MeetingCamera, true)
	old := nextMeeting(t, a)
	one.Camera.Identity++
	one.Camera.Revision++
	reportMeeting(t, a, one)
	select {
	case <-old.Context.Done():
	default:
		t.Fatal("replaced control retained unsent work")
	}
	noMeeting(t, a) // The adapter still owns the old reservation.
	old.Complete(Outcome{Status: Rejected})
	replacement := nextMeeting(t, a)
	if replacement.Revision != one.Camera.Revision || !replacement.Value {
		t.Fatal(replacement)
	}
}

func TestMeetingsHostOwnsStartupAndFailedStopCleanup(t *testing.T) {
	var owner *Meetings
	stopFailure := errors.New("test plugin stop failed")
	plugin := Plugin{ID: "call", Start: func(ctx context.Context, s Services, _ json.RawMessage, _ map[string]Instance) (Instance, error) {
		owner = s.Meetings
		if owner == nil {
			t.Fatal("host omitted coordinator")
		}
		if _, err := owner.Attach(ctx, "call"); err != nil {
			t.Fatal("coordinator not started before provider", err)
		}
		return testInstance{stop: func() error { return stopFailure }}, nil
	}}
	host := New(Config{Plugins: map[string]PluginConfig{"call": {Enabled: true}}}, Services{Live: true}, plugin)
	host.Start(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := host.Stop(ctx); !errors.Is(err, stopFailure) {
		t.Fatal(err)
	}
	if owner == nil {
		t.Fatal("provider not started")
	}
	select {
	case <-owner.done:
	default:
		t.Fatal("failed plugin stop left coordinator running")
	}
	if _, err := owner.Attach(context.Background(), "late"); err == nil {
		t.Fatal("stopped host admitted new work")
	}
}
