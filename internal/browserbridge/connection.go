package browserbridge

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type hello struct {
	Type     string `json:"type"`
	Protocol int    `json:"protocol"`
	Version  string `json:"version"`
	Browser  string `json:"browser"`
	Meet     int    `json:"meet,omitempty"`
}

type refusal struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}
type capabilities struct {
	Type         string `json:"type"`
	Meet         int    `json:"meet"`
	MediaEnabled bool   `json:"mediaEnabled"`
	MeetEnabled  bool   `json:"meetEnabled"`
}

type connection struct {
	browser, version string
	generation       uint64
	meet             int
	ws               *websocket.Conn
	gate             chan struct{}
	caps             chan capabilities
	done             chan struct{}
	run              *listenerRun
}

var upgrader = websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: func(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return strings.HasPrefix(origin, "chrome-extension://") || strings.HasPrefix(origin, "moz-extension://")
}}

func (b *Bridge) connectionsLocked() []*connection {
	all := make([]*connection, 0, len(b.conns))
	for _, c := range b.conns {
		all = append(all, c)
	}
	return all
}

func (b *Bridge) capabilitiesLocked(c *connection) {
	if b.conns[c.browser] != c {
		return
	}
	v := capabilities{Type: "capabilities", Meet: 1, MediaEnabled: b.media != nil, MeetEnabled: b.meet != nil && c.meet == 1}
	select {
	case c.caps <- v:
	default:
		b.retireLocked(c, "browser feature updates overflowed")
	}
}

// retireLocked invalidates targets before best-effort disconnect delivery.
func (b *Bridge) retireLocked(c *connection, reason string) {
	if b.conns[c.browser] != c {
		return
	}
	delete(b.conns, c.browser)
	if reason != "" {
		b.status.Refused = reason
	}
	close(c.done)
	_ = c.ws.Close() // The exact retired connection cannot be used again.
	if b.media != nil {
		select {
		case b.media.updates <- MediaUpdate{Browser: c.browser, Generation: c.generation}:
		default:
		}
	}
	if b.meet != nil {
		select {
		case b.meet.updates <- MeetUpdate{Browser: c.browser, Generation: c.generation, Received: time.Now()}:
		default:
		}
	}
}

func (b *Bridge) mediaLocked(c *connection, sessions []MediaSession) {
	if b.media == nil || b.conns[c.browser] != c {
		return
	}
	select {
	case b.media.updates <- MediaUpdate{Browser: c.browser, Version: c.version, Connected: true, Generation: c.generation, Sessions: sessions}:
	default:
		b.retireLocked(c, "Media updates overflowed; browser reconnecting")
	}
}

func (b *Bridge) meetLocked(c *connection, calls []MeetCall) {
	if b.meet == nil || b.conns[c.browser] != c {
		return
	}
	select {
	case b.meet.updates <- MeetUpdate{Browser: c.browser, Version: c.version, Connected: true, Supported: c.meet == 1, Generation: c.generation, Received: time.Now(), Calls: calls}:
	default:
		b.retireLocked(c, "Meet updates overflowed; browser reconnecting")
	}
}

func (b *Bridge) serve(r *listenerRun, w http.ResponseWriter, req *http.Request) {
	b.mu.Lock()
	if b.closed || b.run != r || r.stopping {
		b.mu.Unlock()
		return
	}
	r.wg.Add(1)
	b.mu.Unlock()
	defer r.wg.Done()
	if !upgrader.CheckOrigin(req) {
		b.mu.Lock()
		b.status.Refused = "Refused a connection that was not from an extension"
		b.mu.Unlock()
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	b.mu.Lock()
	if r.stopping {
		b.mu.Unlock()
		return
	}
	r.sockets[ws] = true
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(r.sockets, ws)
		b.mu.Unlock()
	}()
	ws.SetReadLimit(maxMessage)
	if ws.SetReadDeadline(time.Now().Add(helloTimeout)) != nil {
		return
	}
	var h hello
	if err := ws.ReadJSON(&h); err != nil || h.Type != "hello" {
		return
	}
	browser := strings.TrimSpace(h.Browser)
	if browser == "" || len(browser) > 32 {
		return
	}
	if h.Protocol != protocol {
		reason := "Update the Snoofer Media extension in " + browser
		if h.Protocol > protocol {
			reason = "Update Snoofer: " + browser + "'s extension is newer"
		}
		b.refuseSocket(ws, reason, 4002, "protocol")
		return
	}
	c := &connection{browser: browser, version: h.Version, meet: h.Meet, ws: ws, gate: make(chan struct{}, 1), caps: make(chan capabilities, 16), done: make(chan struct{}), run: r}
	b.mu.Lock()
	if b.closed || r.stopping {
		b.mu.Unlock()
		return
	}
	if b.conns[browser] != nil {
		b.mu.Unlock()
		b.refuseSocket(ws, "Refused another connection claiming "+browser+"; its connection is already active", 4003, "connected")
		return
	}
	b.nextGeneration++
	c.generation = b.nextGeneration
	b.conns[browser] = c
	b.status.Refused = ""
	b.capabilitiesLocked(c)
	r.wg.Add(1)
	go b.writeCapabilities(c)
	b.mediaLocked(c, nil)
	b.meetLocked(c, nil)
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		b.retireLocked(c, "")
		b.mu.Unlock()
	}()
	for {
		if ws.SetReadDeadline(time.Now().Add(idleTimeout)) != nil {
			return
		}
		_, data, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var envelope struct {
			Type     string         `json:"type"`
			Sessions []MediaSession `json:"sessions"`
		}
		if json.Unmarshal(data, &envelope) != nil {
			continue
		}
		switch envelope.Type {
		case "sessions":
			b.mu.Lock()
			enabled := b.media != nil && b.conns[browser] == c
			b.mu.Unlock()
			if !enabled {
				continue
			}
			if err := ValidateMediaSessions(envelope.Sessions); err != nil {
				b.mu.Lock()
				b.retireLocked(c, err.Error())
				b.mu.Unlock()
				return
			}
			b.mu.Lock()
			b.mediaLocked(c, envelope.Sessions)
			b.mu.Unlock()
		case "meet-state":
			b.mu.Lock()
			enabled := b.meet != nil && c.meet == 1 && b.conns[browser] == c
			b.mu.Unlock()
			if !enabled {
				continue
			}
			calls, err := decodeCalls(data)
			if err != nil {
				b.mu.Lock()
				b.retireLocked(c, err.Error())
				b.mu.Unlock()
				return
			}
			b.mu.Lock()
			b.meetLocked(c, calls)
			b.mu.Unlock()
		case "meet-result":
			if result, err := decodeResult(data); err == nil {
				b.acceptResult(c, result)
			}
		}
	}
}

func (b *Bridge) refuseSocket(ws *websocket.Conn, reason string, code int, text string) {
	b.mu.Lock()
	b.status.Refused = reason
	b.mu.Unlock()
	if ws.SetWriteDeadline(time.Now().Add(writeTimeout)) == nil {
		_ = ws.WriteJSON(refusal{Type: "refused", Reason: reason}) // Best effort before close.
	}
	_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(writeTimeout))
}

func (b *Bridge) writeCapabilities(c *connection) {
	defer c.run.wg.Done()
	for {
		select {
		case <-c.done:
			return
		case <-c.run.ctx.Done():
			return
		case update := <-c.caps:
			_, err := b.write(c.run.ctx, c, update, func() bool { return b.conns[c.browser] == c })
			if err != nil {
				b.mu.Lock()
				b.retireLocked(c, "")
				b.mu.Unlock()
				return
			}
		}
	}
}

// write's budget includes gate waiting. check runs under b.mu just before I/O;
// its bool result reports whether a write may have started.
func (b *Bridge) write(ctx context.Context, c *connection, value any, check func() bool) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return false, ctx.Err()
	case <-c.done:
		return false, ErrConnectionChanged
	}
	defer func() { <-c.gate }()
	b.mu.Lock()
	valid := ctx.Err() == nil && !b.closed && !c.run.stopping && c.run.ctx.Err() == nil && b.conns[c.browser] == c && check()
	b.mu.Unlock()
	if !valid {
		return false, ErrConnectionChanged
	}
	deadline, _ := ctx.Deadline()
	if err := c.ws.SetWriteDeadline(deadline); err != nil {
		return false, err
	}
	return true, c.ws.WriteJSON(value)
}

func (l *MediaLease) Send(browser string, command MediaCommand) error {
	b := l.b
	b.mu.Lock()
	c := b.conns[browser]
	valid := l.active && l.ctx.Err() == nil && !b.closed && c != nil && !c.run.stopping && c.run.ctx.Err() == nil && command.Generation != 0 && command.Generation == c.generation
	b.mu.Unlock()
	if !valid {
		return ErrConnectionChanged
	}
	started, err := b.write(l.ctx, c, command, func() bool { return l.active && l.ctx.Err() == nil })
	if !started && err != nil {
		if errors.Is(err, ErrConnectionChanged) || l.ctx.Err() != nil {
			return ErrConnectionChanged
		}
		// Gate/deadline failure is known unsent; it must not reserve a rating
		// as though a socket write might already have changed the page.
		return ErrBusy
	}
	return err
}
