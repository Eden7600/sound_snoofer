package hue

import (
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

// fakeSyncApp emulates the Hue Sync third-party control socket.
type fakeSyncApp struct {
	*httptest.Server
	t *testing.T

	mu       sync.Mutex
	state    map[string]any
	commands []string
	ignore   bool
	clients  []*websocket.Conn
}

func newFakeSyncApp(t *testing.T, state string) *fakeSyncApp {
	t.Helper()
	app := &fakeSyncApp{t: t, state: map[string]any{"state": state, "mode": "video", "intensity": "moderate", "bri": 50.0}}
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

func (a *fakeSyncApp) port() *int {
	port, _ := strconv.Atoi(a.URL[strings.LastIndex(a.URL, ":")+1:])
	return &port
}

// push sends the current state; callers hold mu.
func (a *fakeSyncApp) push(conn *websocket.Conn) {
	if err := conn.WriteJSON(map[string]any{"event": "app_state_update", "data": a.state}); err != nil {
		a.t.Log(err)
	}
}

func (a *fakeSyncApp) command(conn *websocket.Conn, data []byte) {
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
	syncing := a.state["state"] == syncStateSyncing
	switch message.Command {
	case "start_sync":
		a.state["state"] = syncStateSyncing
	case "stop_sync":
		a.state["state"] = "bridge_connected"
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

func (a *fakeSyncApp) set(change func(*fakeSyncApp)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	change(a)
	for _, c := range a.clients {
		a.push(c)
	}
}

func (a *fakeSyncApp) drop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, c := range a.clients {
		c.Close()
	}
	a.clients = nil
}

func (a *fakeSyncApp) log() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.commands...)
}

func closedPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

// startJoined starts the plugin against a fake bridge and a fake Hue Sync app.
func startJoined(t *testing.T, syncState string, live bool) (*harness, *fakeBridge, *fakeSyncApp) {
	t.Helper()
	bridge := newFakeBridge(t, "b1", studio()...)
	app := newFakeSyncApp(t, syncState)
	settings := paired(bridge, "room-1")
	settings.SyncPort = app.port()
	return startHarness(t, bridge, settings, live), bridge, app
}

func TestSyncPortValidation(t *testing.T) {
	for _, raw := range []string{`{"sync_port":0}`, `{"sync_port":70000}`} {
		if validate(json.RawMessage(raw)) == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if err := validate(json.RawMessage(`{"sync_port":24851}`)); err != nil {
		t.Fatal(err)
	}
}

func TestSyncObservedStateAndCommands(t *testing.T) {
	h, _, app := startJoined(t, "bridge_connected", true)
	h.value("hue.sync-status", "Ready")
	if mode := h.value("hue.sync-mode", "video"); mode.Available || !mode.Hidden {
		t.Fatal("mode shown or adjustable while not syncing")
	}
	if sync, _ := h.find("hue.sync"); sync.Hidden {
		t.Fatal("sync hidden while the app is connected")
	}
	h.mustDispatch("hue.sync", "press", 0, "")
	h.value("hue.sync", "On")
	h.waitControl("hue.sync-mode", func(c snoofer.Control) bool { return c.Available && !c.Hidden })
	if intensity, _ := h.find("hue.sync-intensity"); intensity.Hidden {
		t.Fatal("intensity hidden while syncing")
	}
	h.mustDispatch("hue.sync-mode", "set", 0, "games")
	h.value("hue.sync-mode", "games")
	h.mustDispatch("hue.sync-intensity", "set", 0, "extreme")
	h.value("hue.sync-intensity", "extreme")
	h.waitControl("hue.sync", func(c snoofer.Control) bool { return c.Status == "" })
	h.mustDispatch("hue.sync", "press", 0, "")
	h.value("hue.sync", "Off")
	want := []string{
		`{"command":"start_sync","data":{}}`,
		`{"command":"set_app_mode","data":{"mode":"games"}}`,
		`{"command":"set_intensity","data":{"intensity":"extreme"}}`,
		`{"command":"stop_sync","data":{}}`,
	}
	if got := app.log(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("commands\n%s", strings.Join(got, "\n"))
	}
	if err := h.dispatch("hue.sync-mode", "set", 0, "sparkle"); err == nil {
		t.Fatal("invalid mode accepted")
	}
}

func TestBrightnessDialFollowsSync(t *testing.T) {
	h, bridge, app := startJoined(t, syncStateSyncing, true)
	h.value("hue.brightness", "Sync 50%")
	for range 5 {
		h.mustDispatch("hue.brightness", "adjust", 1, "")
	}
	h.value("hue.brightness", "Sync 60%")
	commands := app.log()
	if len(commands) == 0 || len(commands) > 3 || !strings.HasPrefix(commands[0], `{"command":"inc_bri"`) {
		t.Fatalf("commands %v", commands)
	}
	if puts := bridge.putLog(); len(puts) != 0 {
		t.Fatalf("room written while syncing: %v", puts)
	}
	h.mustDispatch("hue.brightness", "press", 0, "")
	waitFor(t, func() bool { return len(bridge.putLog()) == 1 })
	if puts := bridge.putLog(); puts[0] != `grouped_light/gl-1:{"on":{"on":false}}` {
		t.Fatalf("press while syncing wrote %v", puts)
	}
	h.mustDispatch("hue.sync", "press", 0, "")
	h.value("hue.brightness", "Off")
}

func TestSceneStopsSyncFirst(t *testing.T) {
	h, bridge, app := startJoined(t, syncStateSyncing, true)
	h.value("hue.sync", "On")
	h.value("hue.scene-studio-3f2a9c10", "Ready")
	h.mustDispatch("hue.scene-studio-3f2a9c10", "press", 0, "")
	h.value("hue.scene-studio-3f2a9c10", "Active")
	h.value("hue.sync", "Off")
	if got := app.log(); len(got) != 1 || got[0] != `{"command":"stop_sync","data":{}}` {
		t.Fatalf("commands %v", got)
	}
	if puts := bridge.putLog(); len(puts) != 1 || !strings.HasPrefix(puts[0], "scene/3f2a9c10") {
		t.Fatalf("writes %v", puts)
	}
}

func TestSceneNotRecalledWhenSyncDoesNotStop(t *testing.T) {
	h, bridge, app := startJoined(t, syncStateSyncing, true)
	h.value("hue.sync", "On")
	app.set(func(a *fakeSyncApp) { a.ignore = true })
	h.mustDispatch("hue.scene-studio-3f2a9c10", "press", 0, "")
	h.waitControl("hue.scene-studio-3f2a9c10", func(c snoofer.Control) bool { return c.Status == "Pending" })
	h.waitControl("hue.scene-studio-3f2a9c10", func(c snoofer.Control) bool { return c.Status == errSceneSyncNotEnded })
	if puts := bridge.putLog(); len(puts) != 0 {
		t.Fatalf("scene recalled while syncing: %v", puts)
	}
}

func TestSyncUnconfirmed(t *testing.T) {
	h, _, app := startJoined(t, "bridge_connected", true)
	h.value("hue.sync", "Off")
	app.set(func(a *fakeSyncApp) { a.ignore = true })
	h.mustDispatch("hue.sync", "press", 0, "")
	h.waitControl("hue.sync", func(c snoofer.Control) bool { return c.Status == "Pending" })
	c := h.waitControl("hue.sync", func(c snoofer.Control) bool { return c.Status == errSyncNotConfirmed })
	if c.Value != "Off" || len(app.log()) != 1 {
		t.Fatalf("sync %+v, commands %v", c, app.log())
	}
}

func TestSyncBridgeDisconnected(t *testing.T) {
	h, _, app := startJoined(t, "bridge_connected", true)
	h.value("hue.sync-status", "Ready")
	app.set(func(a *fakeSyncApp) { a.state["state"] = syncStateDisconnected })
	h.value("hue.sync-status", "No bridge")
	for _, id := range []string{"hue.sync", "hue.sync-mode", "hue.sync-intensity"} {
		if c, _ := h.find(id); c.Available {
			t.Errorf("%s available without bridge", id)
		}
	}
	if c, _ := h.find("hue.brightness"); !c.Available {
		t.Error("room brightness lost with Hue Sync bridge")
	}
}

func TestSyncRestartWithoutReplay(t *testing.T) {
	h, _, app := startJoined(t, "bridge_connected", true)
	h.value("hue.sync", "Off")
	app.set(func(a *fakeSyncApp) { a.ignore = true })
	h.mustDispatch("hue.sync", "press", 0, "")
	h.waitControl("hue.sync", func(c snoofer.Control) bool { return c.Status == "Pending" })
	app.set(func(a *fakeSyncApp) { a.ignore = false })
	app.drop()
	h.value("hue.sync-status", "N/A")
	h.value("hue.sync-status", "Ready")
	c := h.value("hue.sync", "Off")
	if c.Status != "" || len(app.log()) != 1 {
		t.Fatalf("sync %+v, commands %v", c, app.log())
	}
}

func TestSyncRefusedKeepsRoomWorking(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)
	c := h.waitControl("hue.sync-status", func(c snoofer.Control) bool { return strings.Contains(c.Status, "Third-party control") })
	if c.Value != "N/A" {
		t.Fatalf("sync status %+v", c)
	}
	if s, _ := h.find("hue.sync"); s.Available || !s.Hidden {
		t.Fatal("sync shown without app")
	}
	h.value("hue.brightness", "50%")
}

func TestSyncPreviewSendsNothing(t *testing.T) {
	h, _, app := startJoined(t, "bridge_connected", false)
	h.value("hue.sync-status", "Ready")
	if err := h.dispatch("hue.sync", "press", 0, ""); err == nil {
		t.Error("preview accepted sync")
	}
	time.Sleep(3 * testTiming.syncBrightnessGap)
	if commands := app.log(); len(commands) != 0 {
		t.Fatalf("preview sent %v", commands)
	}
}
