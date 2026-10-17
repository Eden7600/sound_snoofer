package vrclipboard

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/image/font"

	"sound-snoofer/internal/desktopinput"
	"sound-snoofer/internal/vroverlay"
)

func testTarget() desktopinput.Target {
	return desktopinput.Target{Window: 12, Process: 34, Thread: 56, Started: 78, Label: "notepad.exe"}
}
func pointer(kind vroverlay.EventKind, button int) vroverlay.Event {
	box := padButtons[button].bounds
	return vroverlay.Event{Kind: kind, X: float32(box.Min.X + box.Dx()/2), Y: float32(box.Min.Y + box.Dy()/2), Button: 1, Cursor: 2, Device: 3}
}

func TestPanelGeometryAndAllowlist(t *testing.T) {
	actions := map[desktopinput.Action]bool{}
	for n, button := range padButtons {
		if !button.bounds.In(image.Rect(0, 0, panelWidth, panelHeight)) || button.bounds.Dx() < 200 || button.bounds.Dy() < 100 {
			t.Fatal("button too small/outside panel", n)
		}
		if buttonAt(float32(button.bounds.Min.X+3), float32(button.bounds.Min.Y+3)) != n {
			t.Fatal("hit box", n)
		}
		for prior := range n {
			if button.bounds.Overlaps(padButtons[prior].bounds) {
				t.Fatal("overlapping buttons", n)
			}
		}
		if button.action != "" {
			if actions[button.action] {
				t.Fatal("duplicate action")
			}
			actions[button.action] = true
		}
	}
	if len(actions) != 7 {
		t.Fatal(actions)
	}
	for _, point := range [][2]float32{{-1, 150}, {1024, 150}, {32, 512}, {float32(math.NaN()), 150}, {32, float32(math.Inf(1))}, {10, 10}} {
		if buttonAt(point[0], point[1]) != -1 {
			t.Fatal("invalid point hit", point)
		}
	}
}

func TestPanelFreshReleaseActsOnceAndCapturesTarget(t *testing.T) {
	now := time.Unix(100, 0)
	p := newPanel()
	p.activate(testTarget(), now)
	if got := p.handle(pointer(vroverlay.Down, 2), now, desktopinput.Target{}); got != nil {
		t.Fatal("down activated")
	}
	for range 3 {
		p.handle(pointer(vroverlay.Move, 2), now, desktopinput.Target{})
	}
	got := p.handle(pointer(vroverlay.Up, 2), now, desktopinput.Target{})
	if got == nil || got.action != desktopinput.Paste || got.target != testTarget() {
		t.Fatal(got)
	}
	if p.handle(pointer(vroverlay.Up, 2), now, desktopinput.Target{}) != nil {
		t.Fatal("release repeated")
	}
}

func TestPanelInterruptedOrStaleGesturesNeverAct(t *testing.T) {
	now := time.Unix(100, 0)
	cases := map[string]func(*panel, *vroverlay.Event){
		"other button":   func(_ *panel, up *vroverlay.Event) { *up = pointer(vroverlay.Up, 1) },
		"other hand":     func(_ *panel, up *vroverlay.Event) { up.Cursor++ },
		"other device":   func(_ *panel, up *vroverlay.Event) { up.Device++ },
		"target changed": func(p *panel, _ *vroverlay.Event) { p.target.Window++ },
		"hidden": func(p *panel, _ *vroverlay.Event) {
			p.handle(vroverlay.Event{Kind: vroverlay.Hidden}, now, desktopinput.Target{})
		},
		"focus lost": func(p *panel, _ *vroverlay.Event) {
			p.handle(vroverlay.Event{Kind: vroverlay.FocusLost}, now, desktopinput.Target{})
		},
		"stale":             func(_ *panel, up *vroverlay.Event) { up.Age = time.Second },
		"before activation": func(_ *panel, up *vroverlay.Event) { up.Age = time.Millisecond },
		"invalid point":     func(_ *panel, up *vroverlay.Event) { up.X = float32(math.NaN()) },
		"duplicate down":    func(p *panel, _ *vroverlay.Event) { p.handle(pointer(vroverlay.Down, 2), now, desktopinput.Target{}) },
	}
	for name, alter := range cases {
		t.Run(name, func(t *testing.T) {
			p := newPanel()
			p.activate(testTarget(), now)
			p.handle(pointer(vroverlay.Down, 2), now, desktopinput.Target{})
			up := pointer(vroverlay.Up, 2)
			alter(p, &up)
			if got := p.handle(up, now, desktopinput.Target{}); got != nil {
				t.Fatal("dispatched interrupted gesture")
			}
		})
	}
}

func TestPanelLeavingOrChangingCursorCancelsUntilFreshPress(t *testing.T) {
	now := time.Unix(100, 0)
	otherCursor := pointer(vroverlay.Move, 2)
	otherCursor.Cursor++
	for _, test := range []struct {
		name string
		move vroverlay.Event
	}{
		{"leaves panel", vroverlay.Event{Kind: vroverlay.Move, X: -1, Y: -1, Cursor: 2}},
		{"crosses another button", pointer(vroverlay.Move, 1)},
		{"changes cursor", otherCursor},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := newPanel()
			p.activate(testTarget(), now)
			p.handle(pointer(vroverlay.Down, 2), now, testTarget())
			p.handle(test.move, now, testTarget())
			p.handle(pointer(vroverlay.Move, 2), now, testTarget())
			if got := p.handle(pointer(vroverlay.Up, 2), now, testTarget()); got != nil {
				t.Fatal("returning pointer dispatched canceled gesture")
			}
			p.handle(pointer(vroverlay.Down, 2), now, testTarget())
			if got := p.handle(pointer(vroverlay.Up, 2), now, testTarget()); got == nil || got.action != desktopinput.Paste {
				t.Fatal("fresh gesture did not recover", got)
			}
		})
	}
}
func TestPanelRequiresTargetAndExplicitlyRecaptures(t *testing.T) {
	now := time.Unix(100, 0)
	p := newPanel()
	p.activate(desktopinput.Target{}, now)
	p.handle(pointer(vroverlay.Down, 2), now, testTarget())
	if p.handle(pointer(vroverlay.Up, 2), now, testTarget()) != nil {
		t.Fatal("no target sent")
	}
	p.handle(pointer(vroverlay.Down, 7), now, testTarget())
	if p.handle(pointer(vroverlay.Up, 7), now, testTarget()) != nil || p.target != testTarget() {
		t.Fatal("capture sent input or failed")
	}
	p.deactivate()
	p.handle(pointer(vroverlay.Down, 2), now, testTarget())
	if p.handle(pointer(vroverlay.Up, 2), now, testTarget()) != nil {
		t.Fatal("hidden action")
	}
}

func TestPanelRendering(t *testing.T) {
	r, err := newPadRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	p := newPanel()
	p.activate(testTarget(), time.Unix(100, 0))
	p.hover = 2
	im := r.render(p)
	if im.Bounds() != image.Rect(0, 0, panelWidth, panelHeight) || r.thumbnail().Bounds() != image.Rect(0, 0, 128, 128) {
		t.Fatal("texture dimensions")
	}
	if im.RGBAAt(0, 0) != padBackground || im.RGBAAt(520+14, 136+1) != padAccent {
		t.Fatal("background/focus color")
	}
	long := fitted(r.faces[24], "A very long application or failure detail "+string(make([]byte, 2048)), 200)
	if font.MeasureString(r.faces[24], long).Ceil() > 200 {
		t.Fatal("text overflows")
	}
	path := filepath.Join(t.TempDir(), "clipboard.png")
	if preview := os.Getenv("SNOOFER_VRCLIPBOARD_PREVIEW"); preview != "" {
		path = preview
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, im); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
