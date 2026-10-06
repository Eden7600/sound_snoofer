package nowplaying

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"sound-snoofer/snoofer"
)

type bridgeRig struct {
	t        *testing.T
	controls *snoofer.Controls
	port     int
	mu       sync.Mutex
	saved    Settings
	instance snoofer.Instance
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
	r := &bridgeRig{t: t, controls: snoofer.NewControls(), port: freePort(t)}
	raw, _ := json.Marshal(Settings{Port: r.port})
	services := snoofer.Services{Controls: r.controls, Live: true, SaveSettings: func(id string, _, next json.RawMessage) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.saved = Settings{}
		return json.Unmarshal(next, &r.saved)
	}}
	open := func() (windowsSource, error) { return &fakeWindows{}, nil }
	instance, err := start(context.Background(), services, raw, open, timing{poll: 5 * time.Millisecond, windows: 20 * time.Millisecond, observe: time.Second, retry: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	r.instance = instance
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := instance.Stop(ctx); err != nil {
			t.Error("stop:", err)
		}
	})
	return r
}

func (r *bridgeRig) token() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saved.Token
}

func (r *bridgeRig) dial(origin string) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	return websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d/nowplaying", r.port), header)
}

func (r *bridgeRig) connect(browser string) *websocket.Conn {
	r.t.Helper()
	ws, _, err := r.dial("chrome-extension://abcdefghijklmnop")
	if err != nil {
		r.t.Fatal(err)
	}
	if err := ws.WriteJSON(hello{Type: "hello", Token: r.token(), Version: "1", Browser: browser}); err != nil {
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

func TestBridgeRefusesPagesAndBadTokens(t *testing.T) {
	r := startBridgeRig(t)
	if len(r.token()) != 64 {
		t.Fatal("token not created on first start", r.token())
	}
	if _, resp, err := r.dial("https://evil.example"); err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatal("page origin accepted", err)
	}
	ws, _, err := r.dial("chrome-extension://abc")
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if err := ws.WriteJSON(hello{Type: "hello", Token: "stale", Browser: "Brave"}); err != nil {
		t.Fatal(err)
	}
	var closeErr *websocket.CloseError
	if _, _, err := ws.ReadMessage(); !errors.As(err, &closeErr) || closeErr.Code != 4001 {
		t.Fatal("stale token not closed with 4001:", err)
	}
	r.wait("refusal reported", func(v statusView) bool { return strings.Contains(v.Bridge.Refused, "token is out of date") })
}

func TestBridgeSessionsCommandsAndReplacement(t *testing.T) {
	r := startBridgeRig(t)
	ws := r.connect("Brave")
	r.wait("connected", func(v statusView) bool { return len(v.Browsers) == 1 && v.Browsers[0].Name == "Brave" })
	if err := ws.WriteJSON(report{Type: "sessions", Sessions: []browserSession{{ID: "7:0", Tab: 7, Site: "youtube.com", Title: "Video", State: "playing", CanNext: true}}}); err != nil {
		t.Fatal(err)
	}
	v := r.wait("session", func(v statusView) bool { return len(v.Sessions) == 1 })
	if v.Sessions[0].Title != "Video" || v.Sessions[0].App != "Brave · youtube.com" || !v.Sessions[0].CanMute {
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
	second := r.connect("Brave")
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("replaced connection still open")
	}
	r.wait("second connection", func(v statusView) bool { return len(v.Browsers) == 1 })
	// Too many sessions closes the connection and drops its tabs.
	many := make([]browserSession, maxBrowserSessions+1)
	for n := range many {
		many[n] = browserSession{ID: fmt.Sprintf("%d:0", n), State: "paused"}
	}
	if err := second.WriteJSON(report{Type: "sessions", Sessions: many}); err != nil {
		t.Fatal(err)
	}
	r.wait("oversized report refused", func(v statusView) bool { return len(v.Browsers) == 0 && strings.Contains(v.Bridge.Refused, "at most") })
}

func TestBridgeTokenResetDropsConnections(t *testing.T) {
	r := startBridgeRig(t)
	old := r.token()
	ws := r.connect("Chrome")
	r.wait("connected", func(v statusView) bool { return len(v.Browsers) == 1 })
	var reset snoofer.Control
	for _, c := range r.controls.Snapshot() {
		if c.ID == "nowplaying.token-reset" {
			reset = c
		}
	}
	if err := r.controls.Dispatch(context.Background(), snoofer.Request{ID: reset.ID, Revision: reset.Revision, Operation: "press"}); err != nil {
		t.Fatal(err)
	}
	r.wait("dropped", func(v statusView) bool { return len(v.Browsers) == 0 })
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("connection survived the token reset")
	}
	if r.token() == old || len(r.token()) != 64 {
		t.Fatal("token not replaced")
	}
}
