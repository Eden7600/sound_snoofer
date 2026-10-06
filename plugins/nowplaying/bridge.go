package nowplaying

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxMessage   = 1 << 20 // Bytes per message from an extension.
	helloTimeout = 5 * time.Second
	idleTimeout  = 60 * time.Second // The extension pings every 20 s.
	writeTimeout = 2 * time.Second
)

// newToken returns 32 random bytes in hex.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hello is the extension's first message.
type hello struct {
	Type    string `json:"type"`
	Token   string `json:"token"`
	Version string `json:"version"`
	Browser string `json:"browser"`
}

type report struct {
	Type     string           `json:"type"` // "sessions" or "ping".
	Sessions []browserSession `json:"sessions"`
}

// bridgeConn is one connected browser. Writes come from the worker and are
// serialized here.
type bridgeConn struct {
	browser string
	ws      *websocket.Conn
	mu      sync.Mutex
}

func (c *bridgeConn) write(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ws.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return c.ws.WriteJSON(v)
}

// bridge accepts extension connections on localhost. It owns its listener
// and connection goroutines; close stops them and waits.
type bridge struct {
	updates chan<- browserUpdate
	ctx     context.Context

	mu      sync.Mutex
	closed  bool // Set before waiting, so no handler joins the wait group late.
	token   string
	conns   map[string]*bridgeConn
	refused string // Why the last connection was refused, for the GUI.

	server *http.Server
	wg     sync.WaitGroup
}

// listen starts the bridge on 127.0.0.1:port. Updates are delivered until ctx
// ends; the caller then calls close.
func listen(ctx context.Context, port int, token string, updates chan<- browserUpdate) (*bridge, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("browser bridge port %d: %w", port, err)
	}
	b := &bridge{updates: updates, ctx: ctx, token: token, conns: map[string]*bridgeConn{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/nowplaying", b.serve)
	b.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		_ = b.server.Serve(listener) // Returns ErrServerClosed on close.
	}()

	return b, nil
}

// close stops accepting, drops every connection and waits for their
// goroutines.
func (b *bridge) close() {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = b.server.Shutdown(ctx) // Hijacked WebSockets are closed below.
	b.dropAll()
	b.wg.Wait()
}

// dropAll closes every connection; each reader then removes itself and
// reports the disconnect.
func (b *bridge) dropAll() {
	b.mu.Lock()
	conns := make([]*bridgeConn, 0, len(b.conns))
	for _, c := range b.conns {
		conns = append(conns, c)
	}
	b.mu.Unlock()
	for _, c := range conns {
		_ = c.ws.Close() // Errors only mean it is already closed.
	}
}

// setToken replaces the token and drops connections made with the old one.
func (b *bridge) setToken(token string) {
	b.mu.Lock()
	b.token = token
	b.mu.Unlock()
	b.dropAll()
}

func (b *bridge) lastRefusal() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.refused
}

func (b *bridge) refuse(reason string) {
	b.mu.Lock()
	b.refused = reason
	b.mu.Unlock()
}

// send delivers a command to a connected browser.
func (b *bridge) send(browser string, c browserCommand) error {
	b.mu.Lock()
	conn := b.conns[browser]
	b.mu.Unlock()
	if conn == nil {
		return errors.New(browser + " is not connected")
	}
	return conn.write(c)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize: 4096, WriteBufferSize: 4096,
	// Only extensions: web pages cannot present a chrome-extension origin.
	CheckOrigin: func(r *http.Request) bool { return strings.HasPrefix(r.Header.Get("Origin"), "chrome-extension://") },
}

func (b *bridge) serve(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.wg.Add(1)
	b.mu.Unlock()
	defer b.wg.Done()
	if !upgrader.CheckOrigin(r) {
		b.refuse("Refused a connection that was not from an extension")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade has already replied.
	}
	defer ws.Close()
	ws.SetReadLimit(maxMessage)
	if err := ws.SetReadDeadline(time.Now().Add(helloTimeout)); err != nil {
		return
	}
	var h hello
	if err := ws.ReadJSON(&h); err != nil || h.Type != "hello" {
		return
	}
	b.mu.Lock()
	token := b.token
	b.mu.Unlock()
	browser := strings.TrimSpace(h.Browser)
	if subtle.ConstantTimeCompare([]byte(h.Token), []byte(token)) != 1 {
		b.refuse("Refused " + browser + ": the extension's token is out of date. Save the extension files and reload it.")
		_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "token"), time.Now().Add(writeTimeout))
		return
	}
	if browser == "" || len(browser) > 32 {
		return
	}
	conn := &bridgeConn{browser: browser, ws: ws}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	previous := b.conns[browser]
	b.conns[browser] = conn
	b.refused = ""
	b.mu.Unlock()
	if previous != nil {
		_ = previous.ws.Close() // A newer connection from the same browser wins.
	}
	b.deliver(browserUpdate{Browser: browser, Version: h.Version, Connected: true})
	defer func() {
		b.mu.Lock()
		current := b.conns[browser] == conn
		if current {
			delete(b.conns, browser)
		}
		b.mu.Unlock()
		if current {
			b.deliver(browserUpdate{Browser: browser})
		}
	}()
	for {
		if err := ws.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			return
		}
		_, data, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var m report
		if err := json.Unmarshal(data, &m); err != nil {
			return
		}
		if m.Type != "sessions" {
			continue // Pings only keep the connection and worker alive.
		}
		if err := validBrowserSessions(m.Sessions); err != nil {
			b.refuse(err.Error())
			return
		}
		b.deliver(browserUpdate{Browser: browser, Version: h.Version, Connected: true, Sessions: m.Sessions})
	}
}

// deliver blocks until the worker takes the update or the plugin stops, so
// the latest state is never dropped.
func (b *bridge) deliver(u browserUpdate) {
	select {
	case b.updates <- u:
	case <-b.ctx.Done():
	}
}
