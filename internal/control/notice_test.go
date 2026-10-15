package control

import (
	"testing"
	"time"

	"sound-snoofer/internal/routing"
)

func TestNoticeExpiryAndConvergence(t *testing.T) {
	now := time.Unix(100, 0)
	s := State{Connected: true, Plan: &routing.Plan{Topology: &routing.Topology{}}}
	s.SetNotice("Saved · Preview", NoticeSuccess, now)
	if s.Notice != "Saved · Preview" || s.NoticeKind != NoticeSuccess || !s.NoticeUntil.Equal(now.Add(3*time.Second)) {
		t.Fatal("success notice must expire after three seconds", s.Notice, s.NoticeKind, s.NoticeUntil)
	}
	s.Live = true
	s.SetNotice("Saved · Pending", NoticePending, now)
	s.Plan.Topology.Unresolved = []string{"unavailable"}
	s.ResolveNotice(now.Add(time.Minute))
	if s.Notice != "Saved · Pending" || s.NoticeKind != NoticePending || !s.NoticeUntil.IsZero() {
		t.Fatal("unresolved notice changed or expires", s.Notice, s.NoticeKind, s.NoticeUntil)
	}
	s.Plan.Topology.Unresolved = nil
	s.Error = "native write failed"
	s.ResolveNotice(now.Add(time.Minute))
	if s.Notice != "Saved · Pending" || s.Error != "native write failed" {
		t.Fatal("failure claimed applied", s.Notice, s.Error)
	}
	s.Error = ""
	s.ResolveNotice(now.Add(time.Minute))
	if s.Notice != "Applied" || s.NoticeKind != NoticeSuccess || !s.NoticeUntil.Equal(now.Add(time.Minute+3*time.Second)) {
		t.Fatal("not applied with success expiry", s.Notice, s.NoticeKind, s.NoticeUntil)
	}
	s.SetNotice("Save failed: disk full", NoticeError, now)
	if s.Notice != "Save failed: disk full" || s.NoticeKind != NoticeError || !s.NoticeUntil.IsZero() {
		t.Fatal("error notice changed or expires", s.Notice, s.NoticeKind, s.NoticeUntil)
	}
	s.Live = false
	s.SetNotice("Saved · Pending", NoticePending, now)
	s.ResolveNotice(now)
	if s.Notice != "Saved · Preview" {
		t.Fatal("preview claimed applied")
	}
}
