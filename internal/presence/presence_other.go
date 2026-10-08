//go:build !windows

package presence

import "context"

// Watch reports an unknown state once; other platforms do not observe it.
func Watch(ctx context.Context) (<-chan State, <-chan struct{}) {
	states, done := make(chan State, 1), make(chan struct{})
	states <- State{}
	close(done)
	return states, done
}
