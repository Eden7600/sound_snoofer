package control

import "time"

// Feedback is transient action feedback for one binding, not mixer health.
type Feedback struct {
	Kind  NoticeKind
	Until time.Time
}

func actionBindings(a Action) []string {
	switch a.Kind {
	case editRule:
		if len(a.Edits) == 0 {
			return []string{a.Row}
		}
		keys := make([]string, 0, len(a.Edits))
		for _, edit := range a.Edits {
			keys = append(keys, edit.Row)
		}
		return keys
	case startRecording:
		return []string{"record-start"}
	case stopRecording:
		return []string{"record-stop", "snippet-stop"}
	case playSnippet:
		return []string{"snippet-play"}
	case restartEngine:
		return []string{"engine-restart"}
	case gain:
		return []string{"gain:" + a.Target}
	}
	return nil
}

func (s *State) pruneFeedback(now time.Time) {
	next := make(map[string]Feedback, len(s.Feedback))
	for key, value := range s.Feedback {
		if now.Before(value.Until) {
			next[key] = value
		}
	}
	s.Feedback = next
}

func (s *State) actionFeedback(a Action, kind NoticeKind, now time.Time) {
	s.pruneFeedback(now)
	for _, key := range actionBindings(a) {
		// Bindings submitted over the controls pipe must not grow state without bound.
		if key == "" {
			continue
		}
		delete(s.Feedback, key)
		if kind != NoticeSuccess && len(s.Feedback) < 64 {
			s.Feedback[key] = Feedback{Kind: kind, Until: now.Add(10 * time.Second)}
		}
	}
}
