package browserbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	protocol       = 2
	maxMessage     = 1 << 20
	helloTimeout   = 5 * time.Second
	idleTimeout    = 60 * time.Second
	writeTimeout   = 2 * time.Second
	commandTimeout = 5 * time.Second
)

// Bridge is an inert, app-owned transport with independent typed consumers.
type Bridge struct {
	ctx            context.Context
	mu             sync.Mutex
	status         Status
	configErr      error
	cleanupErr     error
	closed         bool
	run            *listenerRun
	media          *MediaLease
	meet           *MeetLease
	conns          map[string]*connection
	pending        map[string]*meetPending
	nextGeneration uint64
	nextRequest    uint64
	retry          time.Duration
}

type listenerRun struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	server   *http.Server
	stopping bool
	wg       sync.WaitGroup
	sockets  map[*websocket.Conn]bool
	err      error // Published by closing done.
}

type lease struct {
	b      *Bridge
	ctx    context.Context
	cancel context.CancelFunc
	active bool // Guarded by b.mu.
	once   sync.Once
	done   chan struct{}
	wg     sync.WaitGroup
	err    error // Published by closing done.
}

// MediaLease owns only the Media subscription and its sends.
type MediaLease struct {
	*lease
	updates chan MediaUpdate
}

// MeetLease owns only Meet observations and bounded correlated requests.
type MeetLease struct {
	*lease
	updates chan MeetUpdate
}

// New extracts the legacy port without validating unrelated provider settings.
// Invalid configuration is diagnostic state rather than an unrelated host failure.
func New(ctx context.Context, legacySettings json.RawMessage) *Bridge {
	b := &Bridge{ctx: ctx, status: Status{Port: 47815}, conns: map[string]*connection{}, pending: map[string]*meetPending{}, retry: 5 * time.Second}
	var settings struct {
		Port json.RawMessage `json:"port"`
	}
	if len(legacySettings) > 0 {
		b.configErr = json.Unmarshal(legacySettings, &settings)

	}
	if b.configErr == nil && len(settings.Port) > 0 {
		var port int
		if bytes.Equal(bytes.TrimSpace(settings.Port), []byte("null")) {
			b.configErr = errors.New("port must be an integer")
		} else {
			b.configErr = json.Unmarshal(settings.Port, &port)
		}
		if port != 0 {
			b.status.Port = port
		}
	}
	if b.configErr == nil && (b.status.Port < 1024 || b.status.Port > 65535) {
		b.configErr = errors.New("port must be 1024–65535")
	}
	if b.configErr != nil {
		b.status.Error = "browser bridge settings: " + b.configErr.Error()
	}
	return b
}

func (b *Bridge) newLeaseLocked(ctx context.Context) (*lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b.closed || b.ctx.Err() != nil {
		return nil, errors.New("browser bridge stopped")
	}
	if b.run != nil && b.run.stopping {
		return nil, errors.New("browser bridge is stopping")
	}
	runCtx, cancel := context.WithCancel(ctx)
	l := &lease{b: b, ctx: runCtx, cancel: cancel, active: true, done: make(chan struct{})}
	go func() {
		select {
		case <-runCtx.Done():
		case <-b.ctx.Done():
		}
		_ = l.Close(context.Background()) // Owner cancellation still joins cleanup.
	}()
	return l, nil
}

func (b *Bridge) AcquireMedia(ctx context.Context) (*MediaLease, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.media != nil {
		return nil, errors.New("Media bridge already acquired")
	}
	l, err := b.newLeaseLocked(ctx)
	if err != nil {
		return nil, err
	}
	m := &MediaLease{lease: l, updates: make(chan MediaUpdate, 16)}
	b.media = m
	b.startLocked()
	for _, c := range b.connectionsLocked() {
		if c.meet != 1 {
			// Old extensions cannot refresh one-shot artwork on a feature change.
			b.retireLocked(c, "")
			continue
		}
		b.mediaLocked(c, nil)
		b.capabilitiesLocked(c)
	}
	return m, nil
}

func (b *Bridge) AcquireMeet(ctx context.Context) (*MeetLease, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.meet != nil {
		return nil, errors.New("Meet bridge already acquired")
	}
	l, err := b.newLeaseLocked(ctx)
	if err != nil {
		return nil, err
	}
	m := &MeetLease{lease: l, updates: make(chan MeetUpdate, 16)}
	b.meet = m
	b.startLocked()
	for _, c := range b.connectionsLocked() {
		b.meetLocked(c, nil)
		b.capabilitiesLocked(c)
	}
	return m, nil
}

func (b *Bridge) startLocked() {
	if b.run != nil {
		return
	}
	ctx, cancel := context.WithCancel(b.ctx)
	r := &listenerRun{ctx: ctx, cancel: cancel, done: make(chan struct{}), sockets: map[*websocket.Conn]bool{}}
	b.run = r
	b.listenLocked(r)
	go b.runListener(r)
}

func (b *Bridge) listenLocked(r *listenerRun) {
	if r.ctx.Err() != nil || b.configErr != nil || r.server != nil {
		return
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(b.status.Port)))
	if err != nil {
		b.status.Error = fmt.Sprintf("browser bridge port %d: %v", b.status.Port, err)
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/nowplaying", func(w http.ResponseWriter, req *http.Request) { b.serve(r, w, req) })
	r.server = &http.Server{Handler: mux, ReadHeaderTimeout: helloTimeout}
	b.status.Error = ""
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		_ = r.server.Serve(listener) // Owned shutdown closes the listener below.
	}()
}

func (b *Bridge) runListener(r *listenerRun) {
	ticker := time.NewTicker(b.retry)
	defer ticker.Stop()
	for r.ctx.Err() == nil {
		select {
		case <-r.ctx.Done():
		case <-ticker.C:
			b.mu.Lock()
			b.listenLocked(r)
			b.mu.Unlock()
		}
	}
	b.mu.Lock()
	r.stopping = true
	for _, c := range b.connectionsLocked() {
		b.retireLocked(c, "")
	}
	for ws := range r.sockets {
		_ = ws.Close() // Includes sockets still waiting for hello.
	}
	server := r.server
	b.mu.Unlock()
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := server.Shutdown(ctx)
		cancel()
		if err != nil {
			// Only this owner's HTTP connections are forced closed after graceful
			// cleanup fails. Preserve that failure before permitting a host restart.
			r.err = fmt.Errorf("browser bridge shutdown: %w", errors.Join(err, server.Close()))
		}
	}
	r.wg.Wait()
	b.mu.Lock()
	if b.run == r {
		b.run = nil
		b.cleanupErr = r.err
	}
	close(r.done)
	b.mu.Unlock()
}

func (b *Bridge) Status() Status {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.status
}

func (b *Bridge) Close(ctx context.Context) error {
	b.mu.Lock()
	b.closed = true
	media, meet, r, priorErr := b.media, b.meet, b.run, b.cleanupErr
	if r != nil {
		r.stopping = true
		r.cancel()
	}
	b.mu.Unlock()
	var errs []error
	if media != nil {
		errs = append(errs, media.Close(ctx))
	}
	if meet != nil {
		errs = append(errs, meet.Close(ctx))
	}
	if r != nil {
		if err := wait(ctx, r.done); err != nil {
			errs = append(errs, err)
		} else if r.err != nil && !errors.Is(errors.Join(errs...), r.err) {
			// Parent cancellation may finish the listener before a captured
			// lease begins Close, so retain the owner error independently.
			errs = append(errs, r.err)
		}
	} else {
		errs = append(errs, priorErr)
	}
	return errors.Join(errs...)
}

func (l *lease) Close(ctx context.Context) error {
	l.once.Do(func() {
		b := l.b
		b.mu.Lock()
		l.active = false
		l.cancel()
		if b.media != nil && b.media.lease == l {
			closeUpdates(b.media.updates)
			b.media = nil
		}
		if b.meet != nil && b.meet.lease == l {
			closeUpdates(b.meet.updates)
			b.meet = nil
		}
		r := b.run
		last := b.media == nil && b.meet == nil
		if last && r != nil {
			r.stopping = true
			r.cancel()
		} else {
			for _, c := range b.connectionsLocked() {
				b.capabilitiesLocked(c)
			}
		}
		b.mu.Unlock()
		go func() {
			l.wg.Wait()
			if last && r != nil {
				<-r.done
				l.err = r.err
			}
			close(l.done)
		}()
	})
	if err := wait(ctx, l.done); err != nil {
		return err
	}
	return l.err
}

// Only the owner produces updates, under b.mu. A concurrent consumer may take
// the last buffered item, so draining must never depend on a preceding len check.
func closeUpdates[T any](updates chan T) {
	for {
		select {
		case <-updates:
		default:
			close(updates)
			return
		}
	}
}

func wait(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *lease) Generation(browser string) uint64 {
	l.b.mu.Lock()
	defer l.b.mu.Unlock()
	if c := l.b.conns[browser]; l.active && l.ctx.Err() == nil && c != nil && !l.b.closed && !c.run.stopping && c.run.ctx.Err() == nil {
		return c.generation
	}
	return 0
}

func (l *lease) Status() Status                   { return l.b.Status() }
func (l *MediaLease) Updates() <-chan MediaUpdate { return l.updates }
func (l *MeetLease) Updates() <-chan MeetUpdate   { return l.updates }
