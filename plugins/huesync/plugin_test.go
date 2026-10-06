package huesync

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"sound-snoofer/snoofer"
)

var testTiming = timing{
	dial:          time.Second,
	write:         time.Second,
	retryBase:     30 * time.Millisecond,
	retryMax:      100 * time.Millisecond,
	brightnessGap: 60 * time.Millisecond,
	confirm:       300 * time.Millisecond,
}

// fakeApp emulates the Hue Sync third-party control socket.
type fakeApp struct {
	*httptest.Server
	t *testing.T

	mu       sync.Mutex
	state    map[string]any
	commands []string
	ignore   bool
	clients  []*websocket.Conn
}

func newFakeApp(t *testing.T) *fakeApp {
	t.Helper()
	app := &fakeApp{t: t, state: map[string]any{"state": stateConnected, "mode": "video", "intensity": "moderate", "bri": 50.0}}
	upgrader := websocket.Upgrader{}
	app.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		app.mu.Lock()
		app.clients = append(app.clients, conn)
		app.push(conn)
		app.mu.Unlock()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			app.command(conn, data)
		}
	}))
	t.Cleanup(func() {
		app.drop()
		app.Server.Close()
	})
	return app
}

func (a *fakeApp) port() int {
	port, _ := strconv.Atoi(a.URL[strings.LastIndex(a.URL, ":")+1:])
	return port
}

// push sends the current state; callers hold mu.
func (a *fakeApp) push(conn *websocket.Conn) {
	if err := conn.WriteJSON(map[string]any{"event": "app_state_update", "data": a.state}); err != nil {
		a.t.Log(err)
	}
}

func (a *fakeApp) command(conn *websocket.Conn, data []byte) {
	var message struct {
		Command string         `json:"command"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(data, &message); err != nil {
		a.t.Error(err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.commands = append(a.commands, string(data))
	if a.ignore {
		return
	}
	syncing := a.state["state"] == stateSyncing
	switch message.Command {
	case "start_sync":
		a.state["state"] = stateSyncing
	case "stop_sync":
		a.state["state"] = stateConnected
	case "inc_bri":
		bri := a.state["bri"].(float64) + message.Data["step"].(float64)
		a.state["bri"] = min(100.0, max(0.0, bri))
	case "set_app_mode":
		if syncing {
			a.state["mode"] = message.Data["mode"]
		}
	case "set_intensity":
		if syncing {
			a.state["intensity"] = message.Data["intensity"]
		}
	}
	a.push(conn)
}

func (a *fakeApp) set(change func(*fakeApp)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	change(a)
	for _, c := range a.clients {
		a.push(c)
	}
}

func (a *fakeApp) drop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, c := range a.clients {
		c.Close()
	}
	a.clients = nil
}

func (a *fakeApp) log() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.commands...)
}

type harness struct {
	t        *testing.T
	controls *snoofer.Controls
}

func startHarness(t *testing.T, port int, live bool) *harness {
	t.Helper()
	h := &harness{t: t, controls: snoofer.NewControls()}
	raw, _ := json.Marshal(Settings{Port: port})
	instance, err := start(context.Background(), snoofer.Services{Controls: h.controls, Live: live}, raw, testTiming)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := instance.Stop(ctx); err != nil {
			t.Errorf("stop: %v", err)
		}
		if len(h.controls.Snapshot()) != 0 {
			t.Error("controls remain after stop")
		}
	})
	return h
}

func (h *harness) find(id string) snoofer.Control {
	for _, c := range h.controls.Snapshot() {
		if c.ID == id {
			return c
		}
	}
	return snoofer.Control{}
}

func (h *harness) wait(id string, ok func(snoofer.Control) bool) snoofer.Control {
	h.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c := h.find(id); c.ID != "" && ok(c) {
			return c
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.t.Fatalf("%s did not reach the expected state; last %+v", id, h.find(id))
	return snoofer.Control{}
}

func (h *harness) value(id, want string) snoofer.Control {
	h.t.Helper()
	return h.wait(id, func(c snoofer.Control) bool { return c.Value == want })
}

func (h *harness) dispatch(id, operation string, delta int, value string) error {
	c := h.find(id)
	return h.controls.Dispatch(context.Background(), snoofer.Request{ID: id, Revision: c.Revision, Operation: operation, Delta: delta, Value: value})
}

func (h *harness) mustDispatch(id, operation string, delta int, value string) {
	h.t.Helper()
	if err := h.dispatch(id, operation, delta, value); err != nil {
		h.t.Fatalf("%s %s: %v", id, operation, err)
	}
}

func TestValidateSettings(t *testing.T) {
	if err := validate(Plugin().Defaults); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{"port":0}`, `{"port":70000}`, `{}`, `{"port":1,"host":"x"}`} {
		if validate(json.RawMessage(raw)) == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestObservedStateAndSyncToggle(t *testing.T) {
	app := newFakeApp(t)
	h := startHarness(t, app.port(), true)
	h.value("huesync.status", "Ready")
	h.value("huesync.brightness", "50%")
	if mode := h.value("huesync.mode", "video"); mode.Available {
		t.Fatal("mode adjustable while not syncing")
	}
	h.mustDispatch("huesync.sync", "press", 0, "")
	h.value("huesync.sync", "On")
	h.wait("huesync.sync", func(c snoofer.Control) bool { return c.Status == "" })
	h.wait("huesync.mode", func(c snoofer.Control) bool { return c.Available })
	h.mustDispatch("huesync.mode", "set", 0, "games")
	h.value("huesync.mode", "games")
	h.mustDispatch("huesync.intensity", "set", 0, "extreme")
	h.value("huesync.intensity", "extreme")
	h.mustDispatch("huesync.brightness", "press", 0, "")
	h.value("huesync.sync", "Off")
	want := []string{
		`{"command":"start_sync","data":{}}`,
		`{"command":"set_app_mode","data":{"mode":"games"}}`,
		`{"command":"set_intensity","data":{"intensity":"extreme"}}`,
		`{"command":"stop_sync","data":{}}`,
	}
	if got := app.log(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("commands\n%s", strings.Join(got, "\n"))
	}
	if err := h.dispatch("huesync.mode", "set", 0, "sparkle"); err == nil {
		t.Fatal("invalid mode accepted")
	}
}

func TestBrightnessTicksAccumulate(t *testing.T) {
	app := newFakeApp(t)
	h := startHarness(t, app.port(), true)
	h.value("huesync.brightness", "50%")
	for range 5 {
		h.mustDispatch("huesync.brightness", "adjust", 1, "")
	}
	h.value("huesync.brightness", "60%")
	commands := app.log()
	if len(commands) == 0 || len(commands) > 3 {
		t.Fatalf("%d commands for 5 ticks: %v", len(commands), commands)
	}
	if !strings.HasPrefix(commands[0], `{"command":"inc_bri","data":{"step":`) {
		t.Fatalf("commands %v", commands)
	}
}

func TestSyncUnconfirmed(t *testing.T) {
	app := newFakeApp(t)
	h := startHarness(t, app.port(), true)
	h.value("huesync.sync", "Off")
	app.set(func(a *fakeApp) { a.ignore = true })
	h.mustDispatch("huesync.sync", "press", 0, "")
	h.wait("huesync.sync", func(c snoofer.Control) bool { return c.Status == "Pending" })
	c := h.wait("huesync.sync", func(c snoofer.Control) bool { return c.Status == "Not confirmed by Hue Sync" })
	if c.Value != "Off" || len(app.log()) != 1 {
		t.Fatalf("sync %+v, commands %v", c, app.log())
	}
}

func TestBridgeDisconnectedDisablesControls(t *testing.T) {
	app := newFakeApp(t)
	h := startHarness(t, app.port(), true)
	h.value("huesync.status", "Ready")
	app.set(func(a *fakeApp) { a.state["state"] = stateDisconnected })
	h.value("huesync.status", "No bridge")
	for _, id := range []string{"huesync.sync", "huesync.brightness", "huesync.mode", "huesync.intensity"} {
		if h.find(id).Available {
			t.Errorf("%s available without bridge", id)
		}
	}
}

func TestRestartReconnectsWithoutReplay(t *testing.T) {
	app := newFakeApp(t)
	h := startHarness(t, app.port(), true)
	h.value("huesync.sync", "Off")
	app.set(func(a *fakeApp) { a.ignore = true })
	h.mustDispatch("huesync.sync", "press", 0, "")
	h.wait("huesync.sync", func(c snoofer.Control) bool { return c.Status == "Pending" })
	app.set(func(a *fakeApp) { a.ignore = false })
	app.drop()
	h.value("huesync.status", "N/A")
	h.value("huesync.status", "Ready")
	c := h.value("huesync.sync", "Off")
	if c.Status != "" || len(app.log()) != 1 {
		t.Fatalf("sync %+v, commands %v", c, app.log())
	}
}

func TestRefusedConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	h := startHarness(t, port, true)
	c := h.wait("huesync.status", func(c snoofer.Control) bool { return strings.Contains(c.Status, "Third-party control") })
	if c.Value != "N/A" || h.find("huesync.sync").Available {
		t.Fatalf("status %+v", c)
	}
}

func TestPreviewSendsNothing(t *testing.T) {
	app := newFakeApp(t)
	h := startHarness(t, app.port(), false)
	h.value("huesync.status", "Ready")
	for _, id := range []string{"huesync.sync", "huesync.brightness"} {
		if err := h.dispatch(id, "press", 0, ""); err == nil {
			t.Errorf("preview accepted %s", id)
		}
	}
	time.Sleep(3 * testTiming.brightnessGap)
	if commands := app.log(); len(commands) != 0 {
		t.Fatalf("preview sent %v", commands)
	}
}
