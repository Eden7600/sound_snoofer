package discord

import (
	"time"

	"sound-snoofer/plugins/audio"
)

// micMute is the audio plugin's Mic mute preference, which Discord mute
// follows. Changes made in Discord become the preference.
type micMute interface {
	MicMute() audio.MicMute
	SetMicMute(muted bool, origin string) error
}

// muteLink is the state of reconciling Discord mute with the preference.
type muteLink struct {
	synced      bool // Discord agreed with the preference since connecting or undeafening.
	lastDiscord bool // Discord mute at the previous reconcile.
	writing     *expected
	adopting    *expected
	failed      *failure // Stops re-imposing a value Discord did not take.
}

type expected struct {
	want     bool
	deadline time.Time
}

type failure struct {
	preference, discord bool
}

// reconcileMute runs after every event and tick. It never has more than one
// Discord write or one preference change in flight.
func (w *worker) reconcileMute(now time.Time) {
	l := &w.mute
	pref := w.mic.MicMute()
	if w.state != stateReady || !w.voice.settingsKnown || !pref.Known || w.voice.deaf {
		// Discord reports itself muted while deafened: neither adopt nor impose,
		// and re-impose the preference once the link resumes.
		*l = muteLink{failed: l.failed}
		return
	}
	discord := w.voice.mute
	defer func() { l.lastDiscord = discord }()
	if l.adopting != nil {
		if pref.Muted != l.adopting.want && !now.After(l.adopting.deadline) {
			return
		}
		l.adopting = nil
	}
	if l.writing != nil {
		switch {
		case discord == l.writing.want:
			l.writing = nil
		case now.After(l.writing.deadline):
			l.failed = &failure{preference: l.writing.want, discord: discord}
			l.writing = nil
			w.notes["discord.mute"] = "Failed"
			return
		default:
			return
		}
	}
	switch {
	case discord == pref.Muted:
		l.synced = true
		l.failed = nil
		delete(w.notes, "discord.mute")
	case l.synced && discord != l.lastDiscord:
		// Discord's own button or keybind: the user acted on another surface.
		if err := w.mic.SetMicMute(discord, "discord"); err != nil {
			w.link.Fail(err.Error(), now)
			return
		}
		l.adopting = &expected{want: discord, deadline: now.Add(verifyDeadline)}
	case l.failed != nil && l.failed.preference == pref.Muted && l.failed.discord == discord:
		// Not retried until the preference or Discord changes.
	case w.services.Live:
		// The preference changed, or Discord differs after connecting.
		l.failed = nil
		delete(w.notes, "discord.mute")
		l.writing = &expected{want: pref.Muted, deadline: now.Add(verifyDeadline)}
		w.send("SET_VOICE_SETTINGS", map[string]any{"mute": pref.Muted}, "mute", "discord.mute", requestTimeout, now)
	}
}

// muteStatus is Pending until Voicemeeter and Discord both follow the preference.
func (w *worker) muteStatus(pref audio.MicMute) string {
	if note := w.notes["discord.mute"]; note != "" {
		return note
	}
	if !pref.Applied || w.mute.writing != nil || w.mute.adopting != nil {
		return "Pending"
	}
	if !w.voice.deaf && w.voice.mute != pref.Muted {
		return "Pending"
	}
	return ""
}
