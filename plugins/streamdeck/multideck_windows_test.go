//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

// decksSurface keeps the latest frame per deck.
type decksSurface struct {
	events chan device.Event
	done   chan struct{}
	mu     sync.Mutex
	frames map[string]device.Frame
}

func (s *decksSurface) Show(id string, f device.Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames[id] = f
}
func (s *decksSurface) Events() <-chan device.Event { return s.events }
func (s *decksSurface) Done() <-chan struct{}       { return s.done }
func (s *decksSurface) frame(id string) (device.Frame, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.frames[id]
	return f, ok
}

// deckRig runs the plugin with settings against decksSurface, recording
// dispatched control IDs and saved settings.
type deckRig struct {
	t        *testing.T
	ctx      context.Context
	surface  *decksSurface
	registry *snoofer.Controls
	mu       sync.Mutex
	pressed  []string
	saved    []Settings
}

func newDeckRig(t *testing.T, settings Settings) *deckRig {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	rig := &deckRig{t: t, ctx: ctx, registry: snoofer.NewControls(),
		surface: &decksSurface{events: make(chan device.Event, 32), done: make(chan struct{}), frames: map[string]device.Frame{}}}
	var controls []snoofer.Control
	for _, id := range []string{"test.a", "test.b", "test.c"} {
		controls = append(controls, snoofer.Control{ID: id, Label: id, Kind: "command", Operations: []string{"press"}, Available: true})
	}
	if err := rig.registry.Publish("test", controls, func(_ context.Context, r snoofer.Request) error {
		rig.mu.Lock()
		defer rig.mu.Unlock()
		rig.pressed = append(rig.pressed, r.ID)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	services := snoofer.Services{Controls: rig.registry, SaveSettings: func(_ string, _, data json.RawMessage) error {
		var s Settings
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		rig.mu.Lock()
		defer rig.mu.Unlock()
		rig.saved = append(rig.saved, s)
		return nil
	}}
	instance, err := startWithSurface(ctx, services, snoofer.MarshalSettings(settings), func(ctx context.Context) surface {
		go func() { <-ctx.Done(); close(rig.surface.done) }()
		return rig.surface
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		stop, release := context.WithTimeout(context.Background(), 2*time.Second)
		defer release()
		if err := instance.Stop(stop); err != nil {
			t.Error(err)
		}
	})
	return rig
}

func (r *deckRig) waitFor(what string, ok func() bool) {
	r.t.Helper()
	for deadline := time.Now().Add(3 * time.Second); !ok(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			r.t.Fatal(what)
		}
	}
}

func (r *deckRig) connect(id, model string) device.Frame {
	r.t.Helper()
	r.surface.events <- device.Event{Connected: true, Device: id, Serial: id, Model: model}
	return r.page(id, "")
}

// page waits until deck id shows a frame whose page is name ("" for any).
func (r *deckRig) page(id, name string) device.Frame {
	r.t.Helper()
	var f device.Frame
	r.waitFor(id+" did not show "+name, func() bool {
		var ok bool
		f, ok = r.surface.frame(id)
		return ok && (name == "" || pageOf(f) == name)
	})
	return f
}

// pageOf reads the current page name a frame shows, wherever the model shows it.
func pageOf(f device.Frame) string {
	for _, d := range f.Dials {
		if d.Value != "" && d.Label != "" {
			return d.Value
		}
	}
	if f.Screen.Value != "" {
		return f.Screen.Value
	}
	for _, k := range f.Keys {
		if k.Icon == "deck-page" {
			return k.Label
		}
	}
	return ""
}

func (r *deckRig) send(e device.Event) { r.surface.events <- e }

func (r *deckRig) lastPressed() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.pressed) == 0 {
		return ""
	}
	return r.pressed[len(r.pressed)-1]
}

func (r *deckRig) lastSaved() (Settings, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.saved) == 0 {
		return Settings{}, false
	}
	return r.saved[len(r.saved)-1], true
}

func (r *deckRig) command(id, op, value string) {
	r.t.Helper()
	var c snoofer.Control
	for _, each := range r.registry.Snapshot() {
		if each.ID == "streamdeck."+id {
			c = each
		}
	}
	if err := r.registry.Dispatch(r.ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: op, Value: value}); err != nil {
		r.t.Fatal(err)
	}
	r.waitFor(id+" not applied", func() bool {
		for _, each := range r.registry.Snapshot() {
			if each.ID == c.ID {
				return each.Revision != c.Revision
			}
		}
		return false
	})
}

// twoPages is a layout of model with pages A and B (Home A); key 0 of A
// binds control.
func twoPages(model, control string) Layout {
	l := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}}.forModel(model)
	l.Pages[0].Keys[0] = Binding{Control: control, Label: control}
	return l
}

func TestDecksKeepTheirOwnLayoutsAndPages(t *testing.T) {
	rig := newDeckRig(t, Settings{Layout: twoPages(plusXLModel, "test.a"), Models: map[string]Layout{"plus": twoPages("plus", "test.b")}})
	rig.connect("XL1", "plus-xl")
	rig.connect("P1", "plus")
	// Remembering each deck saves settings, which redraws every deck.
	rig.waitFor("decks not remembered", func() bool {
		s, ok := rig.lastSaved()
		return ok && len(s.Devices) == 2
	})
	time.Sleep(50 * time.Millisecond)
	xl, plus := rig.page("XL1", ""), rig.page("P1", "")

	// Each deck's input dispatches its own binding.
	rig.send(device.Event{Device: "P1", Key: 0, Encoder: -1, Press: true, Generation: plus.Generation})
	rig.waitFor("+ key", func() bool { return rig.lastPressed() == "test.b" })
	rig.send(device.Event{Device: "XL1", Key: 0, Encoder: -1, Press: true, Generation: xl.Generation})
	rig.waitFor("+ XL key", func() bool { return rig.lastPressed() == "test.a" })

	// Independent: the +'s fourth dial pages only the +.
	plus = rig.page("P1", "A")
	rig.send(device.Event{Device: "P1", Key: -1, Encoder: 3, Delta: 1, Generation: plus.Generation})
	rig.page("P1", "B")
	time.Sleep(100 * time.Millisecond)
	if f, _ := rig.surface.frame("XL1"); pageOf(f) != "A" {
		t.Fatal("independent paging moved the + XL to", pageOf(f))
	}
	// A key past the +'s grid is ignored.
	plus = rig.page("P1", "B")
	rig.send(device.Event{Device: "P1", Key: 9, Encoder: -1, Press: true, Generation: plus.Generation})

	// Mirrored: the + XL's page dial moves both; its press takes both Home.
	rig.command("paging", "set", pagingMirrored)
	if s, ok := rig.lastSaved(); !ok || s.Paging != pagingMirrored {
		t.Fatal("paging not saved", s)
	}
	xl = rig.page("XL1", "A")
	rig.send(device.Event{Device: "XL1", Key: -1, Encoder: 5, Delta: 1, Generation: xl.Generation})
	rig.page("XL1", "B")
	rig.page("P1", "B")
	plus = rig.page("P1", "B")
	rig.send(device.Event{Device: "P1", Key: -1, Encoder: 3, Press: true, Generation: plus.Generation})
	rig.page("P1", "A")
	rig.page("XL1", "A")

	// An unplugged deck leaves the other working.
	rig.send(device.Event{Device: "XL1", Error: "unplugged"})
	rig.mu.Lock()
	rig.pressed = nil
	rig.mu.Unlock()
	plus = rig.page("P1", "A")
	rig.send(device.Event{Device: "P1", Key: 0, Encoder: -1, Press: true, Generation: plus.Generation})
	rig.waitFor("+ after + XL unplugged", func() bool { return rig.lastPressed() == "test.b" })
}

func TestPagesKeyAndNeoTouchButtons(t *testing.T) {
	three := func(model string) Layout {
		return Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}}.forModel(model)
	}
	rig := newDeckRig(t, Settings{Layout: DefaultLayout(), Models: map[string]Layout{"mk2": three("mk2"), "neo": three("neo")}})
	mk2 := rig.connect("M1", "mk2")
	if k := mk2.Keys[14]; k.Label != "A" || k.Icon != "deck-page" {
		t.Fatalf("Pages key %+v", k)
	}
	tap := func(id string, key int) {
		f, _ := rig.surface.frame(id)
		rig.send(device.Event{Device: id, Key: key, Encoder: -1, Press: true, Generation: f.Generation})
		rig.send(device.Event{Device: id, Key: key, Encoder: -1, Release: true})
	}
	tap("M1", 14)
	rig.page("M1", "B")
	tap("M1", 14)
	rig.page("M1", "C")
	// Hold goes Home.
	f := rig.page("M1", "C")
	rig.send(device.Event{Device: "M1", Key: 14, Encoder: -1, Press: true, Generation: f.Generation})
	rig.page("M1", "A")
	rig.send(device.Event{Device: "M1", Key: 14, Encoder: -1, Release: true})

	neo := rig.connect("N1", "neo")
	if neo.Screen.Value != "A" || neo.Screen.Label != "C" || neo.Screen.Icon != "B" {
		t.Fatalf("Neo screen %+v", neo.Screen)
	}
	tap("N1", 9)
	rig.page("N1", "B")
	tap("N1", 8)
	rig.page("N1", "A")
	tap("N1", 8)
	rig.page("N1", "C")
}

func TestDecksAreRememberedAndManaged(t *testing.T) {
	rig := newDeckRig(t, Settings{Layout: DefaultLayout()})
	rig.connect("P1", "plus")
	rig.waitFor("deck not remembered", func() bool {
		s, ok := rig.lastSaved()
		return ok && s.Devices["P1"].Model == "plus"
	})
	rig.command("deck-name", "set", "Desk")
	if s, _ := rig.lastSaved(); s.Devices["P1"].Name != "Desk" || s.Devices["P1"].Model != "plus" {
		t.Fatal("name not saved", s.Devices)
	}
	rig.command("deck-own", "press", "")
	s, _ := rig.lastSaved()
	if own, ok := s.Serials["P1"]; !ok || own.Home != DefaultLayoutFor("plus").Home {
		t.Fatal("own layout not created", s.Serials)
	}
	var profile snoofer.Control
	for _, c := range rig.registry.Snapshot() {
		if c.ID == "streamdeck.profile" {
			profile = c
		}
	}
	if profile.Value != serialTarget+"P1" || profile.OptionLabels[serialTarget+"P1"] != "Desk (Stream Deck +)" {
		t.Fatalf("editor not on the deck's layout: %q %v", profile.Value, profile.OptionLabels)
	}
	rig.command("deck-forget", "press", "")
	if s, _ := rig.lastSaved(); len(s.Devices) != 0 || len(s.Serials) != 0 {
		t.Fatal("deck not forgotten", s)
	}
}
