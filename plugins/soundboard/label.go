package soundboard

import (
	"strings"
	"unicode"
)

// minorWords stay lower case in a title unless they come first.
var minorWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "nor": true, "but": true, "for": true, "of": true,
	"on": true, "in": true, "to": true, "at": true, "by": true, "as": true, "per": true, "via": true, "vs": true,
}

// clipLabel is a clip's display label: its file stem in title case, with
// underscores and hyphens read as spaces (openspec/changes/soundboard-title-labels).
// Words with upper case after the first letter, such as OMG or iPhone, keep
// their case.
func clipLabel(stem string) string {
	words := strings.FieldsFunc(stem, func(r rune) bool { return r == '_' || r == '-' || unicode.IsSpace(r) })
	for n, word := range words {
		runes := []rune(word)
		switch {
		case strings.ContainsFunc(string(runes[1:]), unicode.IsUpper):
			// OMG, iPhone: keep as written.
		case n > 0 && minorWords[strings.ToLower(word)]:
			words[n] = strings.ToLower(word)
		default:
			runes[0] = unicode.ToUpper(runes[0])
			words[n] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
