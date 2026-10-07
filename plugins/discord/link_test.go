package discord

import (
	"testing"
	"time"
)

// sets counts the SET_VOICE_SETTINGS commands Discord received.
func (d *fakeDiscord) sets() int {
	n := 0
	for _, cmd := range d.received() {
		if cmd == "SET_VOICE_SETTINGS" {
			n++
		}
	}
	return n
}

// external changes Discord's state as its own button or keybind would.
func (d *fakeDiscord) external(mute, deaf bool) {
	d.mu.Lock()
	d.mute, d.deaf = mute, deaf
	settings := raw(map[string]any{"mute": d.mute, "deaf": d.deaf})
	d.mu.Unlock()
	d.reply(message{Cmd: "DISPATCH", Evt: "VOICE_SETTINGS_UPDATE", Data: settings})
}

func linked(t *testing.T, muted bool) *harness {
	t.Helper()
	saved := configured
	saved.RefreshToken = "R1"
	h := newHarness(t, saved, true)
	h.mic.muted = muted
	ready(t, h)
	h.cycle()
	return h
}

func TestConnectImposesPreference(t *testing.T) {
	h := linked(t, true)
	if !h.peer.mute || len(h.mic.requests) != 0 {
		t.Fatal("connect adopted Discord or did not impose", h.peer.mute, h.mic.requests)
	}
	if c := h.control(t, "discord.mute"); c.Value != "On" || c.Status != "" || c.Mirrors != "audio.mic-mute" {
		t.Fatal(c.Value, c.Status, c.Mirrors)
	}
}

func TestSnooferChangeImposedOnce(t *testing.T) {
	h := linked(t, false)
	before := h.peer.sets()
	h.mic.muted = true // Mic mute pressed in Snoofer.
	h.cycle()
	if !h.peer.mute || h.peer.sets() != before+1 || len(h.mic.requests) != 0 {
		t.Fatal("impose", h.peer.mute, h.peer.sets()-before, h.mic.requests)
	}
	// Discord's acknowledgement of our own write is not adopted.
	h.cycle()
	if len(h.mic.requests) != 0 || h.peer.sets() != before+1 {
		t.Fatal("acknowledgement looped", h.mic.requests, h.peer.sets()-before)
	}
}

func TestDiscordChangeAdopted(t *testing.T) {
	h := linked(t, false)
	before := h.peer.sets()
	h.peer.external(true, false)
	h.cycle()
	if len(h.mic.requests) != 1 || !h.mic.requests[0] || !h.mic.muted {
		t.Fatal("not adopted", h.mic.requests)
	}
	if h.peer.sets() != before {
		t.Fatal("wrote back to Discord after adopting", h.peer.sets()-before)
	}
	if c := h.control(t, "discord.mute"); c.Value != "On" || c.Status != "" {
		t.Fatal(c.Value, c.Status)
	}
}

func TestDeafenNeitherAdoptsNorImposes(t *testing.T) {
	h := linked(t, false)
	before := h.peer.sets()
	h.peer.external(true, true) // Deafened: Discord reports itself muted.
	h.cycle()
	if len(h.mic.requests) != 0 || h.peer.sets() != before {
		t.Fatal("deafen changed the link", h.mic.requests, h.peer.sets()-before)
	}
	h.mic.muted = true
	h.cycle()
	h.peer.external(false, false) // Undeafened with Discord unmuted.
	h.cycle()
	if !h.peer.mute || h.peer.sets() != before+1 || len(h.mic.requests) != 0 {
		t.Fatal("undeafen did not re-impose", h.peer.mute, h.peer.sets()-before, h.mic.requests)
	}
}

func TestFailedWriteNotRetried(t *testing.T) {
	h := linked(t, false)
	before := h.peer.sets()
	h.peer.frozen = true
	h.mic.muted = true
	h.cycle()
	h.cycle()
	if h.peer.sets() != before+1 {
		t.Fatal("more than one write in flight", h.peer.sets()-before)
	}
	h.now = h.now.Add(4 * time.Second)
	h.cycle()
	h.now = h.now.Add(4 * time.Second)
	h.cycle()
	if h.peer.sets() != before+1 || h.control(t, "discord.mute").Status != "Failed" {
		t.Fatal("retried", h.peer.sets()-before, h.control(t, "discord.mute").Status)
	}
	// A new preference is imposed again.
	h.peer.frozen = false
	h.mic.muted = false
	h.cycle()
	h.mic.muted = true
	h.cycle()
	if !h.peer.mute || h.control(t, "discord.mute").Status != "" {
		t.Fatal("new preference not imposed", h.peer.mute, h.control(t, "discord.mute").Status)
	}
}

func TestMutePressTogglesPreference(t *testing.T) {
	h := linked(t, false)
	h.press("discord.mute")
	if len(h.mic.requests) != 1 || !h.mic.requests[0] || !h.peer.mute {
		t.Fatal(h.mic.requests, h.peer.mute)
	}
}
