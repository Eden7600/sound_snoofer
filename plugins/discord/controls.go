package discord

import (
	"context"
	"fmt"
	"time"

	"sound-snoofer/snoofer"
)

// observed is a control's state as Discord last reported it, or "".
func (w *worker) observed(id string) string {
	v := w.voice
	switch id {
	case "discord.deafen":
		if v.settingsKnown {
			return snoofer.OnOff(v.deaf)
		}
	case "discord.screenshare":
		if v.screenKnown {
			return snoofer.OnOff(v.screen)
		}
	case "discord.leave":
		if v.channelKnown && v.channelID == "" {
			return "left"
		}
	}
	return ""
}

// command retains the published target and desired state through the worker queue.
type command struct {
	request         snoofer.Request
	complete        snoofer.Completion
	generation      int
	channelID       string
	channelLifetime uint64
	known, desired  bool
	targetRevision  uint64
	focused         bool
}

func (w *worker) capture(r snoofer.Request) command {
	return w.captureTargets(r, w.meetingSnapshot())
}

func (w *worker) captureTargets(r snoofer.Request, targets snoofer.MeetingSnapshot) command {
	c := command{request: r, complete: func(snoofer.Outcome) {}, generation: w.generation,
		channelID: w.voice.channelID, channelLifetime: w.channelLifetime}
	switch r.ID {
	case "discord.mute", "discord.video":
		target := targets.Mute
		if r.ID == "discord.video" {
			target = targets.Camera
		}
		c.known, c.desired, c.targetRevision = target.Known, !target.Value, target.Revision
	case "discord.deafen":
		c.known, c.desired = w.voice.settingsKnown, !w.voice.deaf
	case "discord.screenshare":
		c.known, c.desired = w.voice.screenKnown, !w.voice.screen
	}
	return c
}

// handle rechecks the captured context before any side effect. RPC completion
// and the later observed pendingWrite state remain separate.
func (w *worker) handle(ctx context.Context, c command, now time.Time) {
	r := c.request
	reject := func(detail string) { c.complete(snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}) }
	if ctx.Err() != nil {
		c.complete(snoofer.Outcome{Status: snoofer.Unknown, Detail: "Discord stopped before processing"})
		return
	}
	if r.ID == "discord.mute" || r.ID == "discord.video" {
		w.requestMeeting(ctx, c)
		return
	}
	if c.generation != w.generation || w.conn == nil {
		reject("Discord connection changed")
		return
	}
	if r.ID == "discord.connect" {
		if w.state != stateUnauthorized {
			reject("Discord authorization state changed")
			return
		}
		delete(w.notes, r.ID)
		w.state = stateAuthorizing
		c.complete(snoofer.Outcome{Status: snoofer.Submitted, Detail: "Discord authorization requested"})
		w.send("AUTHORIZE", map[string]any{"client_id": w.settings.ClientID, "scopes": scopes}, "authorize", r.ID, authorizeTimeout, now)
		return
	}
	if reason := w.unavailable(); reason != "" {
		reject(reason)
		return
	}
	switch r.ID {
	case "discord.deafen", "discord.screenshare", "discord.leave":
		if !w.voice.channelKnown || c.channelID == "" || c.channelID != w.voice.channelID || c.channelLifetime != w.channelLifetime {
			reject("Discord call changed")
			return
		}
	}
	if !c.focused {
		for action := range w.meetingActions {
			_, id := w.meetingActionValue(action)
			if id == r.ID {
				reject("Discord control pending")
				return
			}
		}
	}
	delete(w.notes, r.ID)
	var cmd, want string
	args := map[string]any{}
	switch r.ID {
	case "discord.deafen":
		if !c.known || !w.voice.settingsKnown {
			reject("Discord voice settings unavailable")
			return
		}
		cmd, want = "SET_VOICE_SETTINGS", snoofer.OnOff(c.desired)
		args["deaf"] = c.desired
	case "discord.screenshare":
		if _, pending := w.pending[r.ID]; pending {
			reject("Discord toggle still pending")
			return
		}
		for _, call := range w.calls {
			if call.purpose == "write" && call.id == r.ID {
				reject("Discord toggle still pending")
				return
			}
		}
		known, current := w.voice.screenKnown, w.voice.screen
		cmd = "TOGGLE_SCREENSHARE"
		if known != c.known || (known && current == c.desired) {
			reject("Discord toggle state changed")
			return
		}
		if c.known {
			want = snoofer.OnOff(c.desired)
		}
	case "discord.leave":
		cmd, want = "SELECT_VOICE_CHANNEL", "left"
		args["channel_id"] = nil
	default:
		reject("Unknown Discord control")
		return
	}
	if ctx.Err() != nil {
		reject("Discord action cancelled before sending")
		return
	}
	if w.sendEvent(cmd, "", args, "write", r.ID, requestTimeout, now, c.complete) && want != "" {
		w.pending[r.ID] = pendingWrite{want: want, deadline: now.Add(verifyDeadline)}
	}
}

// unavailable is why voice controls cannot be used, or "".
func (w *worker) unavailable() string {
	switch w.state {
	case stateSetup:
		return "Setup needed"
	case stateClosed:
		return "Discord closed"
	case stateUnauthorized:
		return "Not connected"
	case stateAuthorizing:
		return "Approve in Discord"
	case stateReady:
		if !w.services.Live {
			return "Preview"
		}
		return ""
	}
	return "Connecting"
}

func (w *worker) controlStatus(id, blocked string) string {
	if _, ok := w.pending[id]; ok {
		return "Pending"
	}
	if note := w.notes[id]; note != "" {
		return note
	}
	return blocked
}

func (w *worker) controls(now time.Time) []snoofer.Control {
	return w.controlsFor(now, w.meetingSnapshot())
}

func (w *worker) controlsFor(now time.Time, targets snoofer.MeetingSnapshot) []snoofer.Control {
	reason := w.unavailable()
	inCall := reason
	if inCall == "" && (!w.voice.channelKnown || w.voice.channelID == "") {
		inCall = "Not in a call"
	}
	channel := "N/A"
	switch {
	case w.state != stateReady || !w.voice.channelKnown:
	case w.voice.channelID == "":
		channel = "Not in a call"
	case w.voice.channelName != "":
		channel = w.voice.channelName
	default:
		channel = "Voice Channel"
	}
	toggle := func(id, label, short, icon string, blocked string, known bool) snoofer.Control {
		value := w.observed(id)
		status := w.controlStatus(id, blocked)
		if id == "discord.deafen" && (blocked != "" || !known) {
			value = "N/A"
			status = blocked
		}
		// Deafened is slashed headphones, drawn like a mute.
		if id == "discord.deafen" && value == "On" {
			icon += "-muted"
		}
		return snoofer.Control{ID: id, Label: label, ShortLabel: short, Group: "Discord", Kind: "toggle", Icon: icon, Value: value,
			Status: status, Operations: []string{"press"}, Available: blocked == "" && known}
	}
	connect := ""
	if w.state == stateAuthorizing {
		connect = "Pending"
	}
	controls := []snoofer.Control{
		{ID: "discord.channel", Label: "Discord Channel", ShortLabel: "Channel", Group: "Discord", Kind: "status", Icon: "call",
			Value: channel, Status: reason, Available: true},
		w.sharedControl(snoofer.MeetingMute, targets.Mute, "discord.mute", "Mute", "mic-mute"),
		w.actualMeetingControl(snoofer.MeetingMute, inCall),
		w.actualMeetingControl(snoofer.MeetingCamera, inCall),
		toggle("discord.deafen", "Discord Deafen", "Deafen", "discord-deafen", inCall, w.voice.settingsKnown),
		w.sharedControl(snoofer.MeetingCamera, targets.Camera, "discord.video", "Camera", "discord-video"),
		toggle("discord.screenshare", "Discord Screen Share", "Share", "screen-share", inCall, true),
		{ID: "discord.leave", Label: "Leave Discord Call", ShortLabel: "Leave", Group: "Discord", Kind: "command", Icon: "call-leave",
			Status: w.controlStatus("discord.leave", inCall), Operations: []string{"press"}, Available: inCall == ""},
		{ID: "discord.connect", Label: "Connect Discord", ShortLabel: "Connect", Group: "Discord", Kind: "command", Icon: "discord-connect",
			Status: w.controlStatus("discord.connect", connect), Operations: []string{"press"}, Available: w.state == stateUnauthorized},
		w.report(now),
	}
	for i := range controls {
		c := &controls[i]
		c.Epoch = uint64(w.generation)
		switch c.ID {
		case "discord.deafen", "discord.screenshare", "discord.leave":
			c.Epoch = w.channelLifetime
		case "discord.mute":
			c.Epoch = targets.Mute.Revision
		case "discord.video":
			c.Epoch = targets.Camera.Revision
		}
	}
	return controls
}

func (w *worker) report(now time.Time) snoofer.Control {
	state, value := snoofer.ConnectionConnecting, "Connecting"
	switch w.state {
	case stateSetup:
		state, value = snoofer.ConnectionUnconfigured, "Setup needed"
	case stateClosed:
		state, value = snoofer.ConnectionDisconnected, "Closed"
	case stateUnauthorized:
		state, value = snoofer.ConnectionAttention, "Not connected"
	case stateAuthorizing:
		state, value = snoofer.ConnectionAttention, "Approve in Discord"
	case stateRetrying:
		state, value = snoofer.ConnectionAttention, "Retrying"
	case stateReady:
		state, value = snoofer.ConnectionConnected, "Connected"
	}
	w.link.Observe(state, now)
	authorization := "Needed"
	if w.settings.RefreshToken != "" {
		authorization = "Saved"
	}
	details := []snoofer.ConnectionDetail{{Label: "Authorization", Value: authorization}}
	if w.state == stateReady {
		details = append(details, snoofer.ConnectionDetail{Label: "Voice Channel", Value: snoofer.OnOff(w.voice.channelID != "")})
	}
	endpoint := w.pipeName
	if endpoint == "" {
		endpoint = `\\.\pipe\discord-ipc-0..9`
	}
	return snoofer.Control{ID: "discord.app-client", Label: "Discord", Group: "Discord", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: w.link.Report(endpoint, details...)}
}

func (w *worker) publish(ctx context.Context, now time.Time) {
	requests := w.requests
	targets := w.meetingSnapshot()
	controls := w.controlsFor(now, targets)
	commands := make(map[string]command, len(controls))
	for _, control := range controls {
		commands[control.ID] = w.captureTargets(snoofer.Request{ID: control.ID}, targets)
	}
	lifetime := ctx
	_ = w.services.Controls.Publish("discord", controls, func(ctx context.Context, r snoofer.Request) error {
		c := commands[r.ID]
		c.request = r
		c.complete = r.ClaimCompletion(lifetime)
		select {
		case requests <- c:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("Discord queue full")
		}
	})
}

// Compatibility bindings request the shared target; actual app state is separate.
func (w *worker) sharedControl(property snoofer.MeetingProperty, target snoofer.MeetingTarget, id, label, icon string) snoofer.Control {
	value := "N/A"
	if target.Known {
		value = snoofer.OnOff(target.Value)
	}
	if property == snoofer.MeetingMute && value == "On" {
		icon += "-muted"
	}
	if property == snoofer.MeetingCamera && value == "Off" {
		icon += "-off"
	}
	status := target.Status
	if !w.services.Live {
		status = "Preview"
	}
	return snoofer.Control{ID: id, Label: label, ShortLabel: label, Group: "Discord", Kind: "toggle", Icon: icon, Value: value, Status: status, Mirrors: "meetings." + string(property), Epoch: target.Revision, Operations: []string{"press"}, Available: w.services.Live && target.Known}
}

func (w *worker) actualMeetingControl(property snoofer.MeetingProperty, blocked string) snoofer.Control {
	known, value := w.voice.settingsKnown, w.voice.mute
	label, icon := "Discord Mute", "mic-mute"
	if property == snoofer.MeetingCamera {
		known, value = w.voice.videoKnown, w.voice.video
		label, icon = "Discord Camera", "discord-video"
	}
	text := "N/A"
	joined := w.conn != nil && w.state == stateReady && w.voice.channelKnown && w.voice.channelID != ""
	if joined && known {
		text = snoofer.OnOff(value)
	}
	status := blocked
	id := meetingControlID(property)
	if status == "" && !known {
		status = "Unknown state"
	}
	if status == "" {
		status = w.notes[id]
		if w.meetingWrites[property] != nil {
			status = "Pending"
		}
	}
	if property == snoofer.MeetingMute && w.voice.deaf && blocked == "" {
		status = "Deafened"
	}
	if property == snoofer.MeetingMute && text == "On" {
		icon += "-muted"
	}
	if property == snoofer.MeetingCamera && text == "Off" {
		icon += "-off"
	}
	return snoofer.Control{ID: id, Label: label, ShortLabel: label, Group: "Discord", Kind: "status", Icon: icon, Value: text, Status: status, SurfaceOnly: true, Available: true}
}
