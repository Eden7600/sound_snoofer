//go:build windows

package streamdeck

import (
	"context"

	"sound-snoofer/internal/presence"
)

// Tests never depend on whether this machine is locked: the watcher reports
// an unknown state unless a test supplies its own.
func init() {
	watchPresence = func(ctx context.Context) (<-chan presence.State, <-chan struct{}) {
		states, done := make(chan presence.State, 1), make(chan struct{})
		states <- presence.State{}
		go func() { <-ctx.Done(); close(done) }()
		return states, done
	}
}
