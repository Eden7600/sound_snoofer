package control

import (
	"fmt"
	"testing"
	"time"
)

func TestFeedbackLifecycle(t *testing.T) {
	now := time.Unix(100, 0)
	s := State{}
	action := Action{Kind: Edit, Row: "monitor"}
	s.actionFeedback(action, NoticeError, now)
	published := s.Feedback
	s.actionFeedback(action, NoticePending, now.Add(time.Second))
	if published["monitor"].Kind != NoticeError || s.Feedback["monitor"].Kind != NoticePending {
		t.Fatal("published feedback mutated or retry not pending")
	}
	s.actionFeedback(action, NoticeSuccess, now.Add(2*time.Second))
	if len(s.Feedback) != 0 {
		t.Fatal("success retained failure")
	}
	s.actionFeedback(Action{Kind: RecordStop}, NoticeError, now)
	if len(s.Feedback) != 2 {
		t.Fatal("stop transport aliases missing")
	}
	s.pruneFeedback(now.Add(10 * time.Second))
	if len(s.Feedback) != 0 {
		t.Fatal("feedback did not expire")
	}
	for i := 0; i < 100; i++ {
		s.actionFeedback(Action{Kind: Edit, Row: fmt.Sprint(i)}, NoticeError, now)
	}
	if len(s.Feedback) > 64 {
		t.Fatal("unbounded feedback")
	}
}
