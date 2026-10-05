package tui

import (
	"sound-snoofer/internal/config"
)

func (s screen) choices() *config.Intent {
	if s.draft != nil {
		return s.draft
	}
	return s.state.Intent
}

func (s *screen) queueEdit(action Action) {
	if s.choices() == nil {
		return
	}
	if len(s.deferredEdits) >= maxDeferredEdits {
		s.pending = "Settings queue full; wait for saved choices"
		return
	}
	next := s.choices().Clone()
	if err := editIntent(next, action); err != nil {
		s.pending = err.Error()
		return
	}
	if s.inflight == 0 {
		s.nextEditID++
		action.ID = s.nextEditID
		action.Revision = s.state.Revision
		select {
		case s.actions <- action:
			s.inflight = action.ID
			s.inflightAction = action
		default:
			s.pending = "A command is already queued"
			return
		}
	} else {
		s.deferredEdits = append(s.deferredEdits, settingEdit{Row: action.Row, Value: action.Value})
	}
	s.draft = next
	s.pending = "Queued"
}

func (s *screen) acceptEditState() {
	if s.inflight == 0 {
		return
	}
	if s.state.EditAck != s.inflight {
		s.pending = "Queued"
		return
	}
	s.inflight = 0
	if s.state.EditError != "" {
		s.draft = nil
		s.deferredEdits = nil
		s.picker = nil
		s.pending = "Settings rejected: " + s.state.EditError
		return
	}
	next := s.state.Intent.Clone()
	if len(s.deferredEdits) != 0 && next != nil {
		s.nextEditID++
		action := Action{Kind: editRule, Revision: s.state.Revision, ID: s.nextEditID, Edits: append([]settingEdit(nil), s.deferredEdits...)}
		if err := editBatch(next, action); err != nil {
			s.pending = "Settings rejected: " + err.Error()
		} else {
			select {
			case s.actions <- action:
				s.inflight = action.ID
				s.inflightAction = action
				s.pending = "Queued"
			default:
				s.pending = "Settings not submitted: command queue full"
			}
		}
	}
	s.deferredEdits = nil
	s.draft = nil
	if s.inflight != 0 {
		s.draft = next
	}
	if s.picker != nil && s.choices() != nil {
		s.picker.revision = s.state.Revision
		s.picker.current = s.choiceValue(s.picker.row)
	}
}
