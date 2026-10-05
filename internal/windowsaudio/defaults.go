package windowsaudio

import (
	"context"
	"time"
)

type Request struct {
	Enabled, Live     bool
	Playback, Capture string
}
type Result struct{ Status string }
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

func (g *Guard) Reconcile(b Backend, r Request, targets [2]string, now time.Time) string {
	if !r.Enabled {
		*g = Guard{}
		return "Default protection Off"
	}
	if !r.Live {
		return "Preview: Windows defaults unchanged"
	}
	result := "Verified"
	for direction, target := range targets {
		if target == "" {
			result = "Voicemeeter default endpoint unavailable or ambiguous"
			continue
		}
		if g.Suspended[direction] {
			result = "Default protection suspended; toggle off/on to retry"
			continue
		}
		mismatch := false
		for role := 0; role < 3; role++ {
			id, err := b.Default(direction, role)
			if err != nil {
				result = err.Error()
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
			result = "Default protection suspended: repeated contention"
			continue
		}
		g.Last[direction] = append(recent, now)
		for role := 0; role < 3; role++ {
			id, e := b.Default(direction, role)
			if e != nil {
				result = e.Error()
				continue
			}
			if id == target {
				continue
			}
			if e = b.Set(target, role); e != nil {
				result = e.Error()
				continue
			}
			id, e = b.Default(direction, role)
			if e != nil || id != target {
				result = "Windows default change unverified"
			}
		}
	}
	return result
}

// Start owns the native apartment and publishes latest status without blocking callers.
func Start(ctx context.Context) (chan Request, chan Result) {
	requests := make(chan Request, 1)
	results := make(chan Result, 1)
	go run(ctx, requests, results)
	return requests, results
}
