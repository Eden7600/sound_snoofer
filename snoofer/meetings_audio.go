package snoofer

import (
	"context"
	"errors"
	"time"
)

// AttachAudio registers the existing audio save path without making it a dependency.
func (m *Meetings) AttachAudio(ctx context.Context, write MeetingMicWrite) (*MeetingAudio, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ctx == nil || m.closed || m.ctx.Err() != nil || ctx.Err() != nil || m.audio != nil || write == nil {
		return nil, errors.New("meeting audio authority unavailable")
	}
	audioCtx, cancel := context.WithCancel(ctx)
	a := &MeetingAudio{owner: m, ctx: audioCtx, cancel: cancel, write: write}
	m.audio = a
	a.stop = context.AfterFunc(ctx, a.Close)
	return a, nil
}

// Close releases persistent authority while retaining the last runtime mute target.
func (a *MeetingAudio) Close() {
	m := a.owner
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.closed {
		return
	}
	a.closed = true
	a.cancel()
	if a.stop != nil {
		a.stop()
	}
	if m.audio == a {
		m.audio = nil
	}
	for _, s := range []*meetingSave{m.audioSave, m.audioNext} {
		if s != nil && s.audio == a {
			m.completeSaveLocked(s, Outcome{Status: Unknown, Detail: "Audio stopped before mute was saved"})
		}
	}
	if m.audioSave != nil && m.audioSave.audio == a {
		m.audioSave = nil
	}
	if m.audioNext != nil && m.audioNext.audio == a {
		m.audioNext = nil
	}
	m.audioError = ""
	m.changedLocked()
}

// Report publishes the saved preference; native apply status never replaces intent.
func (a *MeetingAudio) Report(preference MeetingMicPreference) {
	m := a.owner
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.closed || m.audio != a || m.closed || preference.Revision < a.preference.Revision {
		return
	}
	previous := a.preference
	if preference == previous {
		return
	}
	ownReadback := false
	if save := m.audioSave; save != nil && save.audio == a {
		ownReadback = save.started && previous.Known && preference.Known && previous.Muted != preference.Muted && preference.Muted == save.want && preference.Revision > save.revision
	}
	if next := m.audioNext; next != nil && next.audio == a && preference.Revision != next.revision {
		if ownReadback {
			// A later external intent waits behind our preceding save, and must
			// capture that save's observed revision before it can run.
			next.revision = preference.Revision
		} else {
			m.completeSaveLocked(next, Outcome{Status: Rejected, Detail: "Audio preference changed before the meeting request"})
			m.audioNext = nil
		}
	}
	a.preference = preference
	if preference.Known && (!previous.Known || !m.targets[0].Known || preference.Muted != m.targets[0].Value) {
		m.audioError = ""
		m.setTargetLocked(0, preference.Muted)
	}
	m.changedLocked()
}

func (m *Meetings) externalLocked(n int, value bool) {
	if !m.live {
		return
	}
	if n == 0 && m.audio != nil && m.audio.preference.Known {
		if m.audioNext != nil {
			m.completeSaveLocked(m.audioNext, Outcome{Status: Rejected, Detail: "A newer meeting mute change replaced the request"})
		}
		m.audioNext = &meetingSave{ctx: m.ctx, audio: m.audio, want: value, revision: m.audio.preference.Revision}
		m.audioError = ""
		m.changedLocked()
		return
	}
	m.setTargetLocked(n, value)
}

func (m *Meetings) completeSaveLocked(s *meetingSave, outcome Outcome) {
	if s.complete != nil {
		m.completions = append(m.completions, meetingCompletion{s.complete, outcome})
		s.complete = nil
	}
}

func (m *Meetings) audioFinished(s *meetingSave, outcome Outcome) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.audioSave != s || s.result != nil {
		return
	}
	s.result = &outcome
	m.changedLocked()
}

func (m *Meetings) tick(now time.Time) {
	m.mu.Lock()
	m.tickActionsLocked(now)
	for _, p := range m.peers {
		for n := range p.properties {
			if a := p.properties[n].pending; a != nil && !now.Before(a.deadline) {
				m.finishAttemptLocked(a, Outcome{Status: Unknown, Detail: "Meeting application was not confirmed"})
			}
		}
	}
	if s := m.audioSave; s != nil {
		if s.result == nil && !now.Before(s.deadline) {
			outcome := Outcome{Status: Unknown, Detail: "Mute save was not confirmed"}
			s.result = &outcome
		}
		if s.result != nil {
			outcome := *s.result
			m.completeSaveLocked(s, outcome)
			if outcome.Status != Processed {
				m.audioError = outcome.Detail
				if m.audioError == "" {
					m.audioError = "Mute save was not confirmed"
				}
				m.audioSave = nil
			} else if s.audio.preference.Known && s.audio.preference.Muted == s.want && s.audio.preference.Revision >= s.revision {
				m.audioSave = nil
				m.audioError = ""
			} else if !now.Before(s.deadline) {
				m.audioSave = nil
				m.audioError = "Saved mute preference was not observed"
			}
		}
	}
	var send *meetingSave
	if m.audioSave == nil && m.audioNext != nil {
		s := m.audioNext
		m.audioNext = nil
		if !m.live || m.closed || m.ctx.Err() != nil || s.ctx.Err() != nil || m.audio != s.audio || s.audio.closed || !s.audio.preference.Known || s.revision != s.audio.preference.Revision {
			m.completeSaveLocked(s, Outcome{Status: Rejected, Detail: "Audio mute authority changed"})
		} else if s.audio.preference.Muted == s.want {
			m.audioError = ""
			m.setTargetLocked(0, s.want)
			m.completeSaveLocked(s, Outcome{Status: Processed, Detail: "Mic mute preference already saved; apps are confirming"})
		} else {
			s.started = true
			s.deadline = now.Add(5 * time.Second)
			m.audioSave = s
			send = s
		}
	}
	completions := m.completions
	m.completions = nil
	m.mu.Unlock()
	for _, c := range completions {
		c.complete(c.outcome)
	}
	if send != nil {
		if m.ctx.Err() != nil || send.ctx.Err() != nil || send.audio.ctx.Err() != nil {
			m.audioFinished(send, Outcome{Status: Rejected, Detail: "Mute request canceled before saving"})
			return
		}
		if err := send.audio.write(send.want, send.revision, func(outcome Outcome) { m.audioFinished(send, outcome) }); err != nil {
			m.audioFinished(send, Outcome{Status: Rejected, Detail: err.Error()})
		}
	}
}
