package tui

import (
	"testing"
	"time"

	"voice-snooter/internal/routing"
)

func TestNoticeExpiryAndConvergence(t *testing.T) {
	now := time.Unix(100, 0)
	s := State{Connected: true, Plan: &routing.Plan{Topology: &routing.Topology{}}}
	s.setNotice("Saved · Preview", noticeSuccess, now)
	if s.noticeAt(now.Add(2*time.Second)) == "" || s.noticeAt(now.Add(3*time.Second)) != "" {
		t.Fatal("expiry")
	}
	s.Live = true
	s.setNotice("Saved · Pending", noticePending, now)
	s.Plan.Topology.Unresolved = []string{"unavailable"}
	s.resolveNotice(now.Add(time.Minute))
	if s.noticeAt(now.Add(time.Minute)) != "Saved · Pending" {
		t.Fatal("unresolved claimed applied")
	}
	s.Plan.Topology.Unresolved = nil
	s.resolveNotice(now.Add(time.Minute))
	if s.noticeAt(now.Add(time.Minute)) != "Applied" {
		t.Fatal("not applied")
	}
	s.Error = "native write failed"
	if s.noticeAt(now.Add(2*time.Minute)) != s.Error {
		t.Fatal("expiry hid failure")
	}
	s.Error = ""
	s.setNotice("Save failed: disk full", noticeError, now)
	if s.noticeAt(now.Add(time.Hour)) == "" {
		t.Fatal("error expired")
	}
	s.Live = false
	s.setNotice("Saved · Pending", noticePending, now)
	s.resolveNotice(now)
	if s.Notice != "Saved · Preview" {
		t.Fatal("preview claimed applied")
	}
}
