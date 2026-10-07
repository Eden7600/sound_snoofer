// Package discord controls the Discord desktop client's voice state through
// its local RPC pipe, using the user's own Discord application.
package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"sound-snoofer/snoofer"
)

// Settings are the plugin's saved settings. The client secret and refresh
// token are credentials: they never appear in controls or reports.
type Settings struct {
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (s Settings) redirect() string {
	if s.RedirectURI == "" {
		return "http://127.0.0.1"
	}
	return s.RedirectURI
}

func (s Settings) configured() bool {
	return s.ClientID != "" && s.ClientSecret != ""
}

func validate(raw json.RawMessage) error {
	var s Settings
	if err := snoofer.DecodeSettings(raw, &s); err != nil {
		return err
	}
	if s.ClientID != "" {
		if _, err := strconv.ParseUint(s.ClientID, 10, 64); err != nil {
			return fmt.Errorf("client_id must be the application's numeric ID")
		}
	}
	return nil
}

// Plugin returns inert metadata; the pipe is opened only by Start's worker.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "discord", Label: "Discord", Validate: validate, Defaults: snoofer.MarshalSettings(Settings{}),
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
			var settings Settings
			if err := snoofer.DecodeSettings(raw, &settings); err != nil {
				return nil, err
			}
			w := newWorker(s, raw, settings, dialPipe, http.DefaultClient, tokenURL)
			return start(ctx, w), nil
		}}
}

const (
	tick             = 500 * time.Millisecond
	redial           = 5 * time.Second
	requestTimeout   = 5 * time.Second
	authorizeTimeout = 2 * time.Minute // The user answers Discord's popup.
	verifyDeadline   = 3 * time.Second
	maxBackoff       = 60 * time.Second
)

// Connection states.
const (
	stateSetup          = "setup"          // No client ID or secret.
	stateClosed         = "closed"         // No pipe.
	stateHandshake      = "handshake"      // Waiting for READY.
	stateUnauthorized   = "unauthorized"   // Waiting for the user's Connect.
	stateAuthorizing    = "authorizing"    // Discord's approval popup is open.
	stateExchanging     = "exchanging"     // Token request in flight.
	stateRetrying       = "retrying"       // Token request failed; retrying later.
	stateAuthenticating = "authenticating" // AUTHENTICATE in flight.
	stateReady          = "ready"
)

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Stop cancels the worker and waits for it and its helper goroutines.
func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func start(ctx context.Context, w *worker) *instance {
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(i.done)
		defer w.services.Controls.Remove("discord")
		w.run(runCtx)
	}()
	return i
}

// call is a request awaiting its response.
type call struct {
	purpose  string // What the response completes.
	id       string // Control that issued it, if any.
	deadline time.Time
}

// pendingWrite is a requested state awaiting Discord's update.
type pendingWrite struct {
	want     string
	deadline time.Time
}

// incoming is a frame, or the read error that ended a connection.
type incoming struct {
	generation int
	f          frame
	err        error
}

type tokenResult struct {
	generation int
	t          tokens
	err        error
}

// voice is Discord's state as last reported.
type voice struct {
	settingsKnown bool
	mute, deaf    bool
	channelKnown  bool
	channelID     string
	channelName   string
	videoKnown    bool
	video         bool
	screenKnown   bool
	screen        bool
}

type worker struct {
	services snoofer.Services
	raw      json.RawMessage
	settings Settings
	dial     func() (io.ReadWriteCloser, string, error)
	client   *http.Client
	endpoint string
	requests chan snoofer.Request
	frames   chan incoming
	tokens   chan tokenResult
	helpers  sync.WaitGroup // Reader and token goroutines.

	conn        io.ReadWriteCloser
	pipeName    string
	generation  int
	state       string
	deadline    time.Time // Handshake deadline.
	nextDial    time.Time
	nextRefresh time.Time
	backoff     time.Duration
	nonce       int
	calls       map[string]call
	access      string // In memory only.
	voice       voice
	pending     map[string]pendingWrite
	notes       map[string]string
	link        snoofer.ConnectionTracker
}

func newWorker(s snoofer.Services, raw json.RawMessage, settings Settings, dial func() (io.ReadWriteCloser, string, error), client *http.Client, endpoint string) *worker {
	w := &worker{services: s, raw: append(json.RawMessage(nil), raw...), settings: settings, dial: dial, client: client, endpoint: endpoint,
		requests: make(chan snoofer.Request, 8), frames: make(chan incoming), tokens: make(chan tokenResult),
		calls: map[string]call{}, pending: map[string]pendingWrite{}, notes: map[string]string{}, state: stateClosed}
	if !settings.configured() {
		w.state = stateSetup
	}
	return w
}

func (w *worker) run(ctx context.Context) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	defer w.helpers.Wait()
	defer w.closeConn()
	for {
		w.step(ctx, time.Now())
		w.publish(time.Now())
		select {
		case <-ctx.Done():
			return
		case r := <-w.requests:
			w.handle(ctx, r, time.Now())
		case in := <-w.frames:
			w.receive(ctx, in, time.Now())
		case t := <-w.tokens:
			w.tokenDone(ctx, t, time.Now())
		case <-ticker.C:
		}
	}
}

// step connects, retries token refreshes and expires requests and writes.
func (w *worker) step(ctx context.Context, now time.Time) {
	if w.state == stateSetup {
		return
	}
	if w.conn == nil && !now.Before(w.nextDial) {
		w.connect(ctx, now)
	}
	if w.state == stateHandshake && now.After(w.deadline) {
		w.drop(errors.New("Discord handshake timed out"), now)
	}
	if w.state == stateRetrying && !now.Before(w.nextRefresh) {
		w.startExchange(ctx, "")
	}
	for nonce, c := range w.calls {
		if now.After(c.deadline) {
			delete(w.calls, nonce)
			w.callFailed(c, fmt.Errorf("Discord did not answer %s", c.purpose), now)
		}
	}
	for id, p := range w.pending {
		switch {
		case w.observed(id) == p.want:
			delete(w.pending, id)
		case now.After(p.deadline):
			delete(w.pending, id)
			w.notes[id] = "Failed"
		}
	}
}

func (w *worker) connect(ctx context.Context, now time.Time) {
	conn, name, err := w.dial()
	if err != nil {
		w.nextDial = now.Add(redial)
		w.state = stateClosed
		if !errors.Is(err, errNoDiscord) {
			w.link.Fail(err.Error(), now)
		}
		return
	}
	w.conn, w.pipeName = conn, name
	w.generation++
	generation := w.generation
	w.helpers.Add(1)
	go func() {
		defer w.helpers.Done()
		for {
			f, err := readFrame(conn)
			select {
			case w.frames <- incoming{generation: generation, f: f, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	w.state = stateHandshake
	w.deadline = now.Add(requestTimeout)
	if err := writeFrame(conn, opHandshake, map[string]any{"v": 1, "client_id": w.settings.ClientID}); err != nil {
		w.drop(err, now)
	}
}

// drop closes the connection and forgets everything Discord reported.
func (w *worker) drop(err error, now time.Time) {
	w.closeConn()
	w.state = stateClosed
	w.nextDial = now.Add(redial)
	w.calls = map[string]call{}
	w.pending = map[string]pendingWrite{}
	w.voice = voice{}
	w.access = ""
	if err != nil && !errors.Is(err, io.EOF) {
		w.link.Fail(err.Error(), now)
	}
}

func (w *worker) closeConn() {
	if w.conn == nil {
		return
	}
	// The reader goroutine sees the close as a read error and exits.
	_ = w.conn.Close() // A failed close leaves nothing further to release.
	w.conn = nil
}

// send issues a command; its response completes purpose.
func (w *worker) send(cmd string, args any, purpose, id string, timeout time.Duration, now time.Time) {
	w.sendEvent(cmd, "", args, purpose, id, timeout, now)
}

func (w *worker) sendEvent(cmd, evt string, args any, purpose, id string, timeout time.Duration, now time.Time) {
	if w.conn == nil {
		return
	}
	w.nonce++
	nonce := strconv.Itoa(w.nonce)
	w.calls[nonce] = call{purpose: purpose, id: id, deadline: now.Add(timeout)}
	if err := writeFrame(w.conn, opFrame, message{Cmd: cmd, Evt: evt, Nonce: nonce, Args: args}); err != nil {
		w.drop(err, now)
	}
}

func (w *worker) receive(ctx context.Context, in incoming, now time.Time) {
	if in.generation != w.generation || w.conn == nil {
		return
	}
	if in.err != nil {
		w.drop(in.err, now)
		return
	}
	switch in.f.op {
	case opPing:
		if err := writeFrame(w.conn, opPong, json.RawMessage(in.f.payload)); err != nil {
			w.drop(err, now)
		}
	case opClose:
		var reason rpcError
		_ = json.Unmarshal(in.f.payload, &reason) // A malformed reason still closes.
		w.drop(fmt.Errorf("Discord closed the connection: %s", reason.Message), now)
	case opFrame:
		var m message
		if err := json.Unmarshal(in.f.payload, &m); err != nil {
			w.link.Fail("malformed Discord message", now)
			return
		}
		w.dispatch(ctx, m, now)
	}
}

func (w *worker) dispatch(ctx context.Context, m message, now time.Time) {
	if m.Cmd == "DISPATCH" {
		w.event(ctx, m.Evt, m.Data, now)
		return
	}
	c, ok := w.calls[m.Nonce]
	if !ok {
		return
	}
	delete(w.calls, m.Nonce)
	if m.Evt == "ERROR" {
		var e rpcError
		_ = json.Unmarshal(m.Data, &e) // An unparsed error still fails the call.
		w.callFailed(c, e, now)
		return
	}
	w.link.Activity(now)
	w.callDone(ctx, c, m.Data, now)
}

func (w *worker) event(ctx context.Context, evt string, data json.RawMessage, now time.Time) {
	w.link.Activity(now)
	switch evt {
	case "READY":
		if w.state != stateHandshake {
			return
		}
		if w.settings.RefreshToken != "" {
			w.startExchange(ctx, "")
		} else {
			w.state = stateUnauthorized
		}
	case "VOICE_SETTINGS_UPDATE":
		w.voiceSettings(data)
	case "VOICE_CHANNEL_SELECT":
		var selected struct {
			ChannelID *string `json:"channel_id"`
		}
		if json.Unmarshal(data, &selected) != nil {
			return
		}
		if selected.ChannelID == nil {
			w.leftChannel()
			return
		}
		w.voice.channelKnown, w.voice.channelID, w.voice.channelName = true, *selected.ChannelID, ""
		w.send("GET_CHANNEL", map[string]any{"channel_id": *selected.ChannelID}, "channel", "", requestTimeout, now)
	case "VIDEO_STATE_UPDATE":
		w.voice.videoKnown, w.voice.video = activeOf(data, w.voice.video)
	case "SCREENSHARE_STATE_UPDATE":
		w.voice.screenKnown, w.voice.screen = activeOf(data, w.voice.screen)
	case "ERROR":
		var e rpcError
		_ = json.Unmarshal(data, &e) // Only reported.
		w.link.Fail(e.Error(), now)
	}
}

// activeOf reads an {"active": bool} state update.
func activeOf(data json.RawMessage, previous bool) (bool, bool) {
	var state struct {
		Active *bool `json:"active"`
	}
	if json.Unmarshal(data, &state) != nil || state.Active == nil {
		return false, previous
	}
	return true, *state.Active
}

func (w *worker) voiceSettings(data json.RawMessage) {
	var settings struct {
		Mute *bool `json:"mute"`
		Deaf *bool `json:"deaf"`
	}
	if json.Unmarshal(data, &settings) != nil || settings.Mute == nil || settings.Deaf == nil {
		return
	}
	w.voice.settingsKnown, w.voice.mute, w.voice.deaf = true, *settings.Mute, *settings.Deaf
}

func (w *worker) leftChannel() {
	w.voice.channelKnown, w.voice.channelID, w.voice.channelName = true, "", ""
	w.voice.videoKnown, w.voice.video = true, false
	w.voice.screenKnown, w.voice.screen = true, false
}

func (w *worker) callDone(ctx context.Context, c call, data json.RawMessage, now time.Time) {
	switch c.purpose {
	case "authorize":
		var granted struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(data, &granted) != nil || granted.Code == "" {
			w.state = stateUnauthorized
			w.notes["discord.connect"] = "Failed"
			return
		}
		w.startExchange(ctx, granted.Code)
	case "authenticate":
		w.state = stateReady
		w.backoff = 0
		w.link.Observe(snoofer.ConnectionConnected, now)
		w.send("GET_VOICE_SETTINGS", map[string]any{}, "voice", "", requestTimeout, now)
		w.send("GET_SELECTED_VOICE_CHANNEL", map[string]any{}, "selected", "", requestTimeout, now)
		for _, evt := range []string{"VOICE_SETTINGS_UPDATE", "VOICE_CHANNEL_SELECT", "VIDEO_STATE_UPDATE", "SCREENSHARE_STATE_UPDATE"} {
			w.sendEvent("SUBSCRIBE", evt, map[string]any{}, "subscribe", "", requestTimeout, now)
		}
	case "voice":
		w.voiceSettings(data)
	case "selected":
		var channel struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if string(data) == "null" || json.Unmarshal(data, &channel) != nil || channel.ID == "" {
			w.leftChannel()
			return
		}
		w.voice.channelKnown, w.voice.channelID, w.voice.channelName = true, channel.ID, channel.Name
	case "channel":
		var channel struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &channel) == nil && channel.ID == w.voice.channelID {
			w.voice.channelName = channel.Name
		}
	}
}

func (w *worker) callFailed(c call, err error, now time.Time) {
	switch c.purpose {
	case "authorize":
		// Denied in Discord, or the popup timed out.
		w.state = stateUnauthorized
		w.notes["discord.connect"] = "Failed"
	case "authenticate":
		w.access = ""
		if w.settings.RefreshToken != "" {
			w.state = stateRetrying
			w.nextRefresh = now
		} else {
			w.state = stateUnauthorized
		}
	case "write":
		delete(w.pending, c.id)
		w.notes[c.id] = "Failed"
	case "subscribe", "channel":
		// Optional state: the related control shows N/A.
	}
	w.link.Fail(err.Error(), now)
}

// startExchange requests tokens off the worker: from an authorization code,
// or from the saved refresh token when code is empty.
func (w *worker) startExchange(ctx context.Context, code string) {
	w.state = stateExchanging
	settings, generation := w.settings, w.generation
	w.helpers.Add(1)
	go func() {
		defer w.helpers.Done()
		t, err := exchange(ctx, w.client, w.endpoint, settings, code)
		select {
		case w.tokens <- tokenResult{generation: generation, t: t, err: err}:
		case <-ctx.Done():
		}
	}()
}

func (w *worker) tokenDone(ctx context.Context, r tokenResult, now time.Time) {
	switch {
	case errors.Is(r.err, errInvalidGrant):
		w.saveRefresh("", now)
		if w.conn != nil {
			w.state = stateUnauthorized
		}
		w.link.Fail(r.err.Error(), now)
		return
	case r.err != nil:
		w.link.Fail(r.err.Error(), now)
		w.backoff = min(max(2*w.backoff, redial), maxBackoff)
		if w.conn != nil && r.generation == w.generation {
			w.state = stateRetrying
			w.nextRefresh = now.Add(w.backoff)
		}
		return
	}
	// Refresh tokens rotate: save the new one even if the pipe has closed.
	if r.t.Refresh != "" {
		w.saveRefresh(r.t.Refresh, now)
	}
	if w.conn == nil || r.generation != w.generation {
		return
	}
	w.access = r.t.Access
	w.state = stateAuthenticating
	w.send("AUTHENTICATE", map[string]any{"access_token": r.t.Access}, "authenticate", "", requestTimeout, now)
}

// saveRefresh persists the refresh token with a compare-and-swap of the
// settings. A failed save keeps the token in memory for this session.
func (w *worker) saveRefresh(token string, now time.Time) {
	next := w.settings
	next.RefreshToken = token
	w.settings = next
	raw, err := json.Marshal(next)
	if err == nil && w.services.SaveSettings != nil {
		err = w.services.SaveSettings("discord", w.raw, raw)
	}
	if err != nil {
		w.link.Fail("saving Discord authorization: "+err.Error(), now)
		return
	}
	w.raw = raw
}
