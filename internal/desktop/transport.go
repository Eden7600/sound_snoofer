package desktop

import (
	"context"
	"encoding/json"
	"io"

	"sound-snoofer/internal/tui"
)

// Frames are private to one inherited-pipe controls session. No commands are
// replayed after disconnection; the worker still validates action revisions.
type frame struct {
	State *tui.State `json:",omitempty"`
	Focus bool       `json:",omitempty"`
}

func readActions(ctx context.Context, r io.Reader, actions chan<- tui.Action) error {
	decoder := json.NewDecoder(r)
	for {
		var action tui.Action
		if err := decoder.Decode(&action); err != nil {
			return err
		}
		select {
		case actions <- action:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// offer replaces only stale presentation state, never commands.
func offer(ch chan []byte, data []byte) {
	select {
	case ch <- data:
		return
	default:
	}
	select {
	case <-ch:
	default:
	}
	select {
	case ch <- data:
	default:
	}
}

func writeFrames(ctx context.Context, w io.Writer, updates <-chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-updates:
			if !ok {
				return nil
			}
			if _, err := w.Write(append(data, '\n')); err != nil {
				return err
			}
		}
	}
}
