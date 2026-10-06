package appaudio

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"

	"sound-snoofer/internal/windowsaudio"
)

// app is the sessions that share a display name after rules.
type app struct {
	Name     string
	Icon     string
	Rule     string // Diagnostic: the rule that named or hid it.
	Hidden   bool
	Sessions []windowsaudio.Session
}

// key is the case-insensitive identity of a display name.
func key(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// controlID is stable across restarts; it changes only when the app is renamed.
func controlID(name string) string {
	return fmt.Sprintf("appaudio.app-%x", sha256.Sum256([]byte(key(name))))[:len("appaudio.app-")+12]
}

// group applies exclusions, then rules, and merges sessions into apps in
// first-seen order. Sessions without a known program keep their own name.
func group(sessions []windowsaudio.Session, rules []compiledRule, exclude []string) []*app {
	var out []*app
	byKey := map[string]*app{}
	for _, s := range sessions {
		name := s.Name
		rule, matched := matchRule(rules, s.Path)
		if matched && rule.Name != "" {
			name = rule.Name
		}
		hidden, reason := matched && rule.Hide, ""
		if matched {
			reason = rule.describe()
		}
		if pattern, ok := excludedBy(exclude, s.Path); ok {
			hidden, reason = true, "Excluded ("+pattern+")"
		}
		k := key(name)
		if hidden {
			k = "hidden\x00" + k
		}
		a, ok := byKey[k]
		if !ok {
			a = &app{Name: name, Hidden: hidden, Rule: reason}
			byKey[k] = a
			out = append(out, a)
		}
		if a.Icon == "" {
			a.Icon = s.Icon
		}
		a.Sessions = append(a.Sessions, s)
	}
	return out
}

// volume is the highest session volume.
func (a *app) volume() float64 {
	v := 0.0
	for _, s := range a.Sessions {
		v = max(v, s.Volume)
	}
	return v
}

// muteState is "all", "some" or "none".
func (a *app) muteState() string {
	muted := 0
	for _, s := range a.Sessions {
		if s.Muted {
			muted++
		}
	}
	switch {
	case len(a.Sessions) > 0 && muted == len(a.Sessions):
		return "all"
	case muted > 0:
		return "some"
	}
	return "none"
}

// paths lists the app's executables.
func (a *app) paths() []string {
	var out []string
	for _, s := range a.Sessions {
		if !slices.Contains(out, s.Path) {
			out = append(out, s.Path)
		}
	}
	return out
}

// visible orders picked apps first, in pick order, then unpicked apps heard
// within the window, by name. Picked apps without sessions are included as
// closed placeholders.
func visible(apps []*app, picked []string, heard map[string]time.Time, window time.Duration, now time.Time) []*app {
	byKey := map[string]*app{}
	for _, a := range apps {
		if !a.Hidden {
			byKey[key(a.Name)] = a
		}
	}
	var out []*app
	seen := map[string]bool{}
	for _, name := range picked {
		k := key(name)
		if seen[k] {
			continue
		}
		seen[k] = true
		if a, ok := byKey[k]; ok {
			out = append(out, a)
		} else {
			out = append(out, &app{Name: name})
		}
	}
	var others []*app
	for k, a := range byKey {
		if !seen[k] && now.Sub(heard[k]) <= window {
			others = append(others, a)
		}
	}
	slices.SortFunc(others, func(a, b *app) int { return strings.Compare(key(a.Name), key(b.Name)) })
	return append(out, others...)
}
