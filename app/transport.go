package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"maps"
	"slices"
	"sync"

	"sound-snoofer/snoofer"
)

// Frames are private to one inherited-pipe controls session. No commands are
// replayed after disconnection; the worker still validates action revisions.
type frame struct {
	Version    uint64                             `json:",omitempty"`
	Result     *ActionResult                      `json:",omitempty"`
	State      *ViewState                         `json:",omitempty"`
	Focus      bool                               `json:",omitempty"`
	Artwork    *map[string]string                 `json:",omitempty"`
	Animations *map[string][]snoofer.ArtworkFrame `json:",omitempty"`
}

// ViewUpdate is a private GUI response. A missing Artwork retains the browser's
// cache; an explicit empty map clears it.
type ViewUpdate struct {
	Version          uint64
	State            ViewState
	AnimationVersion uint64
	Artwork          *map[string]string `json:",omitempty"`
}

type viewSender struct {
	version    uint64
	state      []byte
	artwork    map[string]string
	animations map[string][]snoofer.ArtworkFrame
}

func separateArtwork(state ViewState) (ViewState, map[string]string, map[string][]snoofer.ArtworkFrame) {
	state.Controls = slices.Clone(state.Controls)
	artwork := map[string]string{}
	animations := map[string][]snoofer.ArtworkFrame{}
	for i, c := range state.Controls {
		if c.Artwork != "" {
			artwork[c.ID] = c.Artwork
		}
		if len(c.Animation) > 1 {
			animations[c.ID] = c.Animation
		}
		state.Controls[i].Artwork = ""
		state.Controls[i].Animation = nil
	}
	return state, artwork, animations
}

func (s *viewSender) prepare(state ViewState, focus bool) ([]byte, error) {
	stripped, artwork, animations := separateArtwork(state)
	stateJSON, err := json.Marshal(stripped)
	if err != nil {
		return nil, err
	}
	artChanged := s.version == 0 || !maps.Equal(artwork, s.artwork)
	animationChanged := s.version == 0 || !maps.EqualFunc(animations, s.animations, func(a, b []snoofer.ArtworkFrame) bool {
		return slices.Equal(a, b)
	})
	changed := !bytes.Equal(stateJSON, s.state) || artChanged || animationChanged
	if !changed && !focus {
		return nil, nil
	}
	f := frame{Version: s.version, State: &stripped, Focus: focus}
	if changed {
		f.Version++
	}
	if artChanged {
		f.Artwork = &artwork
	}
	if animationChanged {
		f.Animations = &animations
	}
	data, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	// Baselines describe successfully encoded frames; offer retains pending maps.
	s.version, s.state, s.artwork, s.animations = f.Version, stateJSON, artwork, animations
	return data, nil
}

const actionCapacity = 16

// A reply retains its original child session even after another child opens.
// The lane is never closed; cancellation prevents late sends without panics.
type actionSession struct {
	ctx     context.Context
	cancel  context.CancelFunc
	replies chan ActionResult
}

func newActionSession(ctx context.Context, cancel context.CancelFunc) *actionSession {
	return &actionSession{ctx: ctx, cancel: cancel, replies: make(chan ActionResult, actionCapacity)}
}

func (s *actionSession) completion(id string) snoofer.Completion {
	var once sync.Once
	return func(outcome snoofer.Outcome) {
		once.Do(func() {
			if s.ctx.Err() != nil {
				return
			}
			result := ActionResult{ID: id, Status: outcome.Status, Detail: outcome.Detail}
			select {
			case s.replies <- result:
			case <-s.ctx.Done():
			default:
				// A client can reserve only sixteen replies. Detach on a broken
				// invariant so its outstanding calls become Unknown, never lost.
				s.cancel()
			}
		})
	}
}

func readActions(ctx context.Context, r io.Reader, actions chan<- UIAction, session *actionSession) error {
	decoder := json.NewDecoder(r)
	for {
		var action UIAction
		if err := decoder.Decode(&action); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if action.ID != "" && session != nil {
			action.session = session
			action.complete = session.completion(action.ID)
			select {
			case actions <- action:
			default:
				action.reply(snoofer.Outcome{Status: snoofer.Rejected, Detail: "Controls busy; try again"})
			}
			continue
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
		// Focus and complete asset replacements survive newer presentation state.
		if json.Unmarshal(old, &previous) == nil && (previous.Focus || previous.Artwork != nil || previous.Animations != nil) && json.Unmarshal(data, &next) == nil {
			next.Focus = next.Focus || previous.Focus
			if next.Artwork == nil {
				next.Artwork = previous.Artwork
			}
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

func writeFrames(ctx context.Context, w io.Writer, updates <-chan []byte, replies <-chan ActionResult) error {
	encoder := json.NewEncoder(w)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		// The bounded result lane has priority over a continuous telemetry stream.
		select {
		case result := <-replies:
			if err := encoder.Encode(frame{Result: &result}); err != nil {
				return err
			}
			continue
		default:
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-replies:
			if err := encoder.Encode(frame{Result: &result}); err != nil {
				return err
			}
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
