package control

import (
	"path/filepath"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/snoofer"
)

func TestWorkerResultsFeedbackAndOriginNoticeClearing(t *testing.T) {
	saves := 0
	deps := Dependencies{Save: func(string, config.Config, *config.Intent, string) (string, error) {
		saves++
		return "saved", nil
	}}
	actions, states, _ := outcomeWorker(t, testRuleConfig(t), filepath.Join(t.TempDir(), "config"), deps)
	s := nextState(t, states)
	if result := actionOutcome(t, actions, Action{Kind: Edit, Revision: s.Revision, Row: "mic-mute", Value: "true"}); result.Status != snoofer.Processed {
		t.Fatal(result)
	}
	s = nextState(t, states)
	if !s.Intent.MicMuted || saves != 1 {
		t.Fatal("successful edit was not saved", s.Intent, saves)
	}
	if result := actionOutcome(t, actions, Action{Kind: Edit, Origin: "snoofer", Revision: s.Revision, Row: "mic-mute", Value: "not-bool"}); result.Status != snoofer.Rejected || result.Detail == "" {
		t.Fatal(result)
	}
	s = nextState(t, states)
	if !s.Intent.MicMuted || saves != 1 || s.Notice == "" || s.NoticeKind != NoticeError {
		t.Fatal("invalid edit changed intent or hid its error", s.Intent, saves, s.Notice)
	}
	if len(s.Feedback) != 1 || s.Feedback["mic-mute"].Kind != NoticeError {
		t.Fatal("action error was not scoped", s.Feedback)
	}
	notice := s.Notice
	if result := actionOutcome(t, actions, Refresh); result.Status != snoofer.Processed {
		t.Fatal(result)
	}
	s = nextState(t, states)
	if s.Notice != notice {
		t.Fatal("background refresh cleared the action notice", s.Notice)
	}
	if result := actionOutcome(t, actions, Action{Kind: refresh, Origin: "soundboard"}); result.Status != snoofer.Processed {
		t.Fatal(result)
	}
	s = nextState(t, states)
	if feedback, ok := s.Feedback["mic-mute"]; s.Notice != "" || !ok || feedback.Kind != NoticeError {
		t.Fatal("origin notice clearing changed scoped feedback", s.Notice, s.Feedback)
	}
	if result := actionOutcome(t, actions, Action{Kind: Edit, Origin: "discord", Revision: s.Revision, Row: "mic-mute", Value: "false"}); result.Status != snoofer.Processed {
		t.Fatal(result)
	}
	s = nextState(t, states)
	if len(s.Feedback) != 0 || s.Intent.MicMuted || saves != 2 {
		t.Fatal("successful retry retained error or failed to save", s.Feedback, s.Intent, saves)
	}
}
