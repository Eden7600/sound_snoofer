package windowsaudio

import (
	"context"
	"fmt"
	"time"
)

type Request struct {
	Enabled, Live     bool
	Playback, Capture string
	// Ack is closed by the native worker after all previous corrections finish.
	Ack chan struct{}
}

// StatusKind is independent of the displayed diagnostic text.
type StatusKind int

const (
	Unknown StatusKind = iota
	Disabled
	Preview
	Verified
	Attention
)

// Result reports the latest default-policy observation.
type Result struct {
	Status string
	Kind   StatusKind
	// Diagnostics: resolved target endpoint names, contention suspension and
	// the most recent correction. They never affect the policy decision.
	Playback, Capture string
	Suspended         bool
	LastCorrection    time.Time
}
type Endpoint struct{ ID, Name string }
type Backend interface {
	Endpoints(int) ([]Endpoint, error)
	Default(int, int) (string, error)
	Set(string, int) error
	Close()
}
type Guard struct {
	Last      [2][]time.Time
	Suspended [2]bool
	Enabled   bool
}

func (g *Guard) Reconcile(b Backend, r Request, targets [2]string, now time.Time) Result {
	if !r.Enabled {
		*g = Guard{}
		return Result{Status: "Default protection Off", Kind: Disabled}
	}
	if !r.Live {
		return Result{Status: "Preview: Windows defaults unchanged", Kind: Preview}
	}
	result := Result{Status: "Verified", Kind: Verified}
	for direction, target := range targets {
		if target == "" {
			result = Result{Status: "Voicemeeter default endpoint unavailable or ambiguous", Kind: Attention}
			continue
		}
		if g.Suspended[direction] {
			result = Result{Status: "Default protection suspended; toggle off/on to retry", Kind: Attention}
			continue
		}
		mismatch := false
		for role := 0; role < 3; role++ {
			id, err := b.Default(direction, role)
			if err != nil {
				result = Result{Status: err.Error(), Kind: Attention}
				continue
			}
			mismatch = mismatch || id != target
		}
		if !mismatch {
			continue
		}
		recent := []time.Time{}
		for _, at := range g.Last[direction] {
			if now.Sub(at) < 10*time.Second {
				recent = append(recent, at)
			}
		}
		if len(recent) >= 3 {
			g.Suspended[direction] = true
			result = Result{Status: "Default protection suspended: repeated contention", Kind: Attention}
			continue
		}
		g.Last[direction] = append(recent, now)
		for role := 0; role < 3; role++ {
			id, e := b.Default(direction, role)
			if e != nil {
				result = Result{Status: e.Error(), Kind: Attention}
				continue
			}
			if id == target {
				continue
			}
			if e = b.Set(target, role); e != nil {
				result = Result{Status: e.Error(), Kind: Attention}
				continue
			}
			id, e = b.Default(direction, role)
			if e != nil || id != target {
				result = Result{Status: "Windows default change unverified", Kind: Attention}
			}
		}
	}
	return result
}

// Update replaces superseded policy; the control worker is the sole sender.
func Update(requests chan Request, r Request) {
	select {
	case <-requests:
	default:
	}
	requests <- r
}

// Revoke waits until the native apartment has finished any previous correction.
func Revoke(ctx context.Context, requests chan Request, done <-chan struct{}) error {
	ack := make(chan struct{})
	Update(requests, Request{Ack: ack})
	select {
	case <-ack:
		return nil
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("Windows default enforcement shutdown unverified: %w", ctx.Err())
	}
}

// Start owns the native apartment and publishes latest status without blocking callers.
func Start(ctx context.Context) (chan Request, chan Result, <-chan struct{}) {
	requests := make(chan Request, 1)
	results := make(chan Result, 1)
	done := make(chan struct{})
	go func() { defer close(done); run(ctx, requests, results) }()
	return requests, results, done
}
