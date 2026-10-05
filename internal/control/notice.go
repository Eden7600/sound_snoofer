package control

import "time"

type NoticeKind int

const (
	NoticeError NoticeKind = iota
	NoticePending
	NoticeSuccess
)

func (s *State) SetNotice(text string, kind NoticeKind, now time.Time) {
	s.Notice = text
	s.NoticeKind = kind
	s.NoticeUntil = time.Time{}
	if kind == NoticeSuccess {
		s.NoticeUntil = now.Add(3 * time.Second)
	}
}

func (s State) NoticeAt(now time.Time) string {
	if s.StateError != "" {
		return s.StateError
	}
	if s.Error != "" {
		return s.Error
	}
	if !s.NoticeUntil.IsZero() && !now.Before(s.NoticeUntil) {
		return ""
	}
	return s.Notice
}

func (s *State) ResolveNotice(now time.Time) {
	if s.NoticeKind != NoticePending || s.Plan == nil || s.Error != "" || s.StateError != "" {
		return
	}
	if !s.Live {
		s.SetNotice("Saved · Preview", NoticeSuccess, now)
	} else if s.Connected && !s.Plan.HasChanges() && !s.Plan.HasUnresolved() {
		s.SetNotice("Applied", NoticeSuccess, now)
	}
}
