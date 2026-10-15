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

func startBridgeRig(t *testing.T, windowsSessions ...mediasessions.Session) *bridgeRig {
	t.Helper()
	return startBridgeRigOn(t, freePort(t), windowsSessions...)
}

func startBridgeRigOn(t *testing.T, port int, windowsSessions ...mediasessions.Session) *bridgeRig {
	t.Helper()
	r := &bridgeRig{t: t, controls: snoofer.NewControls(), windows: &fakeWindows{sessions: windowsSessions}, port: port}
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
	for version, want := range map[int]string{protocol + 1: "Update Snoofer", protocol - 1: "Update the Snoofer Media extension"} {
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

func TestBridgeSessionsCommandsAndReconnect(t *testing.T) {
	// Firefox's own Windows session has an unrecognisable app ID; its title
	// matching a tab hides it.
	r := startBridgeRig(t, mediasessions.Session{ID: "308046B0AF4A39CB", Instance: "308046B0AF4A39CB", App: "308046B0AF4A39CB", Title: "Video", Status: "playing", CanPlay: true})
	ws := r.connect("moz-extension://x", "Firefox", protocol)
	r.wait("connected", func(v statusView) bool { return len(v.Browsers) == 1 && v.Browsers[0].Name == "Firefox" })
	if err := ws.WriteJSON(report{Type: "sessions", Sessions: []browserSession{{ID: "7:0", Instance: "7:0", Tab: 7, Site: "youtube.com", Title: "Video", State: "playing", CanNext: true}}}); err != nil {
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
	if err := ws.Close(); err != nil {
		t.Fatal(err)
	}
	r.wait("disconnected before reconnect", func(v statusView) bool { return len(v.Browsers) == 0 })
	second := r.connect("moz-extension://x", "Firefox", protocol)
	r.wait("reconnected", func(v statusView) bool { return len(v.Browsers) == 1 })
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

func TestBridgeSkipsMalformedReports(t *testing.T) {
	r := startBridgeRig(t)
	ws := r.connect("moz-extension://x", "Firefox", protocol)
	r.wait("connected", func(v statusView) bool { return len(v.Browsers) == 1 })
	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"sessions","sessions":[{"title":{}}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteJSON(report{Type: "sessions", Sessions: []browserSession{{ID: "7:0", Instance: "7:0", Tab: 7, Site: "youtube.com", Title: "Video", State: "playing"}}}); err != nil {
		t.Fatal(err)
	}
	r.wait("session after malformed report", func(v statusView) bool { return len(v.Browsers) == 1 && len(v.Sessions) == 1 })
}

func TestBridgeRetriesBusyPortAndStopsAfterClosing(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	r := startBridgeRigOn(t, busy.Addr().(*net.TCPAddr).Port)
	r.wait("busy port reported", func(v statusView) bool { return v.Bridge.Error != "" })
	busy.Close()
	r.wait("listening once the port is free", func(v statusView) bool { return v.Bridge.Error == "" })
	r.connect("moz-extension://x", "Firefox", protocol)
	r.wait("connected", func(v statusView) bool { return len(v.Browsers) == 1 })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.instance.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	// A restarted plugin must find the port free as soon as Stop returns.
	again, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", r.port))
	if err != nil {
		t.Fatal("port still held after Stop:", err)
	}
	again.Close()
}

func TestBridgeRejectsSecondLiveConnection(t *testing.T) {
	for _, scheme := range []string{"chrome-extension://", "moz-extension://"} {
		for _, claimant := range []string{"original", "other"} {
			t.Run(scheme+claimant, func(t *testing.T) {
				r := startBridgeRig(t)
				original := r.connect(scheme+"original", "Browser", protocol)
				r.wait("original connected", func(v statusView) bool { return len(v.Browsers) == 1 })
				if err := original.WriteJSON(report{Type: "sessions", Sessions: []browserSession{{ID: "7:0", Instance: "7:0", Tab: 7, Title: "Original", State: "playing", CanNext: true}}}); err != nil {
					t.Fatal(err)
				}
				r.wait("original session", func(v statusView) bool { return len(v.Sessions) == 1 })
				other := r.connect(scheme+claimant, "Browser", protocol)
				if err := other.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
				var refused refusal
				if err := other.ReadJSON(&refused); err != nil || refused.Type != "refused" || !strings.Contains(refused.Reason, "already active") {
					t.Fatalf("second live connection was not refused: %+v, %v", refused, err)
				}
				var closed *websocket.CloseError
				if _, _, err := other.ReadMessage(); !errors.As(err, &closed) || closed.Code != 4003 {
					t.Fatalf("refused connection remained open: %v", err)
				}
				r.wait("original retained and refusal visible", func(v statusView) bool {
					return len(v.Browsers) == 1 && len(v.Sessions) == 1 && v.Sessions[0].Title == "Original" && v.Bridge.Refused == refused.Reason
				})
				var next snoofer.Control
				for _, c := range r.controls.Snapshot() {
					if c.ID == "nowplaying.next" {
						next = c
					}
				}
				if err := r.controls.Dispatch(context.Background(), snoofer.Request{ID: next.ID, Revision: next.Revision, Operation: "press"}); err != nil {
					t.Fatal(err)
				}
				if err := original.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
				var command browserCommand
				if err := original.ReadJSON(&command); err != nil || command.ID != "7:0" || command.Op != "next" {
					t.Fatalf("original no longer receives commands: %+v, %v", command, err)
				}
				if err := original.Close(); err != nil {
					t.Fatal(err)
				}
				r.wait("original disconnected", func(v statusView) bool { return len(v.Browsers) == 0 })
				r.connect(scheme+"other", "Browser", protocol)
				r.wait("new origin accepted after disconnect", func(v statusView) bool { return len(v.Browsers) == 1 && v.Bridge.Refused == "" })
			})
		}
	}
}

func TestBridgeCapturedGenerationNeverUsesReplacement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan browserUpdate, 16)
	port := freePort(t)
	b, err := listen(ctx, port, updates)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); b.close() })
	connect := func() *websocket.Conn {
		t.Helper()
		ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d/nowplaying", port), http.Header{"Origin": []string{"moz-extension://same"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { ws.Close() })
		if err := ws.WriteJSON(hello{Type: "hello", Protocol: protocol, Browser: "Firefox"}); err != nil {
			t.Fatal(err)
		}
		return ws
	}
	first := connect()
	old := <-updates
	duplicate := connect()
	if err := duplicate.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var refused refusal
	if err := duplicate.ReadJSON(&refused); err != nil || refused.Type != "refused" {
		t.Fatalf("duplicate not refused: %+v, %v", refused, err)
	}
	if b.generation("Firefox") != old.Generation {
		t.Fatal("refusal changed incumbent generation")
	}
	if err := b.send("Firefox", browserCommand{Type: "command", ID: "7:0", Instance: "document", Generation: old.Generation, Op: "next"}); err != nil {
		t.Fatal("incumbent command refused", err)
	}
	if err := first.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var incumbentCommand browserCommand
	if err := first.ReadJSON(&incumbentCommand); err != nil || incumbentCommand.Op != "next" {
		t.Fatalf("incumbent lost its command: %+v, %v", incumbentCommand, err)
	}
	var closed *websocket.CloseError
	if _, _, err := duplicate.ReadMessage(); !errors.As(err, &closed) || closed.Code != 4003 {
		t.Fatalf("duplicate received a command or remained open: %v", err)
	}
	select {
	case update := <-updates:
		t.Fatalf("refused connection published an update: %+v", update)
	default:
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	disconnected := <-updates
	if disconnected.Connected || disconnected.Generation != old.Generation {
		t.Fatalf("wrong disconnect: %+v", disconnected)
	}
	second := connect()
	current := <-updates
	if old.Generation == 0 || current.Generation != old.Generation+1 {
		t.Fatal("refusal consumed a generation", old, current)
	}
	if err := b.send("Firefox", browserCommand{Type: "command", ID: "7:0", Instance: "document", Generation: old.Generation, Op: "next"}); !errors.Is(err, errBrowserChanged) {
		t.Fatal("old command accepted", err)
	}
	if err := b.send("Firefox", browserCommand{Type: "command", ID: "7:0", Instance: "document", Generation: current.Generation, Op: "mute", Value: 0}); err != nil {
		t.Fatal(err)
	}
	if err := second.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := second.ReadJSON(&got); err != nil {
		t.Fatal(err)
	}
	if string(got["value"]) != "0" || string(got["instance"]) != `"document"` || got["Generation"] != nil {
		t.Fatal(got)
	}
}

func TestBridgeConcurrentClaimsKeepOneConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan browserUpdate, 16)
	port := freePort(t)
	b, err := listen(ctx, port, updates)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); b.close() })
	var sockets [2]*websocket.Conn
	for n := range sockets {
		sockets[n], _, err = websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d/nowplaying", port), http.Header{"Origin": []string{"chrome-extension://same"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { sockets[n].Close() })
	}
	start := make(chan struct{})
	written := make(chan error, len(sockets))
	for _, ws := range sockets {
		go func() {
			<-start
			written <- ws.WriteJSON(hello{Type: "hello", Protocol: protocol, Browser: "Chrome"})
		}()
	}
	close(start)
	for range sockets {
		if err := <-written; err != nil {
			t.Fatal(err)
		}
	}
	connected := <-updates
	if connected.Generation != 1 || !connected.Connected {
		t.Fatalf("first connection: %+v", connected)
	}
	// A command and a refusal identify the winner without relying on scheduling.
	if err := b.send("Chrome", browserCommand{Type: "command", Generation: connected.Generation, Op: "next"}); err != nil {
		t.Fatal(err)
	}
	commands, refusals := 0, 0
	for _, ws := range sockets {
		if err := ws.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		var message struct{ Type string }
		if err := ws.ReadJSON(&message); err != nil {
			t.Fatal(err)
		}
		switch message.Type {
		case "command":
			commands++
		case "refused":
			refusals++
		default:
			t.Fatalf("unexpected message: %+v", message)
		}
	}
	if commands != 1 || refusals != 1 || b.generation("Chrome") != connected.Generation {
		t.Fatalf("commands=%d refusals=%d generation=%d", commands, refusals, b.generation("Chrome"))
	}
	select {
	case update := <-updates:
		t.Fatalf("second claimant published an update: %+v", update)
	default:
	}
}

func TestBrowserSessionsRequireBoundedInstance(t *testing.T) {
	for _, instance := range []string{"", strings.Repeat("a", 65)} {
		if err := validBrowserSessions([]browserSession{{ID: "7:0", Instance: instance}}); err == nil {
			t.Fatal("invalid instance accepted")
		}
	}
	if err := validBrowserSessions([]browserSession{{ID: "7:0", Instance: "1"}}); err != nil {
		t.Fatal(err)
	}
}
