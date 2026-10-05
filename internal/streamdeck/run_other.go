//go:build !windows

package streamdeck

import (
	"context"
	"fmt"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
)

func Start(ctx context.Context, cfg *config.StreamDeck) (chan control.State, <-chan Event) {
	s := make(chan control.State, 1)
	e := make(chan Event)
	close(e)
	return s, e
}
func Media(string) error { return fmt.Errorf("Windows media unavailable") }
