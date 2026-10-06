package nowplaying

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"sound-snoofer/internal/mediasessions"
	"sound-snoofer/snoofer"
)

type bridgeRig struct {
	t        *testing.T
	controls *snoofer.Controls
	windows  *fakeWindows
	port     int
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func startBridgeRig(t *testing.T) *bridgeRig {
	t.Helper()
	r := &bridgeRig{t: t, controls: snoofer.NewControls(), windows: &fakeWindows{}, port: freePort(t)}
	// The legacy token from the first build still loads, and is ignored.
	raw, _ := json.Marshal(Settings{Port: r.port, Token: "legacy"})
	services := snoofer.Services{Controls: r.controls, Live: true, SaveSettings: func(string, json.RawMessage, json.RawMessage) error {
		t.Error("settings saved; the bridge needs none")
		return errors.New("unexpected save")
	}}
	open := func() (windowsSource, error) { return r.windows, nil }
	instance, err := start(context.Background(), services, raw, open, timing{poll: 5 * time.Millisecond, windows: 20 * time.Millisecond, observe: time.Second, retry: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := instance.Stop(ctx); err != nil {
			t.Error("stop:", err)
		}
	})
	return r
}

func (r *bridgeRig) dial(origin string) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	return websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d/nowplaying", r.port), header)
}

func (r *bridgeRig) connect(origin, browser string, version int) *websocket.Conn {
	r.t.Helper()
	ws, _, err := r.dial(origin)
	if err != nil {
		r.t.Fatal(err)
	}
	if err := ws.WriteJSON(hello{Type: "hello", Protocol: version, Version: "1.0.0", Browser: browser}); err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(func() { ws.Close() })
	return ws
}

func (r *bridgeRig) status() statusView {
	var v statusView
	for _, c := range r.controls.Snapshot() {
		if c.ID == "nowplaying.status" {
			_ = json.Unmarshal(c.ViewData, &v)
		}
	}
	return v
}

func (r *bridgeRig) wait(what string, ready func(statusView) bool) statusView {
	r.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		v := r.status()
		if ready(v) {
			return v
		}
		if time.Now().After(deadline) {
			r.t.Fatalf("timed out: %s (%+v)", what, v)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestBridgeAcceptsOnlyExtensions(t *testing.T) {
	r := startBridgeRig(t)
	for _, origin := range []string{"https://evil.example", "http://127.0.0.1", ""} {
		if _, resp, err := r.dial(origin); err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("origin %q accepted: %v", origin, err)
		}
	}
	// Chrome-family and Firefox extensions connect without any setup.
	r.connect("chrome-extension://abcdefghijklmnop", "Brave", protocol)
	r.connect("moz-extension://0b3c9d1e-1111-2222-3333-444455556666", "Firefox", protocol)
	r.wait("both connected", func(v statusView) bool { return len(v.Browsers) == 2 })
}

func TestBridgeRefusesOtherProtocols(t *testing.T) {
	r := startBridgeRig(t)
	for version, want := range map[int]string{protocol + 1: "Update Snoofer", 0: "Update the Snoofer Media extension"} {
		ws := r.connect("chrome-extension://abc", "Chrome", version)
		if err := ws.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatal(err)
		}
		var m refusal
		if err := ws.ReadJSON(&m); err != nil || m.Type != "refused" || !strings.HasPrefix(m.Reason, want) {
			t.Fatalf("protocol %d: %+v %v", version, m, err)
		}
		var closeErr *websocket.CloseError
		if _, _, err := ws.ReadMessage(); !errors.As(err, &closeErr) || closeErr.Code != 4002 {
			t.Fatal("not closed with 4002:", err)
		}
		r.wait("refusal reported", func(v statusView) bool { return strings.HasPrefix(v.Bridge.Refused, want) && len(v.Browsers) == 0 })
	}
}

func TestBridgeSessionsCommandsAndReplacement(t *testing.T) {
	r := startBridgeRig(t)
	// Firefox's own Windows session has an unrecognisable app ID; its title
	// matching a tab hides it.
	r.windows.sessions = []mediasessions.Session{{ID: "308046B0AF4A39CB", App: "308046B0AF4A39CB", Title: "Video", Status: "playing", CanPlay: true}}
	ws := r.connect("moz-extension://x", "Firefox", protocol)
	r.wait("connected", func(v statusView) bool { return len(v.Browsers) == 1 && v.Browsers[0].Name == "Firefox" })
	if err := ws.WriteJSON(report{Type: "sessions", Sessions: []browserSession{{ID: "7:0", Tab: 7, Site: "youtube.com", Title: "Video", State: "playing", CanNext: true}}}); err != nil {
		t.Fatal(err)
	}
	v := r.wait("session", func(v statusView) bool { return len(v.Sessions) == 1 && v.Sessions[0].Source == "Firefox" })
	if v.Sessions[0].App != "Firefox · youtube.com" || !v.Sessions[0].CanMute {
		t.Fatalf("%+v", v.Sessions[0])
	}
	// Next goes to the focused tab over the WebSocket.
	var next snoofer.Control
	for _, c := range r.controls.Snapshot() {
		if c.ID == "nowplaying.next" {
			next = c
		}
	}
	if err := r.controls.Dispatch(context.Background(), snoofer.Request{ID: next.ID, Revision: next.Revision, Operation: "press"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var cmd browserCommand
	if err := ws.ReadJSON(&cmd); err != nil || cmd.ID != "7:0" || cmd.Op != "next" {
		t.Fatal("command", cmd, err)
	}
	// A newer connection from the same browser replaces the first.
	second := r.connect("moz-extension://x", "Firefox", protocol)
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("replaced connection still open")
	}
	r.wait("second connection", func(v statusView) bool { return len(v.Browsers) == 1 })
	// Too many sessions closes the connection and drops its tabs; the
	// Windows session returns.
	many := make([]browserSession, maxBrowserSessions+1)
	for n := range many {
		many[n] = browserSession{ID: fmt.Sprintf("%d:0", n), State: "paused"}
	}
	if err := second.WriteJSON(report{Type: "sessions", Sessions: many}); err != nil {
		t.Fatal(err)
	}
	r.wait("oversized report refused", func(v statusView) bool {
		return len(v.Browsers) == 0 && strings.Contains(v.Bridge.Refused, "at most") && len(v.Sessions) == 1 && v.Sessions[0].Source == "windows"
	})
}
