package nowplaying

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
	"time"

	"sound-snoofer/internal/mediasessions"
	"sound-snoofer/snoofer"
)

type fakeWindows struct {
	sessions []mediasessions.Session
	art      map[string][]byte
	artCalls int
	commands []string
	decline  bool
}

func (f *fakeWindows) Snapshot() ([]mediasessions.Session, error) {
	return append([]mediasessions.Session(nil), f.sessions...), nil
}

func (f *fakeWindows) Art(id string) ([]byte, error) {
	f.artCalls++
	return f.art[id], nil
}

func (f *fakeWindows) Command(id string, op mediasessions.Op, value int64) error {
	f.commands = append(f.commands, id+":"+[]string{"", "play", "pause", "toggle", "next", "prev", "seek"}[op]+":"+time.Duration(value).String())
	if f.decline {
		return mediasessions.ErrDeclined
	}
	return nil
}

func (f *fakeWindows) Close() error { return nil }

func cover(t *testing.T) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 300, 200))
	for i := range im.Pix {
		im.Pix[i] = 200
	}
	im.Set(0, 0, color.RGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := jpeg.Encode(&b, im, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

type rig struct {
	t        *testing.T
	w        *worker
	win      *fakeWindows
	controls *snoofer.Controls
	commands chan snoofer.Request
	sent     []browserCommand
	now      time.Time
}

func newRig(t *testing.T, live bool) *rig {
	r := &rig{t: t, win: &fakeWindows{art: map[string][]byte{"Spotify.exe": cover(t)}}, controls: snoofer.NewControls(), commands: make(chan snoofer.Request, 8), now: time.Unix(1_000_000, 0)}
	r.w = newWorker(snoofer.Services{Controls: r.controls, Live: live}, func() (windowsSource, error) { return r.win, nil }, defaultTiming)
	r.w.send = func(_ string, c browserCommand) error {
		r.sent = append(r.sent, c)
		return nil
	}
	return r
}

// tick advances time past the Windows poll and publishes.
func (r *rig) tick(d time.Duration) {
	r.now = r.now.Add(d)
	r.w.pollAt = time.Time{}
	r.w.step(r.now)
	r.w.publish(r.commands, r.now)
}

func (r *rig) control(id string) snoofer.Control {
	for _, c := range r.controls.Snapshot() {
		if c.ID == id {
			return c
		}
	}
	return snoofer.Control{}
}

func (r *rig) members() []snoofer.Control {
	var out []snoofer.Control
	for _, c := range r.controls.Snapshot() {
		if c.Collection == "nowplaying.sessions" {
			out = append(out, c)
		}
	}
	return out
}

func (r *rig) press(id, op string, delta int) {
	r.w.handle(snoofer.Request{ID: id, Operation: op, Delta: delta}, r.now)
	r.w.publish(r.commands, r.now)
}

func spotify(status string, position int64, updated time.Time) mediasessions.Session {
	return mediasessions.Session{ID: "Spotify.exe", App: "Spotify.exe", Title: "Song", Artist: "Band", Status: status, PositionMs: position, DurationMs: 200_000,
		UpdatedMs: updated.UnixMilli(), Rate: 1, CanPlay: true, CanPause: true, CanNext: true, CanPrev: true, CanSeek: true, ArtKey: "k1"}
}

func TestWindowsSessionsFocusAndArt(t *testing.T) {
	r := newRig(t, true)
	r.win.sessions = []mediasessions.Session{spotify("playing", 60_000, r.now), {ID: "Brave", App: "Brave", Title: "Video", Status: "paused", CanPlay: true}}
	r.tick(0)
	if m := r.members(); len(m) != 2 {
		t.Fatalf("members %+v", m)
	}
	dial := r.control("nowplaying.dial")
	if dial.ShortLabel != "Song" || dial.Value != "1:00 / 3:20" || dial.Artwork == "" {
		t.Fatalf("dial %+v", dial)
	}
	r.tick(10 * time.Second) // Interpolated while playing.
	if v := r.control("nowplaying.dial").Value; v != "1:10 / 3:20" {
		t.Fatal("interpolation", v)
	}
	if r.win.artCalls != 1 {
		t.Fatal("artwork fetched again for the same track", r.win.artCalls)
	}
	if r.control("nowplaying.mute").Hidden != true || r.control("nowplaying.next").Hidden {
		t.Fatal("transport visibility")
	}
	if app := appName("Microsoft.ZuneMusic_8wekyb3d8bbwe!Microsoft.ZuneMusic"); app != "Media Player" {
		t.Fatal(app)
	}
}

func TestCommandsPendingAndObserved(t *testing.T) {
	r := newRig(t, true)
	r.win.sessions = []mediasessions.Session{spotify("playing", 60_000, r.now)}
	r.tick(0)
	r.press("nowplaying.dial", "adjust", 3)
	if len(r.win.commands) != 1 || r.win.commands[0] != "Spotify.exe:seek:75µs" {
		t.Fatal("seek", r.win.commands)
	}
	if r.control("nowplaying.dial").Status != "Pending" {
		t.Fatal("seek not pending")
	}
	r.win.sessions = []mediasessions.Session{spotify("playing", 75_000, r.now)}
	r.tick(100 * time.Millisecond)
	if s := r.control("nowplaying.dial").Status; s != "" {
		t.Fatal("seek not observed", s)
	}
	r.press("nowplaying.toggle", "press", 0)
	r.tick(time.Second)
	if r.control("nowplaying.toggle").Value != "Playing" || r.control("nowplaying.dial").Status != "Pending" {
		t.Fatal("toggle state", r.control("nowplaying.toggle").Value, r.control("nowplaying.dial").Status)
	}
	r.tick(3 * time.Second) // Never paused: reported, not retried.
	if s := r.control("nowplaying.dial").Status; s != "No response" || len(r.win.commands) != 2 {
		t.Fatal("unobserved toggle", s, r.win.commands)
	}
	r.win.decline = true
	r.press("nowplaying.next", "press", 0)
	if s := r.control("nowplaying.dial").Status; s != "Declined" {
		t.Fatal("declined", s)
	}
}

func TestBrowserTabsReplaceItsWindowsSession(t *testing.T) {
	r := newRig(t, true)
	r.win.sessions = []mediasessions.Session{spotify("paused", 0, r.now), {ID: "Brave", App: "Brave", Title: "One tab", Status: "playing", CanPlay: true}}
	r.tick(0)
	if len(r.members()) != 2 {
		t.Fatal("without the extension Brave is one session")
	}
	art := thumbFor(t)
	r.w.browser(browserUpdate{Browser: "Brave", Version: "1", Connected: true, Sessions: []browserSession{
		{ID: "1:0", Tab: 1, Site: "youtube.com", Title: "Video A", State: "playing", ArtKey: "a", Art: art, CanNext: true, DurationMs: 100_000, CanSeek: true},
		{ID: "2:0", Tab: 2, Site: "podcasts.example", Title: "Episode", State: "playing", ArtKey: "b"},
	}})
	r.tick(time.Second)
	members := r.members()
	if len(members) != 3 {
		t.Fatalf("members %d", len(members))
	}
	for _, m := range members {
		if m.Label == "One tab" {
			t.Fatal("Brave's Windows session shown alongside its tabs")
		}
	}
	// Pressing a background tab toggles it and focuses it.
	var episode snoofer.Control
	for _, m := range members {
		if m.Label == "Episode" {
			episode = m
		}
	}
	r.press(episode.ID, "press", 0)
	if len(r.sent) != 1 || r.sent[0].ID != "2:0" || r.sent[0].Op != "toggle" || r.control("nowplaying.focus").Value != episode.ID {
		t.Fatal("tab press", r.sent, r.control("nowplaying.focus").Value)
	}
	if r.control("nowplaying.mute").Hidden {
		t.Fatal("tabs can mute")
	}
	// Art arrives once and is kept while its key is unchanged.
	r.w.browser(browserUpdate{Browser: "Brave", Version: "1", Connected: true, Sessions: []browserSession{{ID: "1:0", Site: "youtube.com", Title: "Video A", State: "playing", ArtKey: "a"}}})
	r.tick(time.Second)
	if r.members()[0].Artwork != art {
		t.Fatal("tab artwork lost")
	}
	// Disconnecting brings Brave's Windows session back.
	r.w.browser(browserUpdate{Browser: "Brave"})
	r.tick(time.Second)
	if len(r.members()) != 2 {
		t.Fatal("Windows session not restored")
	}
}

func TestFocusFollowsNewPlaybackUnlessChosen(t *testing.T) {
	r := newRig(t, true)
	a := mediasessions.Session{ID: "A", App: "A.exe", Title: "A", Status: "playing", CanPlay: true}
	b := mediasessions.Session{ID: "B", App: "B.exe", Title: "B", Status: "paused", CanPlay: true}
	r.win.sessions = []mediasessions.Session{a, b}
	r.tick(0)
	focus := func() string { return r.control("nowplaying.dial").ShortLabel }
	if focus() != "A" {
		t.Fatal("initial focus", focus())
	}
	b.Status = "playing"
	r.win.sessions = []mediasessions.Session{a, b}
	r.tick(time.Second)
	if focus() != "B" {
		t.Fatal("focus did not follow new playback", focus())
	}
	// Choosing A holds focus against new playback for 30 s.
	r.press("nowplaying.focus", "set", 0)
	r.w.handle(snoofer.Request{ID: "nowplaying.focus", Operation: "set", Value: controlID("windows:A")}, r.now)
	b.Status = "paused"
	r.win.sessions = []mediasessions.Session{a, b}
	r.tick(time.Second)
	b.Status = "playing"
	r.win.sessions = []mediasessions.Session{a, b}
	r.tick(time.Second)
	if focus() != "A" {
		t.Fatal("chosen focus overridden", focus())
	}
}

func TestPreviewSendsNothing(t *testing.T) {
	r := newRig(t, false)
	r.win.sessions = []mediasessions.Session{spotify("playing", 0, r.now)}
	r.tick(0)
	r.press("nowplaying.toggle", "press", 0)
	if len(r.win.commands) != 0 || r.control("nowplaying.status").Value != "Preview" {
		t.Fatal("preview sent", r.win.commands)
	}
}

func TestStatusView(t *testing.T) {
	r := newRig(t, true)
	r.win.sessions = []mediasessions.Session{spotify("playing", 0, r.now)}
	r.tick(0)
	var view statusView
	if err := json.Unmarshal(r.control("nowplaying.status").ViewData, &view); err != nil {
		t.Fatal(err)
	}
	if view.Windows != "Connected" || len(view.Sessions) != 1 || !view.Sessions[0].Focused || view.Sessions[0].App != "Spotify" {
		t.Fatalf("%+v", view)
	}
}

func thumbFor(t *testing.T) string {
	t.Helper()
	art, err := thumbnail(cover(t))
	if err != nil {
		t.Fatal(err)
	}
	return art
}
