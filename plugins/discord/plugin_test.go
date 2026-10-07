package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

// fakeDiscord answers RPC commands on the server end of an in-memory pipe.
type fakeDiscord struct {
	mu       sync.Mutex
	conn     net.Conn
	commands []string // Commands received, in order.
	mute     bool
	deaf     bool
	channel  *string
	video    bool
	frozen   bool         // Accept writes without reporting changes.
	deny     bool         // Deny AUTHORIZE.
	out      chan message // Replies, written by one goroutine like a buffered OS pipe.
}

func (d *fakeDiscord) serve(conn net.Conn) {
	d.conn = conn
	out := make(chan message, 256)
	d.out = out
	go func() {
		for m := range out {
			if writeFrame(conn, opFrame, m) != nil {
				return
			}
		}
	}()
	go func() {
		for {
			f, err := readFrame(conn)
			if err != nil {
				close(out)
				return
			}
			if f.op == opHandshake {
				d.reply(message{Cmd: "DISPATCH", Evt: "READY", Data: raw(map[string]any{"v": 1})})
				continue
			}
			var m message
			if json.Unmarshal(f.payload, &m) != nil {
				close(out)
				return
			}
			d.answer(m)
		}
	}()
}

func (d *fakeDiscord) reply(m message) {
	d.out <- m
}

func raw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func (d *fakeDiscord) answer(m message) {
	d.mu.Lock()
	d.commands = append(d.commands, m.Cmd)
	args, _ := json.Marshal(m.Args)
	var a map[string]any
	_ = json.Unmarshal(args, &a)
	settings := func() json.RawMessage { return raw(map[string]any{"mute": d.mute, "deaf": d.deaf}) }
	var data json.RawMessage
	var events []message
	evt := ""
	switch m.Cmd {
	case "AUTHORIZE":
		if d.deny {
			evt, data = "ERROR", raw(rpcError{Code: 5000, Message: "OAuth2 Error"})
		} else {
			data = raw(map[string]any{"code": "abc"})
		}
	case "AUTHENTICATE":
		if a["access_token"] == "" {
			evt, data = "ERROR", raw(rpcError{Code: 4009, Message: "Invalid token"})
		} else {
			data = raw(map[string]any{"user": map[string]any{"id": "7"}})
		}
	case "GET_VOICE_SETTINGS":
		data = settings()
	case "GET_SELECTED_VOICE_CHANNEL":
		if d.channel == nil {
			data = json.RawMessage("null")
		} else {
			data = raw(map[string]any{"id": *d.channel, "name": "General"})
		}
	case "GET_CHANNEL":
		data = raw(map[string]any{"id": a["channel_id"], "name": "General"})
	case "SUBSCRIBE":
		data = raw(map[string]any{"evt": m.Evt})
	case "SET_VOICE_SETTINGS":
		if !d.frozen {
			if v, ok := a["mute"].(bool); ok {
				d.mute = v
			}
			if v, ok := a["deaf"].(bool); ok {
				d.deaf = v
			}
		}
		data = settings()
		if !d.frozen {
			events = append(events, message{Cmd: "DISPATCH", Evt: "VOICE_SETTINGS_UPDATE", Data: settings()})
		}
	case "TOGGLE_VIDEO":
		d.video = !d.video
		data = raw(map[string]any{})
		events = append(events, message{Cmd: "DISPATCH", Evt: "VIDEO_STATE_UPDATE", Data: raw(map[string]any{"active": d.video})})
	case "SELECT_VOICE_CHANNEL":
		d.channel = nil
		data = json.RawMessage("null")
		events = append(events, message{Cmd: "DISPATCH", Evt: "VOICE_CHANNEL_SELECT", Data: raw(map[string]any{"channel_id": nil})})
	}
	d.mu.Unlock()
	d.reply(message{Cmd: m.Cmd, Evt: evt, Nonce: m.Nonce, Data: data})
	for _, e := range events {
		d.reply(e)
	}
}

func (d *fakeDiscord) received() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.commands...)
}

// tokenServer issues rotating tokens; refresh token "bad" is invalid.
func tokenServer(t *testing.T) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("client_secret") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Form.Get("grant_type") == "authorization_code" && r.Form.Get("code") == "abc" && r.Form.Get("redirect_uri") == "http://127.0.0.1":
			_, _ = io.WriteString(w, `{"access_token":"A1","refresh_token":"R1"}`)
		case r.Form.Get("grant_type") == "refresh_token" && r.Form.Get("refresh_token") == "bad":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
		case r.Form.Get("grant_type") == "refresh_token":
			_, _ = io.WriteString(w, `{"access_token":"A2","refresh_token":"R2"}`)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

type harness struct {
	w     *worker
	peer  *fakeDiscord
	saved *Settings
	now   time.Time
	ctx   context.Context
	dials int
}

func newHarness(t *testing.T, settings Settings, live bool) *harness {
	t.Helper()
	h := &harness{peer: &fakeDiscord{}, saved: &Settings{}, now: time.Unix(1000, 0)}
	ctx, cancel := context.WithCancel(context.Background())
	h.ctx = ctx
	tokens := tokenServer(t)
	services := snoofer.Services{Controls: snoofer.NewControls(), Live: live, SaveSettings: func(_ string, _, next json.RawMessage) error {
		return json.Unmarshal(next, h.saved)
	}}
	raw, _ := json.Marshal(settings)
	h.w = newWorker(services, raw, settings, func() (io.ReadWriteCloser, string, error) {
		h.dials++
		client, server := net.Pipe()
		h.peer.serve(server)
		return client, `\\.\pipe\discord-ipc-0`, nil
	}, tokens.Client(), tokens.URL)
	t.Cleanup(func() {
		cancel()
		h.w.closeConn()
		h.w.helpers.Wait()
	})
	return h
}

// settle runs the worker's event handling until it goes quiet.
func (h *harness) settle() {
	for {
		select {
		case in := <-h.w.frames:
			h.w.receive(h.ctx, in, h.now)
		case r := <-h.w.tokens:
			h.w.tokenDone(h.ctx, r, h.now)
		case <-time.After(150 * time.Millisecond):
			return
		}
	}
}

func (h *harness) press(id string) {
	h.w.handle(h.ctx, snoofer.Request{ID: id, Operation: "press"}, h.now)
	h.settle()
	h.w.step(h.ctx, h.now)
}

func (h *harness) control(t *testing.T, id string) snoofer.Control {
	t.Helper()
	for _, c := range h.w.controls(h.now) {
		if c.ID == id {
			return c
		}
	}
	t.Fatal("missing", id)
	return snoofer.Control{}
}

var configured = Settings{ClientID: "123456789", ClientSecret: "secret"}

func ready(t *testing.T, h *harness) {
	t.Helper()
	h.w.step(h.ctx, h.now)
	h.settle()
	if h.w.state == stateUnauthorized {
		h.press("discord.connect")
	}
	h.settle()
	if h.w.state != stateReady {
		t.Fatal("not ready", h.w.state)
	}
}

func TestFramingRoundTrip(t *testing.T) {
	var b bytes.Buffer
	if err := writeFrame(&b, opFrame, message{Cmd: "SUBSCRIBE", Evt: "READY", Nonce: "1"}); err != nil {
		t.Fatal(err)
	}
	f, err := readFrame(&b)
	if err != nil || f.op != opFrame || !strings.Contains(string(f.payload), `"nonce":"1"`) {
		t.Fatal(f, err)
	}
	oversized := []byte{1, 0, 0, 0, 0, 0, 0x10, 0}
	if _, err := readFrame(bytes.NewReader(oversized)); err == nil {
		t.Fatal("accepted an oversized frame")
	}
}

func TestFirstUseApprovalSavesRefreshToken(t *testing.T) {
	channel := "42"
	h := newHarness(t, configured, true)
	h.peer.channel = &channel
	h.w.step(h.ctx, h.now)
	h.settle()
	if c := h.control(t, "discord.connect"); !c.Available || h.w.state != stateUnauthorized {
		t.Fatal("connect", c.Available, h.w.state)
	}
	if c := h.control(t, "discord.mute"); c.Available || c.Status != "Not connected" {
		t.Fatal(c.Available, c.Status)
	}
	h.press("discord.connect")
	h.settle()
	if h.w.state != stateReady || h.saved.RefreshToken != "R1" {
		t.Fatal(h.w.state, h.saved.RefreshToken)
	}
	if c := h.control(t, "discord.channel"); c.Value != "General" {
		t.Fatal(c.Value)
	}
	if c := h.control(t, "discord.mute"); !c.Available || c.Value != "Off" {
		t.Fatal(c.Available, c.Value)
	}
	published, _ := json.Marshal(h.w.controls(h.now))
	for _, secret := range []string{"secret", "R1", "A1"} {
		if strings.Contains(string(published), secret) {
			t.Fatalf("controls expose %q", secret)
		}
	}
}

func TestDeniedApproval(t *testing.T) {
	h := newHarness(t, configured, true)
	h.peer.deny = true
	h.w.step(h.ctx, h.now)
	h.settle()
	h.press("discord.connect")
	if h.w.state != stateUnauthorized || h.control(t, "discord.connect").Status != "Failed" {
		t.Fatal(h.w.state, h.control(t, "discord.connect").Status)
	}
}

func TestSavedTokenRefreshesAndRotates(t *testing.T) {
	saved := configured
	saved.RefreshToken = "R1"
	h := newHarness(t, saved, true)
	ready(t, h)
	if h.saved.RefreshToken != "R2" {
		t.Fatal("rotated token not saved", h.saved.RefreshToken)
	}
	for _, cmd := range h.peer.received() {
		if cmd == "AUTHORIZE" {
			t.Fatal("prompted although a token was saved")
		}
	}
}

func TestInvalidGrantRequiresConnect(t *testing.T) {
	saved := configured
	saved.RefreshToken = "bad"
	h := newHarness(t, saved, true)
	h.w.step(h.ctx, h.now)
	h.settle()
	if h.w.state != stateUnauthorized || h.saved.RefreshToken != "" || h.w.settings.RefreshToken != "" {
		t.Fatal(h.w.state, h.saved.RefreshToken)
	}
}

func TestWritesVerifiedOrFailed(t *testing.T) {
	channel := "42"
	h := newHarness(t, configured, true)
	h.peer.channel = &channel
	ready(t, h)
	h.press("discord.mute")
	if c := h.control(t, "discord.mute"); c.Value != "On" || c.Status != "" {
		t.Fatal("mute", c.Value, c.Status)
	}
	h.press("discord.video")
	if c := h.control(t, "discord.video"); c.Value != "On" || c.Status != "" {
		t.Fatal("video", c.Value, c.Status)
	}
	h.peer.frozen = true
	h.press("discord.deafen")
	if c := h.control(t, "discord.deafen"); c.Status != "Pending" {
		t.Fatal("pending", c.Status)
	}
	h.now = h.now.Add(4 * time.Second)
	h.w.step(h.ctx, h.now)
	if c := h.control(t, "discord.deafen"); c.Status != "Failed" || c.Value != "Off" {
		t.Fatal("failed", c.Value, c.Status)
	}
	h.press("discord.leave")
	if c := h.control(t, "discord.channel"); c.Value != "Not in a call" {
		t.Fatal(c.Value)
	}
	for _, id := range []string{"discord.video", "discord.screenshare", "discord.leave"} {
		if c := h.control(t, id); c.Available || c.Status != "Not in a call" {
			t.Fatal(id, c.Available, c.Status)
		}
	}
}

func TestPreviewRefusesWrites(t *testing.T) {
	h := newHarness(t, configured, false)
	ready(t, h)
	before := len(h.peer.received())
	h.press("discord.mute")
	if len(h.peer.received()) != before || h.control(t, "discord.mute").Status != "Preview" {
		t.Fatal("preview wrote to Discord")
	}
}

func TestReconnectDoesNotReplay(t *testing.T) {
	saved := configured
	saved.RefreshToken = "R1"
	h := newHarness(t, saved, true)
	ready(t, h)
	h.peer.frozen = true
	h.press("discord.mute")
	_ = h.peer.conn.Close()
	h.settle()
	if h.w.state != stateClosed || h.w.voice.settingsKnown || len(h.w.pending) != 0 {
		t.Fatal("state survived the drop", h.w.state, h.w.voice, h.w.pending)
	}
	h.peer.commands = nil
	h.now = h.now.Add(redial)
	h.peer.frozen = false
	ready(t, h)
	for _, cmd := range h.peer.received() {
		if cmd == "SET_VOICE_SETTINGS" {
			t.Fatal("replayed a write after reconnecting")
		}
	}
}

func TestClosedAndSetup(t *testing.T) {
	h := newHarness(t, Settings{}, true)
	h.w.step(h.ctx, h.now)
	if h.dials != 0 || h.control(t, "discord.mute").Status != "Setup needed" {
		t.Fatal("setup", h.dials, h.control(t, "discord.mute").Status)
	}
	h = newHarness(t, configured, true)
	h.w.dial = func() (io.ReadWriteCloser, string, error) { return nil, "", errNoDiscord }
	h.w.step(h.ctx, h.now)
	if c := h.control(t, "discord.mute"); c.Status != "Discord closed" || h.control(t, "discord.app-client").Connection.LastError != "" {
		t.Fatal(c.Status, h.control(t, "discord.app-client").Connection)
	}
}

func TestStopJoinsHelpers(t *testing.T) {
	services := snoofer.Services{Controls: snoofer.NewControls(), Live: true}
	w := newWorker(services, nil, configured, func() (io.ReadWriteCloser, string, error) { return nil, "", errNoDiscord }, http.DefaultClient, tokenURL)
	i := start(context.Background(), w)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := i.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if len(services.Controls.Snapshot()) != 0 {
		t.Fatal("controls left after stop")
	}
}
