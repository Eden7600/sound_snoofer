package app

import (
	"context"
	"encoding/json"
	"io"
	"maps"

	"sound-snoofer/snoofer"
)

// Frames are private to one inherited-pipe controls session. No commands are
// replayed after disconnection; the worker still validates action revisions.
type frame struct {
	State      *ViewState                        `json:",omitempty"`
	Focus      bool                              `json:",omitempty"`
	Animations map[string][]snoofer.ArtworkFrame `json:",omitempty"` // Every animated control's frames, sent only when they change.
}

// animationSender tracks the artwork whose animations a controls session has,
// so the frames cross the pipe once rather than with every state.
type animationSender struct {
	sent map[string]string // Control ID to the Artwork its frames animate.
}

// changed returns every animation when the set differs from what was sent,
// otherwise nil.
func (a *animationSender) changed(controls []snoofer.Control) map[string][]snoofer.ArtworkFrame {
	current := map[string]string{}
	frames := map[string][]snoofer.ArtworkFrame{}
	for _, c := range controls {
		if len(c.Animation) > 1 {
			current[c.ID] = c.Artwork
			frames[c.ID] = c.Animation
		}
	}
	if a.sent != nil && maps.Equal(current, a.sent) {
		return nil
	}
	a.sent = current
	return frames
}

func readActions(ctx context.Context, r io.Reader, actions chan<- UIAction) error {
	decoder := json.NewDecoder(r)
	for {
		var action UIAction
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
	case old := <-ch:
		var previous, next frame
		// Focus and animations survive replacement; state is superseded.
		if json.Unmarshal(old, &previous) == nil && (previous.Focus || previous.Animations != nil) && json.Unmarshal(data, &next) == nil {
			next.Focus = next.Focus || previous.Focus
			if next.Animations == nil {
				next.Animations = previous.Animations
			}
			if merged, err := json.Marshal(next); err == nil {
				data = merged
			}
		}
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
