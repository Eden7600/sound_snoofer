package browserbridge

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

type meetPending struct {
	id      string
	lease   *MeetLease
	conn    *connection
	command MeetCommand
	started bool // Guarded by b.mu; results cannot confirm an unsent request.
	in      chan MeetResult
	out     chan MeetResult
}

func (l *MeetLease) Send(ctx context.Context, browser string, command MeetCommand) (<-chan MeetResult, error) {
	if !validCommand(command) {
		return nil, fmt.Errorf("invalid Meet command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b := l.b
	b.mu.Lock()
	defer b.mu.Unlock()
	c := b.conns[browser]
	if !l.active || l.ctx.Err() != nil || c == nil || c.generation != command.Generation || c.run.stopping {
		return nil, ErrConnectionChanged
	}
	if c.meet != 1 {
		return nil, ErrUnavailable
	}
	if len(b.pending) >= 8 {
		return nil, ErrBusy
	}
	for _, p := range b.pending {
		old := p.command
		if p.conn == c && old.Tab == command.Tab && old.Instance == command.Instance && old.Call == command.Call && old.Control == command.Control {
			return nil, ErrBusy
		}
	}
	if b.nextRequest == ^uint64(0) {
		return nil, errors.New("Meet request IDs exhausted")
	}
	b.nextRequest++
	p := &meetPending{id: strconv.FormatUint(b.nextRequest, 10), lease: l, conn: c, command: command, in: make(chan MeetResult, 1), out: make(chan MeetResult, 1)}
	b.pending[p.id] = p
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	stop := context.AfterFunc(l.ctx, cancel)
	l.wg.Add(1)
	c.run.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer c.run.wg.Done()
		defer cancel()
		defer stop()
		result := b.runMeet(ctx, p)
		b.mu.Lock()
		delete(b.pending, p.id)
		b.mu.Unlock()
		p.out <- result
		close(p.out)
	}()
	return p.out, nil
}

func (p *meetPending) result(status, reason string) MeetResult {
	c := p.command
	return MeetResult{ID: p.id, Tab: c.Tab, Instance: c.Instance, Call: c.Call, Control: c.Control, Revision: c.Revision, Value: c.Value, Status: status, Reason: reason}
}

func (b *Bridge) runMeet(ctx context.Context, p *meetPending) MeetResult {
	wire := struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		MeetCommand
	}{Type: "meet-command", ID: p.id, MeetCommand: p.command}
	started, err := b.write(ctx, p.conn, wire, func() bool {
		if !p.lease.active || p.lease.ctx.Err() != nil || b.pending[p.id] != p {
			return false
		}
		p.started = true
		return true
	})
	if err != nil {
		if started {
			return p.result("unknown", "disconnected")
		}
		if p.lease.ctx.Err() != nil || errors.Is(ctx.Err(), context.Canceled) {
			return p.result("rejected", "disabled")
		}
		if errors.Is(err, ErrConnectionChanged) {
			return p.result("rejected", "stale")
		}
		return p.result("rejected", "busy")
	}
	select {
	case result := <-p.in:
		return result
	case <-p.conn.done:
		return p.result("unknown", "disconnected")
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return p.result("unknown", "not-confirmed")
		}
		return p.result("unknown", "stopped")
	}
}

func (b *Bridge) acceptResult(c *connection, result MeetResult) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.pending[result.ID]
	if p == nil || p.conn != c || b.conns[c.browser] != c || !p.lease.active || !p.started {
		return
	}
	expected := p.result(result.Status, result.Reason)
	if expected != result {
		return
	}
	select {
	case p.in <- result:
	default: // The first matching result owns this operation.
	}
}
