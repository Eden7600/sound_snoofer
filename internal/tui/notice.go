package tui

import "time"

type noticeKind int

const (
	noticeError noticeKind = iota
	noticePending
	noticeSuccess
)

func (s *State) setNotice(text string, kind noticeKind, now time.Time) {
	s.Notice = text
	s.NoticeKind = kind
	s.NoticeUntil = time.Time{}
	if kind == noticeSuccess {
		s.NoticeUntil = now.Add(3 * time.Second)
	}
}

func (s State) noticeAt(now time.Time) string {
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

func (s *State) resolveNotice(now time.Time) {
	if s.NoticeKind != noticePending || s.Plan == nil || s.Error != "" || s.StateError != "" {
		return
	}
	if !s.Live {
		s.setNotice("Saved · Preview", noticeSuccess, now)
	} else if s.Connected && !s.Plan.HasChanges() && !s.Plan.HasUnresolved() {
		s.setNotice("Applied", noticeSuccess, now)
	}
}
