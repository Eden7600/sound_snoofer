package soundboard

import "testing"

func TestClipLabel(t *testing.T) {
	for stem, want := range map[string]string{
		"air_horn":         "Air Horn",
		"sad-trombone":     "Sad Trombone",
		"rick_and_morty":   "Rick and Morty",
		"the_end":          "The End",
		"OMG-wow":          "OMG Wow",
		"new_iPhone_sound": "New iPhone Sound",
		"bruh":             "Bruh",
		"__double__under":  "Double Under",
		"Already Titled":   "Already Titled",
		"élan_vital":       "Élan Vital",
	} {
		if got := clipLabel(stem); got != want {
			t.Errorf("clipLabel(%q) = %q, want %q", stem, got, want)
		}
	}
}
