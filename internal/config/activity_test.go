package config

import (
	"strings"
	"testing"
)

func activityConfig(activity string) (Config, error) {
	b := strings.Replace(string(DefaultBytes()), `"profiles": {`, `"profiles": {"activity": `+activity+`,`, 1)
	return Decode([]byte(b))
}

func TestActivityConfig(t *testing.T) {
	c, err := activityConfig(`{"check": 2}`)
	if err != nil {
		t.Fatal(err)
	}
	if a := c.Profiles.Activity; a.Check != 2 || a.SilenceDB != -70 || a.SilentAfterS != 10 {
		t.Fatal(a)
	}
	for _, bad := range []string{`{}`, `{"check": 5}`, `{"check": 1, "silence_db": -10}`, `{"check": 1, "silence_db": -130}`, `{"check": 1, "silent_after_s": 1}`, `{"check": 1, "extra": 1}`} {
		if _, err := activityConfig(bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	if c, _ := Decode(DefaultBytes()); c.Profiles.Activity != nil {
		t.Fatal("activity must be opt-in")
	}
}
