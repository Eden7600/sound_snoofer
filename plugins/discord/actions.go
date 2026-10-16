package discord

import (
	"time"

	"sound-snoofer/snoofer"
)

type meetingActionWrite struct {
	command    snoofer.MeetingActionCommand
	generation int
	deadline   time.Time
}

func (w *worker) meetingActionValue(action snoofer.MeetingAction) (snoofer.MeetingValue, string) {
	switch action {
	case snoofer.MeetingDeafen:
		return w.meetingReported.Deafen, "discord.deafen"
	case snoofer.MeetingShare:
		return w.meetingReported.Share, "discord.screenshare"
	case snoofer.MeetingLeave:
		return w.meetingReported.Leave, "discord.leave"
	}
	return snoofer.MeetingValue{}, ""
}

func (w *worker) finishMeetingAction(p *meetingActionWrite, status snoofer.OutcomeStatus, detail string) {
	if w.meetingActions[p.command.Action] != p {
		return
	}
	delete(w.meetingActions, p.command.Action)
	p.command.Complete(snoofer.Outcome{Status: status, Detail: detail})
}

func (w *worker) settleMeetingActions(now time.Time) {
	for _, p := range w.meetingActions {
		_, id := w.meetingActionValue(p.command.Action)
		switch {
		case w.generation != p.generation || w.conn == nil:
			w.finishMeetingAction(p, snoofer.Unknown, "Discord connection changed before confirmation")
		case p.command.Action == snoofer.MeetingLeave && w.voice.channelKnown && w.voice.channelID == "":
			w.finishMeetingAction(p, snoofer.Processed, "Discord call departure observed")
		case p.command.Lifetime != w.meetingLifetime():
			w.finishMeetingAction(p, snoofer.Unknown, "Discord call changed before confirmation")
		case p.command.Action != snoofer.MeetingLeave && w.observed(id) == snoofer.OnOff(p.command.Value):
			w.finishMeetingAction(p, snoofer.Processed, "Discord state observed")
		case !now.Before(p.deadline):
			w.finishMeetingAction(p, snoofer.Unknown, "Discord state not confirmed")
		}
	}
}

func (w *worker) applyMeetingAction(c snoofer.MeetingActionCommand, now time.Time) {
	reject := func(detail string) { c.Complete(snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}) }
	w.reportMeetings(now)
	if c.Context.Err() != nil || !w.services.Live || w.conn == nil || w.state != stateReady || !w.meetingReported.Joined || c.Peer != "discord" || c.Lifetime != w.meetingLifetime() {
		reject("Discord call changed or unavailable")
		return
	}
	value, id := w.meetingActionValue(c.Action)
	if id == "" || !value.Available || value.Revision != c.Revision || (c.Action == snoofer.MeetingDeafen && !value.Known) || (c.Action == snoofer.MeetingShare && !value.Known && !c.Value) || (c.Action == snoofer.MeetingLeave && !c.Value) {
		reject("Discord control changed or unavailable")
		return
	}
	if _, pending := w.pending[id]; pending || w.meetingActions[c.Action] != nil {
		reject("Discord control pending")
		return
	}
	if c.Action != snoofer.MeetingLeave && value.Known && value.Value == c.Value {
		c.Complete(snoofer.Outcome{Status: snoofer.Processed, Detail: "Discord state observed"})
		return
	}
	p := &meetingActionWrite{command: c, generation: w.generation, deadline: now.Add(verifyDeadline)}
	w.meetingActions[c.Action] = p
	local := w.capture(snoofer.Request{ID: id, Operation: "press"})
	local.known, local.desired, local.focused = value.Known, c.Value, true
	local.complete = func(outcome snoofer.Outcome) {
		// RPC acknowledgement alone does not prove the app changed state.
		if outcome.Status != snoofer.Processed {
			w.finishMeetingAction(p, outcome.Status, outcome.Detail)
		}
	}
	w.handle(c.Context, local, now)
}
