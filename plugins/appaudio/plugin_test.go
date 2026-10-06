package appaudio

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
)

// fakeSessions is a session backend whose state the test controls. Paths in
// stubborn ignore volume writes, like programs that reset their own volume.
type fakeSessions struct {
	mu       sync.Mutex
	sessions []windowsaudio.Session
	peaks    map[string]float64
	stubborn map[string]bool
	writes   int
}

func (f *fakeSessions) List() ([]windowsaudio.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.sessions), nil
}

func (f *fakeSessions) Peak(key string) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.peaks[key], nil
}

func (f *fakeSessions) update(key string, change func(*windowsaudio.Session)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	for n := range f.sessions {
		if f.sessions[n].Key == key {
			if !f.stubborn[f.sessions[n].Path] {
				change(&f.sessions[n])
			}
			return nil
		}
	}
	return errors.New("audio session ended")
}

func (f *fakeSessions) SetVolume(key string, v float64) error {
	return f.update(key, func(s *windowsaudio.Session) { s.Volume = v })
}

func (f *fakeSessions) SetMute(key string, m bool) error {
	return f.update(key, func(s *windowsaudio.Session) { s.Muted = m })
}

func (f *fakeSessions) Close() {}

func (f *fakeSessions) setPeak(key string, peak float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.peaks[key] = peak
}

func studio() *fakeSessions {
	session := func(key, path, name string, volume float64, muted bool) windowsaudio.Session {
		return windowsaudio.Session{Key: key, Device: "Voicemeeter Input", Path: path, Name: name, Volume: volume, Muted: muted, Active: true}
	}
	return &fakeSessions{
		sessions: []windowsaudio.Session{
			session("c1", `C:\Chrome\chrome.exe`, "Google Chrome", 0.8, false),
			session("c2", `C:\Chrome\chrome.exe`, "Google Chrome", 0.6, true),
			session("c3", `C:\Chrome\chrome.exe`, "Google Chrome", 0.8, false),
			session("d1", `C:\Discord\Discord.exe`, "Discord", 0.4, false),
			session("s1", `C:\Snoofer\snoofer.exe`, "snoofer", 1, false),
			session("v1", `C:\VB\voicemeeter8x64.exe`, "VB-AUDIO Mixing Console", 1, false),
			session("g1", `C:\Games\game.exe`, "Game", 1, false),
			session("l1", `C:\Games\launcher.exe`, "Launcher", 1, false),
			session("x1", `C:\Tools\stubborn.exe`, "Stubborn", 0.5, false),
		},
		peaks:    map[string]float64{"c1": 0.5, "d1": 0.2, "v1": 0.3, "x1": 0.1},
		stubborn: map[string]bool{`C:\Tools\stubborn.exe`: true},
	}
}

type harness struct {
	t        *testing.T
	controls *snoofer.Controls
	backend  *fakeSessions
	mu       sync.Mutex
	saved    Settings
}

func startHarness(t *testing.T, backend *fakeSessions, settings Settings, live bool) *harness {
	t.Helper()
	h := &harness{t: t, controls: snoofer.NewControls(), backend: backend, saved: settings}
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	services := snoofer.Services{Controls: h.controls, Live: live, SaveSettings: func(id string, expected, next json.RawMessage) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		current, _ := json.Marshal(h.saved)
		if id != "appaudio" || string(current) != string(expected) {
			t.Errorf("save for %s expected %s, have %s", id, expected, current)
		}
		h.saved = Settings{}
		return json.Unmarshal(next, &h.saved)
	}}
	open := func() (windowsaudio.SessionBackend, error) { return backend, nil }
	instance, err := start(context.Background(), services, raw, open, timing{poll: 5 * time.Millisecond, list: 20 * time.Millisecond, observe: 150 * time.Millisecond, retry: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := instance.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	return h
}

func (h *harness) wait(what string, ready func([]snoofer.Control) bool) []snoofer.Control {
	h.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		list := h.controls.Snapshot()
		if ready(list) {
			return list
		}
		if time.Now().After(deadline) {
			var names []string
			for _, c := range list {
				names = append(names, c.ID+"="+c.Label+":"+c.Value+"/"+c.Status)
			}
			h.t.Fatalf("timed out waiting for %s: %v", what, names)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// apps lists visible app labels in their published order.
func apps(list []snoofer.Control) []string {
	var members []snoofer.Control
	for _, c := range list {
		if c.Collection == "appaudio.apps" {
			members = append(members, c)
		}
	}
	slices.SortFunc(members, func(a, b snoofer.Control) int { return a.Order - b.Order })
	var out []string
	for _, c := range members {
		out = append(out, c.Label)
	}
	return out
}

func find(list []snoofer.Control, label string) snoofer.Control {
	for _, c := range list {
		if c.Collection == "appaudio.apps" && c.Label == label {
			return c
		}
	}
	return snoofer.Control{}
}

func (h *harness) dispatch(label, op, value string, delta int) {
	h.t.Helper()
	for attempt := 0; attempt < 50; attempt++ {
		c := find(h.controls.Snapshot(), label)
		if label == "edit" {
			for _, e := range h.controls.Snapshot() {
				if e.ID == "appaudio.edit" {
					c = e
				}
			}
		}
		err := h.controls.Dispatch(context.Background(), snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: op, Value: value, Delta: delta})
		if err == nil {
			return
		}
		time.Sleep(2 * time.Millisecond) // A newer revision raced the dispatch.
	}
	h.t.Fatalf("dispatch %s %s failed", label, op)
}

func (h *harness) edit(op, app, value string) {
	h.t.Helper()
	data, _ := json.Marshal(edit{Op: op, App: app, Value: value})
	h.dispatch("edit", "set", string(data), 0)
}

func TestVisibilityGroupingAndDefaults(t *testing.T) {
	h := startHarness(t, studio(), Settings{Picked: []string{"Game", "Closed App"}}, true)
	list := h.wait("apps", func(l []snoofer.Control) bool { return len(apps(l)) == 5 })
	// Picked first in order (Game silent, Closed App absent), then heard apps by
	// name. Snoofer and Voicemeeter are hidden by default; the launcher is silent.
	if got := apps(list); !slices.Equal(got, []string{"Game", "Closed App", "Discord", "Google Chrome", "Stubborn"}) {
		t.Fatal("order", got)
	}
	chrome := find(list, "Google Chrome")
	if chrome.Value != "Mixed" || !chrome.Meter.Known || chrome.Meter.DB > -5 || chrome.Meter.DB < -7 {
		t.Fatalf("chrome %+v", chrome)
	}
	if find(list, "Closed App").Value != "Closed" || find(list, "Discord").Value != "40%" {
		t.Fatal("values", find(list, "Closed App").Value, find(list, "Discord").Value)
	}
	var view statusView
	for _, c := range list {
		if c.ID == "appaudio.status" {
			if err := json.Unmarshal(c.ViewData, &view); err != nil {
				t.Fatal(err)
			}
		}
	}
	hidden := map[string]string{}
	for _, a := range view.Apps {
		if a.Hidden {
			hidden[a.Name] = a.Rule
		}
		if a.Name == "Google Chrome" && (a.Sessions != 3 || len(a.Executables) != 1) {
			t.Fatalf("chrome view %+v", a)
		}
	}
	if !strings.HasPrefix(hidden["snoofer"], "Hidden by default") || hidden["VB-AUDIO Mixing Console"] == "" {
		t.Fatal("hidden", hidden)
	}
}

func TestWritesObserveAndIgnore(t *testing.T) {
	backend := studio()
	h := startHarness(t, backend, Settings{}, true)
	h.wait("chrome", func(l []snoofer.Control) bool { return find(l, "Google Chrome").Value == "Mixed" })

	// Mixed mute: a press mutes every session.
	h.dispatch("Google Chrome", "press", "", 0)
	h.wait("chrome muted", func(l []snoofer.Control) bool {
		c := find(l, "Google Chrome")
		return c.Value == "Muted" && c.Status == ""
	})
	// Two detents from 40%.
	h.dispatch("Discord", "adjust", "", 2)
	h.wait("discord 44%", func(l []snoofer.Control) bool {
		c := find(l, "Discord")
		return c.Value == "44%" && c.Status == ""
	})
	h.dispatch("Discord", "set", "75", 0)
	h.wait("discord 75%", func(l []snoofer.Control) bool { return find(l, "Discord").Value == "75%" })
	// A program that resets its own volume is reported, not fought.
	h.dispatch("Stubborn", "adjust", "", -5)
	h.wait("pending", func(l []snoofer.Control) bool { return find(l, "Stubborn").Status == "Pending" })
	h.wait("ignored", func(l []snoofer.Control) bool {
		c := find(l, "Stubborn")
		return c.Status == "Ignored by app" && c.Value == "50%"
	})
}

func TestPreviewNeverWrites(t *testing.T) {
	backend := studio()
	h := startHarness(t, backend, Settings{}, false)
	h.wait("preview", func(l []snoofer.Control) bool {
		for _, c := range l {
			if c.ID == "appaudio.status" {
				return c.Value == "Preview" && len(apps(l)) > 0
			}
		}
		return false
	})
	h.dispatch("Discord", "press", "", 0)
	time.Sleep(50 * time.Millisecond)
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.writes != 0 {
		t.Fatal("preview wrote", backend.writes)
	}
}

func TestEdits(t *testing.T) {
	backend := studio()
	h := startHarness(t, backend, Settings{Picked: []string{"Discord"}}, true)
	h.wait("apps", func(l []snoofer.Control) bool { return len(apps(l)) == 3 })

	// Hiding drops the app and its pick.
	h.edit("hide", "Discord", "")
	h.wait("discord hidden", func(l []snoofer.Control) bool { return find(l, "Discord").ID == "" })
	h.mu.Lock()
	if len(h.saved.Picked) != 0 || len(h.saved.Rules) != 1 || !h.saved.Rules[0].Hide || h.saved.Rules[0].Match != `(^|\\)discord\.exe$` {
		t.Fatalf("hide saved %+v", h.saved)
	}
	h.mu.Unlock()
	h.edit("unhide", "Discord", "")
	h.wait("discord back", func(l []snoofer.Control) bool { return find(l, "Discord").ID != "" })

	// Unhiding a default needs an explicit show rule ahead of the defaults.
	h.edit("unhide", "VB-AUDIO Mixing Console", "")
	h.wait("voicemeeter shown", func(l []snoofer.Control) bool { return find(l, "VB-AUDIO Mixing Console").ID != "" })

	// Combining the launcher into the game makes one app; picking keeps it
	// visible while silent, and its controls cover both programs.
	h.edit("pick", "Game", "")
	h.edit("combine", "Launcher", "Game")
	list := h.wait("combined", func(l []snoofer.Control) bool { return find(l, "Launcher").ID == "" && find(l, "Game").ID != "" })
	game := find(list, "Game")
	h.dispatch("Game", "press", "", 0)
	h.wait("game muted", func(l []snoofer.Control) bool { return find(l, "Game").Value == "Muted" })
	backend.mu.Lock()
	for _, s := range backend.sessions {
		if (s.Key == "g1" || s.Key == "l1") && !s.Muted {
			t.Error("combined session not muted", s.Key)
		}
	}
	backend.mu.Unlock()

	// Renaming moves the pick; the control ID follows the name.
	h.edit("rename", "Game", "Flight sim")
	list = h.wait("renamed", func(l []snoofer.Control) bool { return find(l, "Flight sim").ID != "" })
	if find(list, "Flight sim").ID == game.ID {
		t.Fatal("renamed app kept the old control ID")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !slices.Equal(h.saved.Picked, []string{"Flight sim"}) {
		t.Fatal("picks after rename", h.saved.Picked)
	}
}

func TestHeardWindow(t *testing.T) {
	w := &worker{}
	now := time.Unix(10000, 0)
	all := []*app{{Name: "A", Sessions: []windowsaudio.Session{{Key: "a"}}}, {Name: "B", Sessions: []windowsaudio.Session{{Key: "b"}}}}
	heard := map[string]time.Time{"a": now.Add(-4 * time.Minute), "b": now.Add(-6 * time.Minute)}
	got := visible(all, nil, heard, w.settings.window(), now)
	if len(got) != 1 || got[0].Name != "A" {
		t.Fatal("window", got)
	}
}

func TestExeMatch(t *testing.T) {
	match, err := exeMatch([]string{`C:\A\Game.exe`, `D:\B\launcher.exe`, `C:\A\Game.exe`, windowsaudio.SystemSounds})
	if err != nil || match != `(^|\\)game\.exe$|(^|\\)launcher\.exe$|^system$` {
		t.Fatal(match, err)
	}
	if _, err := exeMatch([]string{""}); err == nil {
		t.Fatal("unknown program matched")
	}
	rules, err := compileRules([]Rule{{Match: match, Name: "Game"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{`E:\other\GAME.EXE`, `x\launcher.exe`, "system"} {
		if r, ok := matchRule(rules, p); !ok || r.Name != "Game" {
			t.Error("no match", p)
		}
	}
	if _, ok := matchRule(rules, `C:\notgame.exe`); ok {
		t.Error("partial name matched")
	}
}
