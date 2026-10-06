package appaudio

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"sound-snoofer/internal/windowsaudio"
)

// Rule hides, renames or explicitly shows sessions whose executable path
// matches. The first matching rule decides; a rule with neither Hide nor Name
// shows the app as Windows reports it and stops later rules.
type Rule struct {
	Match string `json:"match"` // Case-insensitive regexp on the executable path, or "system".
	Name  string `json:"name,omitempty"`
	Hide  bool   `json:"hide,omitempty"`
}

// defaultRules follow the user's rules and hide audio plumbing.
var defaultRules = []Rule{
	{Match: `(^|\\)snoofer\.exe$`, Hide: true},
	{Match: `(^|\\)voicemeeter[^\\]*\.exe$`, Hide: true},
	{Match: `(^|\\)audiodg\.exe$`, Hide: true},
}

type compiledRule struct {
	Rule
	pattern *regexp.Regexp
	builtIn bool
}

// compileRules prepares the user's rules followed by the defaults.
func compileRules(user []Rule) ([]compiledRule, error) {
	var out []compiledRule
	for n, r := range append(slices.Clone(user), defaultRules...) {
		pattern, err := regexp.Compile("(?i)" + r.Match)
		if err != nil || r.Match == "" {
			return nil, fmt.Errorf("rule %d: invalid match %q", n+1, r.Match)
		}
		out = append(out, compiledRule{Rule: r, pattern: pattern, builtIn: n >= len(user)})
	}
	return out, nil
}

// matchRule returns the first rule matching a session's executable path.
func matchRule(rules []compiledRule, path string) (compiledRule, bool) {
	if path == "" {
		return compiledRule{}, false
	}
	for _, r := range rules {
		if r.pattern.MatchString(path) {
			return r, true
		}
	}
	return compiledRule{}, false
}

// describe is a rule's diagnostic text.
func (r compiledRule) describe() string {
	switch {
	case r.Rule == (Rule{}):
		return ""
	case r.Hide && r.builtIn:
		return "Hidden by default (" + r.Match + ")"
	case r.Hide:
		return "Hidden (" + r.Match + ")"
	case r.Name != "":
		return "Named " + r.Name + " (" + r.Match + ")"
	default:
		return "Shown (" + r.Match + ")"
	}
}

// exeMatch is the rule pattern for exactly the given executables.
func exeMatch(paths []string) (string, error) {
	var names []string
	for _, p := range paths {
		if p == windowsaudio.SystemSounds {
			names = append(names, "")
			continue
		}
		if p != "" {
			names = append(names, regexp.QuoteMeta(strings.ToLower(filepath.Base(p))))
		}
	}
	slices.Sort(names)
	names = slices.Compact(names)
	if len(names) == 0 {
		return "", fmt.Errorf("this app's program is unknown")
	}
	var parts []string
	system := false
	for _, n := range names {
		if n == "" {
			system = true
			continue
		}
		parts = append(parts, `(^|\\)`+n+`$`)
	}
	if system {
		parts = append(parts, `^system$`)
	}
	return strings.Join(parts, "|"), nil
}
