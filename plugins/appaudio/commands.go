package appaudio

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"sound-snoofer/snoofer"
)

type sessionIdentity struct {
	key, path string
	pid       uint32
}

type appContext struct {
	key        string
	generation uint64
	sessions   []sessionIdentity
	wantMute   bool
}

type command struct {
	request            snoofer.Request
	target             appContext
	targetRequired     bool
	focusGeneration    uint64
	settingsGeneration uint64
	expectedSettings   string
	complete           snoofer.Completion
}

func appKey(a *app) string {
	k := key(a.Name)
	if a.Hidden {
		return "hidden\x00" + k
	}
	return k
}

// refreshContexts ignores telemetry but remembers observed disappearance and regrouping.
func (w *worker) refreshContexts(now time.Time) {
	current := make(map[string]appContext, len(w.apps))
	for _, a := range w.apps {
		k := appKey(a)
		identities := make([]sessionIdentity, 0, len(a.Sessions))
		for _, s := range a.Sessions {
			identities = append(identities, sessionIdentity{key: s.Key, path: s.Path, pid: s.PID})
		}
		slices.SortFunc(identities, func(a, b sessionIdentity) int { return strings.Compare(a.key, b.key) })
		previous, exists := w.contexts[k]
		generation := previous.generation
		if !exists || !slices.Equal(previous.sessions, identities) {
			w.nextGeneration++
			generation = w.nextGeneration
		}
		wantMute := a.muteState() != "all"
		if pending, ok := w.pending[key(a.Name)]; ok && pending.mute {
			wantMute = !pending.wantMute
		}
		current[k] = appContext{key: k, generation: generation, sessions: identities, wantMute: wantMute}
	}
	w.contexts = current
	focused := w.focusedApp(visible(w.apps, w.settings.Picked, w.heard, w.settings.window(), now))
	var target appContext
	if focused != nil {
		target = current[appKey(focused)]
	}
	if w.focusSelection != w.focus || w.focusTarget.key != target.key || w.focusTarget.generation != target.generation {
		w.focusGeneration++
	}
	w.focusSelection, w.focusTarget = w.focus, target
}

// capture uses only immutable values closed over by the published handler.
func capture(r snoofer.Request, contexts map[string]appContext, focused appContext, focusGeneration, settingsGeneration uint64, raw string) command {
	c := command{request: r, expectedSettings: raw, settingsGeneration: settingsGeneration}
	switch r.ID {
	case "appaudio.focus":
		c.target, c.focusGeneration = focused, focusGeneration
	case "appaudio.edit":
		var e edit
		if json.Unmarshal([]byte(r.Value), &e) == nil {
			switch e.Op {
			case "hide", "rename", "combine":
				c.targetRequired = true
				c.target = contexts[key(e.App)]
			case "unhide":
				c.targetRequired = true
				c.target = contexts["hidden\x00"+key(e.App)]
				if c.target.key == "" {
					c.target = contexts[key(e.App)]
				}
			}
		}
	default:
		for _, target := range contexts {
			if !strings.HasPrefix(target.key, "hidden\x00") && controlID(target.key) == r.ID {
				c.target = target
				break
			}
		}
	}
	return c
}

func (w *worker) process(ctx context.Context, c command, now time.Time) {
	result := snoofer.Outcome{Status: snoofer.Rejected, Detail: "App selection changed; try again"}
	w.refreshContexts(now)
	switch {
	case ctx.Err() != nil:
		result = snoofer.Outcome{Status: snoofer.Unknown, Detail: "App audio stopped before completion"}
	case c.request.ID == "appaudio.focus" && (c.focusGeneration != w.focusGeneration || c.target.key != w.focusTarget.key):
	case c.targetRequired && c.target.key == "":
	case c.target.key != "" && c.target.generation != w.contexts[c.target.key].generation:
	case (c.request.ID == "appaudio.edit" || c.request.ID == "appaudio.deck-apps") && (c.settingsGeneration != w.settingsGeneration || c.expectedSettings != string(w.raw)):
		result.Detail = "App audio settings changed; try again"
	default:
		result = w.handle(c, now)
	}
	c.complete(result)
}

func failedOutcome(err error) snoofer.Outcome {
	status := snoofer.Failed
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		status = snoofer.Unknown
	}
	return snoofer.Outcome{Status: status, Detail: err.Error()}
}
