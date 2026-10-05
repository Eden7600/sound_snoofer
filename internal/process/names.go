// Package process observes exact executable names in the current Windows session.
package process

import "strings"

// Contains matches an executable name without inferring hardware presence.
func Contains(names []string, want string) bool {
	for _, name := range names {
		if strings.EqualFold(name, want) {
			return true
		}
	}
	return false
}
