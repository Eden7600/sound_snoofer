package appaudio

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"sound-snoofer/internal/windowsaudio"
)

// defaultExclude hides audio plumbing and Hue Sync, which reacts to system
// audio and makes its own session behave erratically.
var defaultExclude = []string{"snoofer.exe", "voicemeeter*.exe", "audiodg.exe", "huesync.exe"}

// excluded is the exclusion list in effect: the defaults until the user
// edits it, then exactly the saved list.
func (s Settings) excluded() []string {
	if s.Exclude == nil {
		return slices.Clone(defaultExclude)
	}
	return slices.Clone(s.Exclude)
}

// programFile is the name exclusion patterns match: the executable's file
// name, or "system" for System sounds.
func programFile(p string) string {
	if p == windowsaudio.SystemSounds {
		return "system"
	}
	if p == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(p))
}

// normalizePattern validates and lower-cases an exclusion pattern.
func normalizePattern(pattern string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(pattern))
	if p == "" || strings.ContainsAny(p, `/\`) {
		return "", fmt.Errorf("exclude %q: use a program file name such as game.exe", pattern)
	}
	if _, err := path.Match(p, ""); err != nil {
		return "", fmt.Errorf("exclude %q: invalid pattern", pattern)
	}
	return p, nil
}

// excludedBy returns the first pattern matching a session's program.
func excludedBy(patterns []string, executable string) (string, bool) {
	file := programFile(executable)
	if file == "" {
		return "", false
	}
	for _, p := range patterns {
		if ok, _ := path.Match(strings.ToLower(p), file); ok {
			return p, true
		}
	}
	return "", false
}
