package discord

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"sound-snoofer/snoofer"
)

type meetingWrite struct {
	command  snoofer.MeetingCommand
	deadline time.Time
	nonce    string
}

func (w *worker) meetingLifetime() string {
	return fmt.Sprintf("%d/%d/%s", w.generation, w.channelLifetime, w.voice.channelID)
}

// reportMeetings runs on every observation, before another event can hide an ABA
// change. Observations remain actual readback, not the coordinator's target.
func (w *worker) reportMeetings(now time.Time) {
	w.settleMeetingActions(now)
	if w.meetings == nil {
		return
	}
	joined := w.conn != nil && w.state == stateReady && w.voice.channelKnown && w.voice.channelID != ""
	peer := snoofer.MeetingPeer{ID: "discord", Lifetime: w.meetingLifetime(), Label: "Discord", Joined: joined, NewCall: w.meetingNewCall}
	peer.Mute = snoofer.MeetingValue{Known: w.voice.settingsKnown, Value: w.voice.mute, Available: joined && w.voice.settingsKnown && !w.voice.deaf && w.services.Live}
	peer.Camera = snoofer.MeetingValue{Known: w.voice.videoKnown, Value: w.voice.video, Available: joined && w.voice.videoKnown && w.services.Live}
	peer.Deafen = snoofer.MeetingValue{Known: w.voice.settingsKnown, Value: w.voice.deaf, Available: joined && w.voice.settingsKnown && w.services.Live}
	peer.Share = snoofer.MeetingValue{Known: w.voice.screenKnown, Value: w.voice.screen, Available: joined && w.services.Live}
	peer.Leave = snoofer.MeetingValue{Known: joined, Value: joined, Available: joined && w.services.Live}
	previous := w.meetingReported
	advance := func(value, old snoofer.MeetingValue, revision *uint64) snoofer.MeetingValue {
		if *revision == 0 || previous.Lifetime != peer.Lifetime || previous.Joined != peer.Joined || value.Known != old.Known || value.Value != old.Value || value.Available != old.Available {
			*revision = *revision + 1
		}
		value.Revision = *revision
		return value
	}
	peer.Mute = advance(peer.Mute, previous.Mute, &w.muteRevision)
	peer.Camera = advance(peer.Camera, previous.Camera, &w.cameraRevision)
	peer.Deafen = advance(peer.Deafen, previous.Deafen, &w.actionRevisions[0])
	peer.Share = advance(peer.Share, previous.Share, &w.actionRevisions[1])
	peer.Leave = advance(peer.Leave, previous.Leave, &w.actionRevisions[2])
	if peer == previous {
		return
	}
	w.meetingReported = peer
	var peers []snoofer.MeetingPeer
	if joined {
		peers = []snoofer.MeetingPeer{peer}
	}
	if err := w.meetings.Report(peers); err != nil {
		w.link.Fail(err.Error(), now)
	}
}

func (w *worker) meetingValue(property snoofer.MeetingProperty) snoofer.MeetingValue {
	if property == snoofer.MeetingMute {
		return w.meetingReported.Mute
	}
	if property == snoofer.MeetingCamera {
		return w.meetingReported.Camera
	}
	return snoofer.MeetingValue{}
}

func meetingControlID(property snoofer.MeetingProperty) string {
	if property == snoofer.MeetingMute {
		return "discord.call-mute"
	}
	return "discord.call-camera"
}

func (w *worker) finishMeeting(property snoofer.MeetingProperty, status snoofer.OutcomeStatus, detail string) {
	p := w.meetingWrites[property]
	if p == nil {
		return
	}
	delete(w.meetingWrites, property)
	delete(w.calls, p.nonce)
	id := meetingControlID(property)
	if status == snoofer.Processed {
		delete(w.notes, id)
	} else {
		w.notes[id] = "Not confirmed"
	}
	p.command.Complete(snoofer.Outcome{Status: status, Detail: detail})
}

func (w *worker) finishMeetings(detail string) {
	for property := range w.meetingWrites {
		w.finishMeeting(property, snoofer.Unknown, detail)
	}
}

func (w *worker) syncMeetings(now time.Time) {
	if w.meetings == nil {
		return
	}
	w.reportMeetings(now)
	for property, p := range w.meetingWrites {
		value := w.meetingValue(property)
		switch {
		case !w.meetingReported.Joined || p.command.Lifetime != w.meetingReported.Lifetime:
			w.finishMeeting(property, snoofer.Unknown, "Discord call changed before confirmation")
		case value.Known && value.Available && value.Value == p.command.Value && value.Revision != p.command.Revision:
			w.finishMeeting(property, snoofer.Processed, "Discord state observed")
		case !now.Before(p.deadline):
			w.finishMeeting(property, snoofer.Unknown, "Discord state not confirmed")
		}
	}
	for _, property := range []snoofer.MeetingProperty{snoofer.MeetingMute, snoofer.MeetingCamera} {
		value, target := w.meetingValue(property), w.sharedTarget(property)
		if w.meetingWrites[property] == nil && value.Known && value.Available && target.Known && value.Value == target.Value {
			delete(w.notes, meetingControlID(property))
		}
	}
	for {
		command, ok := w.meetings.NextAction()
		if !ok {
			break
		}
		w.applyMeetingAction(command, now)
	}
	for {
		command, ok := w.meetings.Next()
		if !ok {
			return
		}
		w.applyMeeting(command, now)
	}
}

func (w *worker) applyMeeting(c snoofer.MeetingCommand, now time.Time) {
	reject := func(detail string) { c.Complete(snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}) }
	if c.Context.Err() != nil {
		reject("Meeting target changed before sending")
		return
	}
	peer := w.meetingReported
	if !w.services.Live || w.conn == nil || w.state != stateReady || !peer.Joined || c.Peer != "discord" || c.Lifetime != w.meetingLifetime() {
		reject("Discord call changed or unavailable")
		return
	}
	value := w.meetingValue(c.Property)
	if !value.Known || !value.Available || value.Revision != c.Revision {
		reject("Discord control changed or unavailable")
		return
	}
	if w.meetingWrites[c.Property] != nil {
		reject("Discord control pending")
		return
	}
	if value.Value == c.Value {
		c.Complete(snoofer.Outcome{Status: snoofer.Processed, Detail: "Discord state observed"})
		return
	}
	cmd := "TOGGLE_VIDEO"
	args := map[string]any{}
	if c.Property == snoofer.MeetingMute {
		cmd = "SET_VOICE_SETTINGS"
		args["mute"] = c.Value
	}
	// Recheck cancellation at the final native write boundary. Once submitted,
	// keep the reservation until readback or timeout even if newer intent wins.
	if c.Context.Err() != nil {
		reject("Meeting target changed before sending")
		return
	}
	p := &meetingWrite{command: c, deadline: now.Add(verifyDeadline)}
	w.meetingWrites[c.Property] = p
	delete(w.notes, meetingControlID(c.Property))
	if w.sendEvent(cmd, "", args, "meeting", string(c.Property), requestTimeout, now, nil) {
		p.nonce = strconv.Itoa(w.nonce)
	}
}

func (w *worker) meetingSnapshot() snoofer.MeetingSnapshot {
	if w.services.Meetings == nil {
		return snoofer.MeetingSnapshot{}
	}
	return w.services.Meetings.Snapshot()
}

func (w *worker) sharedTarget(property snoofer.MeetingProperty) snoofer.MeetingTarget {
	s := w.meetingSnapshot()
	if property == snoofer.MeetingMute {
		return s.Mute
	}
	return s.Camera
}

func (w *worker) requestMeeting(ctx context.Context, c command) {
	property := snoofer.MeetingMute
	if c.request.ID == "discord.video" {
		property = snoofer.MeetingCamera
	}
	if w.services.Meetings == nil || !w.services.Live || !c.known {
		c.complete(snoofer.Outcome{Status: snoofer.Rejected, Detail: "Meeting target unavailable"})
		return
	}
	if err := w.services.Meetings.Request(ctx, property, c.targetRevision, c.desired, c.complete); err != nil {
		c.complete(snoofer.Outcome{Status: snoofer.Rejected, Detail: err.Error()})
	}
}
