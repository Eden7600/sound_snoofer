package tui

import (
	"testing"
	"time"

	"sound-snoofer/internal/routing"
)

func TestNoticeExpiryAndConvergence(t *testing.T) {
	now := time.Unix(100, 0)
	s := State{Connected: true, Plan: &routing.Plan{Topology: &routing.Topology{}}}
	s.SetNotice("Saved · Preview", noticeSuccess, now)
	if s.NoticeAt(now.Add(2*time.Second)) == "" || s.NoticeAt(now.Add(3*time.Second)) != "" {
		t.Fatal("expiry")
	}
	s.Live = true
	s.SetNotice("Saved · Pending", noticePending, now)
	s.Plan.Topology.Unresolved = []string{"unavailable"}
	s.ResolveNotice(now.Add(time.Minute))
	if s.NoticeAt(now.Add(time.Minute)) != "Saved · Pending" {
		t.Fatal("unresolved claimed applied")
	}
	s.Plan.Topology.Unresolved = nil
	s.ResolveNotice(now.Add(time.Minute))
	if s.NoticeAt(now.Add(time.Minute)) != "Applied" {
		t.Fatal("not applied")
	}
	s.Error = "native write failed"
	if s.NoticeAt(now.Add(2*time.Minute)) != s.Error {
		t.Fatal("expiry hid failure")
	}
	s.Error = ""
	s.SetNotice("Save failed: disk full", noticeError, now)
	if s.NoticeAt(now.Add(time.Hour)) == "" {
		t.Fatal("error expired")
	}
	s.Live = false
	s.SetNotice("Saved · Pending", noticePending, now)
	s.ResolveNotice(now)
	if s.Notice != "Saved · Preview" {
		t.Fatal("preview claimed applied")
	}
}
