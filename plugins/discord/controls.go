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
			return onOff(v.deaf)
		}
	case "discord.video":
		if v.videoKnown {
			return onOff(v.video)
		}
	case "discord.screenshare":
		if v.screenKnown {
			return onOff(v.screen)
		}
	case "discord.leave":
		if v.channelKnown && v.channelID == "" {
			return "left"
		}
	}
	return ""
}

// handle runs one request. Dispatch enforces availability; the checks here
// cover state that changed since the request was made.
func (w *worker) handle(_ context.Context, r snoofer.Request, now time.Time) {
	delete(w.notes, r.ID)
	if r.ID == "discord.connect" {
		if w.state == stateUnauthorized && w.conn != nil {
			w.state = stateAuthorizing
			w.send("AUTHORIZE", map[string]any{"client_id": w.settings.ClientID, "scopes": scopes}, "authorize", r.ID, authorizeTimeout, now)
		}
		return
	}
	if w.state != stateReady {
		return
	}
	if !w.services.Live {
		w.notes[r.ID] = "Preview"
		return
	}
	var want string
	switch r.ID {
	case "discord.mute":
		// The linked preference; reconcileMute applies it to Discord.
		if err := w.mic.SetMicMute(!w.mic.MicMute().Muted, "discord"); err != nil {
			w.notes[r.ID] = "Failed"
			w.link.Fail(err.Error(), now)
		}
		return
	case "discord.deafen":
		want = onOff(!w.voice.deaf)
		w.send("SET_VOICE_SETTINGS", map[string]any{"deaf": !w.voice.deaf}, "write", r.ID, requestTimeout, now)
	case "discord.video":
		if w.voice.videoKnown {
			want = onOff(!w.voice.video)
		}
		w.send("TOGGLE_VIDEO", map[string]any{}, "write", r.ID, requestTimeout, now)
	case "discord.screenshare":
		if w.voice.screenKnown {
			want = onOff(!w.voice.screen)
		}
		w.send("TOGGLE_SCREENSHARE", map[string]any{}, "write", r.ID, requestTimeout, now)
	case "discord.leave":
		want = "left"
		w.send("SELECT_VOICE_CHANNEL", map[string]any{"channel_id": nil}, "write", r.ID, requestTimeout, now)
	default:
		return
	}
	if want != "" {
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
		channel = "Voice channel"
	}
	toggle := func(id, label, short, icon string, blocked string, known bool) snoofer.Control {
		value := w.observed(id)
		if value == "On" {
			icon += "-on"
		}
		return snoofer.Control{ID: id, Label: label, ShortLabel: short, Group: "Discord", Kind: "toggle", Icon: icon, Value: value,
			Status: w.controlStatus(id, blocked), Operations: []string{"press"}, Available: blocked == "" && known}
	}
	connect := ""
	if w.state == stateAuthorizing {
		connect = "Pending"
	}
	return []snoofer.Control{
		{ID: "discord.channel", Label: "Discord channel", ShortLabel: "Channel", Group: "Discord", Kind: "status", Icon: "call",
			Value: channel, Status: reason, Available: true},
		w.muteControl(reason),
		toggle("discord.deafen", "Discord deafen", "Deafen", "discord-deafen", reason, w.voice.settingsKnown),
		toggle("discord.video", "Discord camera", "Camera", "discord-video", inCall, true),
		toggle("discord.screenshare", "Discord screen share", "Share", "screen-share", inCall, true),
		{ID: "discord.leave", Label: "Leave Discord call", ShortLabel: "Leave", Group: "Discord", Kind: "command", Icon: "call-leave",
			Status: w.controlStatus("discord.leave", inCall), Operations: []string{"press"}, Available: inCall == ""},
		{ID: "discord.connect", Label: "Connect Discord", ShortLabel: "Connect", Group: "Discord", Kind: "command", Icon: "discord-connect",
			Status: w.controlStatus("discord.connect", connect), Operations: []string{"press"}, Available: w.state == stateUnauthorized},
		w.report(now),
	}
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
		details = append(details, snoofer.ConnectionDetail{Label: "Voice channel", Value: onOff(w.voice.channelID != "")})
	}
	endpoint := w.pipeName
	if endpoint == "" {
		endpoint = `\\.\pipe\discord-ipc-0..9`
	}
	return snoofer.Control{ID: "discord.app-client", Label: "Discord", Group: "Discord", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: w.link.Report(endpoint, details...)}
}

func (w *worker) publish(now time.Time) {
	requests := w.requests
	_ = w.services.Controls.Publish("discord", w.controls(now), func(ctx context.Context, r snoofer.Request) error {
		select {
		case requests <- r:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("Discord queue full")
		}
	})
}

func onOff(on bool) string {
	if on {
		return "On"
	}
	return "Off"
}

// muteControl toggles the Mic mute preference that Discord mute follows. It
// mirrors audio.mic-mute, so automatic deck layouts show only one of them.
func (w *worker) muteControl(reason string) snoofer.Control {
	pref := w.mic.MicMute()
	c := snoofer.Control{ID: "discord.mute", Label: "Discord mute", ShortLabel: "Mute", Group: "Discord", Kind: "toggle", Icon: "mic-mute",
		Mirrors: "audio.mic-mute", Operations: []string{"press"}, Status: reason, Available: reason == "" && pref.Known}
	if pref.Known {
		c.Value = onOff(pref.Muted)
		if pref.Muted {
			c.Icon = "mic-mute-muted"
		}
	}
	if reason == "" {
		c.Status = w.muteStatus(pref)
	}
	return c
}
