package app

import (
	"strconv"

	"sound-snoofer/snoofer"
)

type enablementProposal struct {
	id, source, expected, text string
	owner                      *actionSession
	config                     snoofer.Config
}

// The tray actor owns the proposal; an abandoned child never leaves it for a new one.
type lifecycle struct {
	path     string
	host     *snoofer.Host
	next     uint64
	proposal *enablementProposal
}

func (l *lifecycle) pending() *enablementProposal {
	if l.proposal != nil && l.proposal.owner.ctx.Err() != nil {
		l.proposal = nil
	}
	return l.proposal
}

func (l *lifecycle) handle(a UIAction) (snoofer.Outcome, bool) {
	p := l.pending()
	reject := func(detail string) (snoofer.Outcome, bool) {
		return snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}, false
	}
	if a.session != nil && a.session.ctx.Err() != nil {
		return reject("Controls closed")
	}
	switch a.Kind {
	case "retry":
		if p != nil {
			return reject("Finish the current confirmation first")
		}
		l.host.Retry()
		return snoofer.Outcome{Status: snoofer.Submitted, Detail: "Plugin retry requested"}, false
	case "selection":
		if p != nil {
			return reject("Finish the current confirmation first")
		}
		if a.session == nil || a.ID == "" {
			return reject("Controls session unavailable")
		}
		current, revision, err := snoofer.Load(l.path)
		if err != nil {
			return lifecycleFailure(err), false
		}
		if a.ConfigRevision == "" || a.ConfigRevision != revision {
			return reject("Configuration changed; try again")
		}
		next, err := l.host.SelectionFrom(current, a.Plugin, a.Enable)
		if err != nil {
			return reject(err.Error())
		}
		l.next++
		l.proposal = &enablementProposal{id: strconv.FormatUint(l.next, 10), source: a.ID, owner: a.session, expected: revision, config: next, text: confirmation(current, next)}
		return snoofer.Outcome{Status: snoofer.Processed, Detail: "Review changes"}, false
	case "confirm", "cancel":
		if p == nil || a.ProposalID == "" || a.ProposalID != p.id || a.session != p.owner {
			return reject("Confirmation changed; try again")
		}
		l.proposal = nil
		if a.Kind == "cancel" {
			return snoofer.Outcome{Status: snoofer.Processed, Detail: "Cancelled"}, false
		}
		if _, err := snoofer.Save(l.path, p.config, p.expected); err != nil {
			return lifecycleFailure(err), false
		}
		return snoofer.Outcome{Status: snoofer.Processed, Detail: "Saved; restart requested"}, true
	default:
		return reject("Unknown lifecycle action")
	}
}
