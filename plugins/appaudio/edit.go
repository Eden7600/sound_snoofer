package appaudio

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
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
	// The first edit materializes the default exclusions so they can be removed.
	next := w.settings
	next.Picked, next.Exclude, next.Rules = slices.Clone(w.settings.Picked), w.settings.excluded(), slices.Clone(w.settings.Rules)
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
	case "exclude":
		pattern, err := normalizePattern(e.Value)
		if err != nil {
			return next, err
		}
		if !slices.Contains(next.Exclude, pattern) {
			next.Exclude = append(next.Exclude, pattern)
		}
		return next, nil
	case "include":
		pattern := strings.ToLower(strings.TrimSpace(e.Value))
		if !slices.Contains(next.Exclude, pattern) {
			return next, fmt.Errorf("%s is not excluded", e.Value)
		}
		next.Exclude = slices.DeleteFunc(next.Exclude, func(p string) bool { return p == pattern })
		return next, nil
	case "recent":
		minutes, err := strconv.Atoi(strings.TrimSpace(e.Value))
		if err != nil || minutes < 1 || minutes > 24*60 {
			return next, fmt.Errorf("recent window must be 1–1440 minutes")
		}
		next.RecentMinutes = minutes
		return next, nil
	case "separate":
		// Removes a rule naming programs into this app; combined programs
		// return as their own apps and a rename reverts.
		index := slices.IndexFunc(next.Rules, func(r Rule) bool { return r.Match == e.Value && !r.Hide && key(r.Name) == key(e.App) })
		if index < 0 {
			return next, fmt.Errorf("that rule no longer exists")
		}
		next.Rules = slices.Delete(next.Rules, index, index+1)
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
		for _, p := range target.paths() {
			if file := programFile(p); file != "" && !slices.Contains(next.Exclude, file) {
				next.Exclude = append(next.Exclude, file)
			}
		}
		if pickIndex >= 0 {
			next.Picked = slices.Delete(next.Picked, pickIndex, pickIndex+1)
		}
	case "unhide":
		// Every pattern hiding one of its programs goes, wildcards included.
		next.Exclude = slices.DeleteFunc(next.Exclude, func(pattern string) bool {
			return slices.ContainsFunc(target.paths(), func(p string) bool {
				_, hit := excludedBy([]string{pattern}, p)
				return hit
			})
		})
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
