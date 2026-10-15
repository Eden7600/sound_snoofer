package nowplaying

import (
	"context"
	"errors"
	"strings"
	"time"

	"sound-snoofer/snoofer"
)

// Track/capability changes invalidate queued work; progress and artwork do not.
type sessionTarget struct {
	key, source, instance, title, artist, album, track string
	generation                                         uint64
	duration                                           int64
	play, pause, next, prev, seek, mute                bool
}

func targetOf(s session) sessionTarget {
	return sessionTarget{s.Key, s.Source, s.Instance, s.Title, s.Artist, s.Album, s.Track, s.Generation,
		s.DurationMs, s.CanPlay, s.CanPause, s.CanNext, s.CanPrev, s.CanSeek, s.CanMute}
}

type mediaContext struct {
	target sessionTarget
	epoch  uint64
}

type command struct {
	request                             snoofer.Request
	target                              session
	context                             mediaContext
	focusGeneration, settingsGeneration uint64
	expectedSettings                    string
	wantDeckOff                         bool
	op                                  string
	rating                              string
	value                               int64
	complete                            snoofer.Completion
}

func (w *worker) refreshContexts() {
	next := make(map[string]mediaContext, len(w.sessions))
	for _, s := range w.sessions {
		target := targetOf(s)
		current := w.contexts[s.Key]
		if current.epoch == 0 || current.target != target {
			w.nextContext++
			current = mediaContext{target, w.nextContext}
		}
		next[s.Key] = current
	}
	w.contexts = next
	focus := next[w.focus]
	if w.focusContext != focus || w.focusChosen != w.chosen {
		w.focusGeneration++
		w.focusContext, w.focusChosen = focus, w.chosen
	}
}

// capture runs against the immutable publication, before its handler enqueues.
func capture(r snoofer.Request, sessions map[string]session, contexts map[string]mediaContext, focus session, focusGeneration, settingsGeneration uint64, raw string, deckOff bool) command {
	c := command{request: r}
	switch r.ID {
	case "nowplaying.deck-media":
		c.settingsGeneration, c.expectedSettings, c.wantDeckOff = settingsGeneration, raw, !deckOff
		return c
	case "nowplaying.dial", "nowplaying.toggle", "nowplaying.next", "nowplaying.prev", "nowplaying.mute", "nowplaying.like", "nowplaying.dislike":
		c.target, c.focusGeneration = focus, focusGeneration
	default:
		c.target = sessions[r.ID]
	}
	c.context = contexts[c.target.Key]
	c.op = strings.TrimPrefix(r.ID, "nowplaying.")
	if c.op == "like" || c.op == "dislike" {
		c.rating = c.op
		if c.target.Rating == c.rating {
			c.rating = "none"
		}
		c.op = "rate"
		return c
	}
	if c.op == "dial" || c.op != "next" && c.op != "prev" && c.op != "mute" {
		c.op = "toggle"
	}
	if c.op == "toggle" && !c.target.playing() || c.op == "mute" && !c.target.Muted {
		c.value = 1
	}
	return c
}

func reject(detail string) snoofer.Outcome {
	return snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}
}

func operationError(err error) snoofer.Outcome {
	status := snoofer.Failed
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		status = snoofer.Unknown
	}
	return snoofer.Outcome{Status: status, Detail: err.Error()}
}

func (w *worker) process(ctx context.Context, c command, now time.Time) {
	result := func() snoofer.Outcome {
		if ctx.Err() != nil {
			return snoofer.Outcome{Status: snoofer.Unknown, Detail: "Now Playing stopped before completion"}
		}
		w.refreshContexts()
		if c.settingsGeneration != 0 {
			if c.settingsGeneration != w.settingsGeneration || c.expectedSettings != string(w.raw) {
				return reject("Media settings changed; try again")
			}
		} else {
			if c.focusGeneration != 0 && c.focusGeneration != w.focusGeneration {
				return reject("Media focus changed; try again")
			}
			if c.request.ID != "nowplaying.dial" || c.request.Operation != "reset" {
				if c.context.epoch == 0 || c.context != w.contexts[c.target.Key] || !w.targetAvailable(c.target) {
					return reject("Media session changed or unavailable; try again")
				}
			}
		}
		return w.handle(c, now)
	}()
	if c.complete != nil {
		c.complete(result)
	}
}

func (w *worker) targetAvailable(s session) bool {
	if s.Instance == "" || s.Generation == 0 {
		return false
	}
	if s.Source == "windows" {
		return w.win != nil && s.Generation == w.winGeneration
	}
	return w.browserGeneration(s.Source) == s.Generation
}
