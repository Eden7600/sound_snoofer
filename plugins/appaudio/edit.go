package appaudio

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// edit is one GUI change to picks or rules.
type edit struct {
	Op    string `json:"op"`
	App   string `json:"app"`
	Value string `json:"value,omitempty"`
}

// edit applies a GUI change and saves it. Failures are reported on the
// status control rather than returned, because input is queued.
func (w *worker) edit(raw string) {
	var e edit
	if err := json.Unmarshal([]byte(raw), &e); err != nil || strings.TrimSpace(e.App) == "" {
		w.editErr = "Invalid app edit"
		return
	}
	next, err := w.applyEdit(e)
	if err == nil {
		err = w.save(next)
	}
	w.editErr = ""
	if err != nil {
		w.editErr = err.Error()
	}
}

func (w *worker) applyEdit(e edit) (Settings, error) {
	next := Settings{Picked: slices.Clone(w.settings.Picked), Rules: slices.Clone(w.settings.Rules), RecentMinutes: w.settings.RecentMinutes}
	pickIndex := slices.IndexFunc(next.Picked, func(p string) bool { return key(p) == key(e.App) })
	switch e.Op {
	case "pick":
		if pickIndex < 0 {
			next.Picked = append(next.Picked, e.App)
		}
		return next, nil
	case "unpick":
		if pickIndex >= 0 {
			next.Picked = slices.Delete(next.Picked, pickIndex, pickIndex+1)
		}
		return next, nil
	case "move":
		to := pickIndex - 1
		if e.Value == "down" {
			to = pickIndex + 1
		}
		if pickIndex < 0 || to < 0 || to >= len(next.Picked) {
			return next, fmt.Errorf("cannot move %s", e.App)
		}
		next.Picked[pickIndex], next.Picked[to] = next.Picked[to], next.Picked[pickIndex]
		return next, nil
	}

	// Rule edits need the app's executables.
	var target *app
	for _, a := range w.apps {
		if key(a.Name) == key(e.App) && (target == nil || a.Hidden == (e.Op == "unhide")) {
			target = a
		}
	}
	if target == nil {
		return next, fmt.Errorf("%s is not running", e.App)
	}
	match, err := exeMatch(target.paths())
	if err != nil {
		return next, err
	}
	// GUI rules replace any earlier rule for exactly these executables.
	next.Rules = slices.DeleteFunc(next.Rules, func(r Rule) bool { return r.Match == match })
	switch e.Op {
	case "hide":
		next.Rules = slices.Insert(next.Rules, 0, Rule{Match: match, Hide: true})
		if pickIndex >= 0 {
			next.Picked = slices.Delete(next.Picked, pickIndex, pickIndex+1)
		}
	case "unhide":
		// Without its own rule, a default still hides it: show explicitly.
		if rules, err := compileRules(next.Rules); err == nil {
			for _, p := range target.paths() {
				if r, ok := matchRule(rules, p); ok && r.Hide {
					next.Rules = slices.Insert(next.Rules, 0, Rule{Match: match})
					break
				}
			}
		}
	case "rename", "combine":
		name := strings.TrimSpace(e.Value)
		if name == "" {
			return next, fmt.Errorf("name required")
		}
		next.Rules = slices.Insert(next.Rules, 0, Rule{Match: match, Name: name})
		if pickIndex >= 0 {
			// The renamed app takes the pick's place; a pick of the combined
			// target elsewhere is dropped.
			next.Picked = slices.Delete(next.Picked, pickIndex, pickIndex+1)
			next.Picked = slices.DeleteFunc(next.Picked, func(p string) bool { return key(p) == key(name) })
			pickIndex = min(pickIndex, len(next.Picked))
			next.Picked = slices.Insert(next.Picked, pickIndex, name)
		}
	default:
		return next, fmt.Errorf("unknown edit %q", e.Op)
	}
	return next, nil
}

func (w *worker) save(next Settings) error {
	rules, err := compileRules(next.Rules)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := w.services.SaveSettings("appaudio", w.raw, raw); err != nil {
		return err
	}
	w.raw, w.settings, w.rules = raw, next, rules
	w.listAt = time.Time{} // Regroup on the next step.
	return nil
}
