package snoofer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MeetingProperty identifies the two shared call preferences.
type MeetingProperty string

// The shared mute value means muted; the camera value means camera enabled.
const (
	MeetingMute   MeetingProperty = "mute"
	MeetingCamera MeetingProperty = "camera"
)

// MeetingValue is an adapter's observed state, never its requested state.
type MeetingValue struct {
	Known, Value, Available bool
	Revision                uint64
	Identity                uint64 // Changes when the observed control is replaced.
}

// MeetingPeer separates a stable source slot from its exact writable lifetime.
type MeetingPeer struct {
	ID, Lifetime, Label string
	Joined, NewCall     bool
	Mute, Camera        MeetingValue
}

// MeetingCommand is one claimed explicit write owned by the adapter worker.
type MeetingCommand struct {
	Context        context.Context
	Peer, Lifetime string
	Property       MeetingProperty
	Revision       uint64
	Value          bool
	TargetRevision uint64
	Complete       Completion
}

// MeetingTarget describes accepted intent and current application status.
type MeetingTarget struct {
	Known, Value bool
	Revision     uint64
	Status       string
}

// MeetingSnapshot is an independent copy of both shared targets.
type MeetingSnapshot struct{ Mute, Camera MeetingTarget }

// MeetingMicPreference is audio's saved mute authority and observed apply state.
type MeetingMicPreference struct {
	Known, Muted, Applied bool
	Revision              uint64
}

// MeetingMicWrite queues the existing revision-checked audio save operation.
type MeetingMicWrite func(bool, uint64, Completion) error

// Meetings owns shared intent. Provider workers retain all native/transport I/O.
type Meetings struct {
	mu           sync.Mutex
	controls     *Controls
	live         bool
	ctx          context.Context
	cancel       context.CancelFunc
	done         chan struct{}
	wake         chan struct{}
	closed       bool
	adapters     map[string]*MeetingAdapter
	peers        map[string]*meetingPeer
	targets      [2]MeetingTarget
	nextRevision uint64
	audio        *MeetingAudio
	audioSave    *meetingSave
	audioNext    *meetingSave
	audioError   string
	completions  []meetingCompletion
}

// MeetingAdapter is one provider attachment; its worker owns claimed commands.
type MeetingAdapter struct {
	owner    *Meetings
	provider string
	ctx      context.Context
	changed  chan struct{}
	closed   bool
	stop     func() bool
}

// MeetingAudio registers the single optional persistent mute authority.
type MeetingAudio struct {
	cancel     context.CancelFunc
	owner      *Meetings
	ctx        context.Context
	write      MeetingMicWrite
	preference MeetingMicPreference
	closed     bool
	stop       func() bool
}

type meetingPeer struct {
	adapter    *MeetingAdapter
	state      MeetingPeer
	present    bool
	properties [2]meetingProperty
}

type meetingProperty struct {
	pending *meetingAttempt
	echo    [2]*meetingAttempt
	fenced  bool
	blocked uint64
	epoch   uint64
}

type meetingAttempt struct {
	peer               *meetingPeer
	property           int
	lifetime           string
	intent             uint64
	peerEpoch          uint64
	revision, identity uint64
	want               bool
	cancel             context.CancelFunc
	deadline           time.Time
	done, observed     bool
	confirmed          bool
}

type meetingSave struct {
	ctx      context.Context
	audio    *MeetingAudio
	want     bool
	revision uint64
	complete Completion
	started  bool
	result   *Outcome
	deadline time.Time
}

type meetingCompletion struct {
	complete Completion
	outcome  Outcome
}

// NewMeetings is inert; the host starts it before its providers.
func NewMeetings(controls *Controls, live bool) *Meetings {
	if controls == nil {
		controls = NewControls()
	}
	return &Meetings{controls: controls, live: live, done: make(chan struct{}), wake: make(chan struct{}, 1), adapters: map[string]*MeetingAdapter{}, peers: map[string]*meetingPeer{}}
}

// Start owns the publish/completion worker until its context ends or Close runs.
func (m *Meetings) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("meeting state stopped")
	}
	if m.ctx != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	go m.run()
	return nil
}

func (m *Meetings) run() {
	defer close(m.done)
	defer m.controls.Remove("meetings")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		m.tick(time.Now())
		m.publish()
		select {
		case <-m.ctx.Done():
			m.stop()
			return
		case <-m.wake:
		case <-ticker.C:
		}
	}
}

// Close cancels work and joins the owner within the supplied deadline.
func (m *Meetings) Close(ctx context.Context) error {
	m.mu.Lock()
	started := m.ctx != nil
	if !started {
		m.closed = true
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	if !started {
		return nil
	}
	select {
	case <-m.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Meetings) stop() {
	m.mu.Lock()
	m.closed = true
	completions := m.completions
	m.completions = nil
	for _, p := range m.peers {
		for n := range p.properties {
			if a := p.properties[n].pending; a != nil {
				a.cancel()
			}
		}
	}
	for _, s := range []*meetingSave{m.audioSave, m.audioNext} {
		if s != nil && s.complete != nil {
			completions = append(completions, meetingCompletion{s.complete, Outcome{Status: Unknown, Detail: "Meeting state stopped before saving"}})
			s.complete = nil
		}
	}
	m.mu.Unlock()
	for _, c := range completions {
		c.complete(c.outcome)
	}
}

func notifyMeeting(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (m *Meetings) changedLocked() {
	notifyMeeting(m.wake)
	for _, a := range m.adapters {
		notifyMeeting(a.changed)
	}
}

// Attach admits one uniquely named provider without starting native operations.
func (m *Meetings) Attach(ctx context.Context, provider string) (*MeetingAdapter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !validID(provider) || m.adapters[provider] != nil {
		return nil, errors.New("invalid or duplicate meeting provider")
	}
	if m.ctx == nil || m.closed || m.ctx.Err() != nil || ctx.Err() != nil {
		return nil, errors.New("meeting state unavailable")
	}
	a := &MeetingAdapter{owner: m, provider: provider, ctx: ctx, changed: make(chan struct{}, 1)}
	m.adapters[provider] = a
	a.stop = context.AfterFunc(ctx, a.Close)
	return a, nil
}

// Changed signals that the worker should report fresh state and claim available work.
// Signals coalesce; receiving one does not imply a command exists.
func (a *MeetingAdapter) Changed() <-chan struct{} { return a.changed }

// Close retires participation while preserving unresolved effects across reconnect.
func (a *MeetingAdapter) Close() {
	m := a.owner
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.closed {
		return
	}
	a.closed = true
	if a.stop != nil {
		a.stop()
	}
	if m.adapters[a.provider] == a {
		delete(m.adapters, a.provider)
	}
	for _, p := range m.peers {
		if p.adapter == a {
			p.present = false
			m.cancelPeerLocked(p)
		}
	}
	m.cameraPeriodLocked()
	m.pruneRetiredLocked()
	m.changedLocked()
}

func meetingIndex(property MeetingProperty) (int, bool) {
	switch property {
	case MeetingMute:
		return 0, true
	case MeetingCamera:
		return 1, true
	}
	return 0, false
}

func meetingPropertyName(n int) MeetingProperty {
	if n == 0 {
		return MeetingMute
	}
	return MeetingCamera
}
func meetingObserved(p MeetingPeer, n int) MeetingValue {
	if n == 0 {
		return p.Mute
	}
	return p.Camera
}
func meetingBool(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (m *Meetings) cancelPeerLocked(p *meetingPeer) {
	for n := range p.properties {
		if attempt := p.properties[n].pending; attempt != nil {
			attempt.cancel()
		}
	}
}

// Report synchronously replaces this provider's active peer snapshot.
// It can refuse excess new slots while still refreshing admitted and absent peers.
func (a *MeetingAdapter) Report(peers []MeetingPeer) error {
	if len(peers) > 64 {
		return errors.New("too many meeting peers")
	}
	seen := map[string]MeetingPeer{}
	for _, peer := range peers {
		if peer.ID == "" || len(peer.ID) > 256 || peer.Lifetime == "" || len(peer.Lifetime) > 512 || seen[peer.ID].ID != "" {
			return errors.New("invalid meeting peer identity")
		}
		seen[peer.ID] = peer
		for _, value := range []MeetingValue{peer.Mute, peer.Camera} {
			if value.Available && (!value.Known || value.Revision == 0) {
				return errors.New("invalid meeting observation")
			}
		}
	}
	peers = append([]MeetingPeer(nil), peers...)
	sort.Slice(peers, func(i, j int) bool { return peers[i].ID < peers[j].ID })
	m := a.owner
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.closed || a.ctx.Err() != nil || m.closed {
		return errors.New("meeting provider stopped")
	}
	changed := false
	for _, p := range m.peers {
		incoming, present := seen[p.state.ID]
		replacedCall := present && incoming.NewCall && incoming.Lifetime != p.state.Lifetime
		if p.adapter == a && p.present && (!present || replacedCall) {
			p.present = false
			changed = true
			m.cancelPeerLocked(p)
		}
	}
	// Remove departed calls before new baselines can inherit the old camera period.
	m.cameraPeriodLocked()
	m.pruneRetiredLocked()
	retained := 0
	for _, p := range m.peers {
		if p.adapter.provider == a.provider {
			retained++
		}
	}
	var capacityErr error
	for _, state := range peers {
		key := a.provider + "\x00" + state.ID
		p := m.peers[key]
		if p == nil {
			if retained >= 128 {
				capacityErr = errors.New("meeting source capacity reached; unresolved commands retained")
				continue
			}
			p = &meetingPeer{}
			m.peers[key] = p
			retained++
		}
		old, established := p.state, p.present && p.adapter == a && p.state.Lifetime == state.Lifetime
		newCall := state.NewCall && (!established || !old.Joined)
		changed = changed || !established || state != old
		if !established {
			m.cancelPeerLocked(p)
		}
		p.adapter, p.state, p.present = a, state, true
		for n := range p.properties {
			property := &p.properties[n]
			if newCall {
				property.epoch++
				property.fenced, property.blocked = false, 0
				property.echo = [2]*meetingAttempt{}
			}
			current, previous := meetingObserved(state, n), meetingObserved(old, n)
			if current.Identity != previous.Identity && property.pending != nil {
				property.pending.cancel()
			}
			if !state.Joined || !current.Known || !current.Available {
				continue
			}
			echo := property.echo[meetingBool(current.Value)]
			own := echo != nil
			if own {
				echo.observed = true
				if echo.intent == m.targets[n].Revision && echo.peerEpoch == property.epoch {
					property.fenced, property.blocked = false, 0
				}
				// Consume an own effect only when it is actually observed changing.
				if !established || !previous.Known || previous.Value != current.Value {
					property.echo[meetingBool(current.Value)] = nil
				}
			}
			if !m.targets[n].Known {
				if n == 1 || m.audio == nil || !m.audio.preference.Known {
					m.setTargetLocked(n, current.Value)
				}
			} else if established && old.Joined && previous.Known && previous.Available && previous.Identity == current.Identity && previous.Value != current.Value && !own {
				m.externalLocked(n, current.Value)
			}
		}
	}
	m.cameraPeriodLocked()
	if changed {
		m.changedLocked()
	}
	return capacityErr
}

// Retired slots retain live operations and unobserved possible effects, including
// older intent. A new target releases a write fence, not delayed-echo identity.
// Admission is capped instead of forgetting a fence and replaying on reconnect.
func (m *Meetings) pruneRetiredLocked() {
	for key, p := range m.peers {
		if p.present {
			continue
		}
		keep := false
		for _, property := range p.properties {
			keep = keep || property.pending != nil || property.fenced
			for _, echo := range property.echo {
				keep = keep || echo != nil && !echo.observed
			}
		}
		if !keep {
			delete(m.peers, key)
		}
	}
}

func (m *Meetings) cameraPeriodLocked() {
	for _, p := range m.peers {
		if p.present && p.state.Joined {
			return
		}
	}
	if m.targets[1].Known {
		m.nextRevision++
		m.targets[1] = MeetingTarget{Revision: m.nextRevision}
		for _, p := range m.peers {
			if attempt := p.properties[1].pending; attempt != nil {
				attempt.cancel()
			}
			p.properties[1].echo = [2]*meetingAttempt{}
			p.properties[1].fenced = false
			p.properties[1].blocked = 0
		}
	}
}

func (m *Meetings) setTargetLocked(n int, value bool) {
	m.nextRevision++
	m.targets[n] = MeetingTarget{Known: true, Value: value, Revision: m.nextRevision}
	for _, p := range m.peers {
		property := &p.properties[n]
		property.fenced, property.blocked = false, 0
		if property.pending != nil {
			property.pending.cancel()
		}
	}
	m.pruneRetiredLocked()
	m.changedLocked()
}

// Next reserves one peer/property until its terminal completion or deadline.
// Cancellation invalidates unsent work but does not release a claimed reservation.
func (a *MeetingAdapter) Next() (MeetingCommand, bool) {
	m := a.owner
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.live || a.closed || a.ctx.Err() != nil || m.closed || m.ctx.Err() != nil {
		return MeetingCommand{}, false
	}
	keys := make([]string, 0, len(m.peers))
	for key, p := range m.peers {
		if p.adapter == a && p.present && p.state.Joined {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		p := m.peers[key]
		for n := range p.properties {
			target, value, property := m.targets[n], meetingObserved(p.state, n), &p.properties[n]
			echo := property.echo[meetingBool(target.Value)]
			waitingReadback := echo != nil && echo.intent == target.Revision && echo.confirmed && !echo.observed
			if !target.Known || !value.Known || !value.Available || value.Value == target.Value || property.pending != nil || property.fenced || property.blocked == target.Revision || waitingReadback {
				continue
			}
			if n == 0 && (m.audioSave != nil || m.audioNext != nil || m.audioError != "") {
				continue
			}
			ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
			attempt := &meetingAttempt{peer: p, property: n, lifetime: p.state.Lifetime, intent: target.Revision, peerEpoch: property.epoch, revision: value.Revision, identity: value.Identity, want: target.Value, cancel: cancel, deadline: time.Now().Add(5 * time.Second)}
			property.pending = attempt
			property.echo[meetingBool(target.Value)] = attempt
			m.changedLocked()
			return MeetingCommand{Context: ctx, Peer: p.state.ID, Lifetime: p.state.Lifetime, Property: meetingPropertyName(n), Revision: value.Revision, Value: target.Value, TargetRevision: target.Revision,
				Complete: func(outcome Outcome) { m.finishAttempt(attempt, outcome) }}, true
		}
	}
	return MeetingCommand{}, false
}

func (m *Meetings) finishAttempt(a *meetingAttempt, outcome Outcome) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finishAttemptLocked(a, outcome)
}

func (m *Meetings) finishAttemptLocked(a *meetingAttempt, outcome Outcome) {
	if a.done {
		return
	}
	a.done = true
	a.cancel()
	property := &a.peer.properties[a.property]
	if property.pending == a {
		property.pending = nil
	}
	current := m.targets[a.property].Revision == a.intent && property.epoch == a.peerEpoch
	if outcome.Status == Processed {
		// Transport confirmation can precede its snapshot. Keep the expected
		// echo and inhibit replay until the actual report catches up.
		a.confirmed = true
	}
	if outcome.Status != Processed {
		observed := meetingObserved(a.peer.state, a.property)
		stillCaptured := a.peer.state.Lifetime == a.lifetime && observed.Revision == a.revision && observed.Identity == a.identity
		if current && !a.observed && (outcome.Status != Rejected || stillCaptured) {
			property.blocked = a.intent
		}
		if outcome.Status == Rejected {
			if property.echo[meetingBool(a.want)] == a {
				property.echo[meetingBool(a.want)] = nil
			}
		} else if current && !a.observed {
			property.fenced = true
		}
	}
	m.pruneRetiredLocked()
	m.changedLocked()
}

// Snapshot returns accepted targets separately from adapter observations.
func (m *Meetings) Snapshot() MeetingSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked()
}

func (m *Meetings) snapshotLocked() MeetingSnapshot {
	targets := m.targets
	for n := range targets {
		for _, p := range m.peers {
			if !p.present || !p.state.Joined {
				continue
			}
			property := p.properties[n]
			if echo := property.echo[meetingBool(targets[n].Value)]; echo != nil && echo.intent == targets[n].Revision && echo.confirmed && !echo.observed {
				targets[n].Status = "Pending"
				if !time.Now().Before(echo.deadline) {
					targets[n].Status = "Not confirmed"
					break
				}
			}
			if property.pending != nil && property.pending.intent == targets[n].Revision {
				targets[n].Status = "Pending"
			}
			if property.fenced || property.blocked == targets[n].Revision && property.blocked != 0 {
				targets[n].Status = "Not confirmed"
				break
			}
		}
	}
	if m.audioSave != nil || m.audioNext != nil {
		targets[0].Status = "Pending"
	}
	if m.audioError != "" {
		targets[0].Status = "Not confirmed"
	}
	return MeetingSnapshot{Mute: targets[0], Camera: targets[1]}
}

// Request accepts a captured shared-target edit; mute uses audio's save boundary
// when attached. Completion does not imply every app has observed the target.
func (m *Meetings) Request(ctx context.Context, property MeetingProperty, revision uint64, value bool, complete Completion) error {
	n, ok := meetingIndex(property)
	if !ok {
		return errors.New("invalid meeting property")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	if !m.live || m.closed || m.ctx == nil || m.ctx.Err() != nil || !m.targets[n].Known || m.targets[n].Revision != revision {
		m.mu.Unlock()
		return errors.New("meeting target changed or unavailable")
	}
	if n == 0 && m.audio != nil && m.audio.preference.Known {
		if m.audioSave != nil || m.audioNext != nil {
			m.mu.Unlock()
			return errors.New("meeting mute save pending")
		}
		m.audioError = ""
		m.audioNext = &meetingSave{ctx: ctx, audio: m.audio, want: value, revision: m.audio.preference.Revision, complete: complete}
		m.changedLocked()
		m.mu.Unlock()
		return nil
	}
	m.setTargetLocked(n, value)
	m.mu.Unlock()
	if complete != nil {
		complete(Outcome{Status: Processed, Detail: "Meeting target accepted; apps are confirming"})
	}
	return nil
}

func (m *Meetings) publish() {
	snapshot := m.Snapshot()
	controls := make([]Control, 0, 2)
	for n, target := range []MeetingTarget{snapshot.Mute, snapshot.Camera} {
		property, label, icon := MeetingMute, "Mute", "mic-mute"
		if n == 1 {
			property, label, icon = MeetingCamera, "Camera", "discord-video"
		}
		if target.Known && (n == 0 && target.Value || n == 1 && !target.Value) {
			if n == 0 {
				icon = "mic-mute-muted"
			} else {
				icon = "discord-video-off"
			}
		}
		value := "N/A"
		if target.Known {
			value = OnOff(target.Value)
		}
		controls = append(controls, Control{ID: "meetings." + string(property), Label: label, ShortLabel: label, Group: "Meetings", Kind: "toggle", Icon: icon, Value: value, Status: target.Status, Available: m.live && target.Known, Epoch: target.Revision, Operations: []string{"press"}})
	}
	_ = m.controls.Publish("meetings", controls, func(ctx context.Context, r Request) error {
		property, target := MeetingMute, snapshot.Mute
		if r.ID == "meetings.camera" {
			property, target = MeetingCamera, snapshot.Camera
		}
		if r.Operation != "press" {
			return fmt.Errorf("invalid meeting operation")
		}
		return m.Request(ctx, property, target.Revision, !target.Value, r.ClaimCompletion(m.ctx))
	})
}
