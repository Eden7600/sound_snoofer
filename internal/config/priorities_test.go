package config

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestGeneratedPatterns(t *testing.T) {
	name := "Speakers (AirPods Pro)"
	if got := ExactPattern(name); got != `(?i)^Speakers \(AirPods Pro\)$` {
		t.Fatal(got)
	}
	if got := DevicePattern(name); got != `(?i)AirPods Pro` {
		t.Fatal(got)
	}
	for _, name := range []string{"Line 1+2 [Volt.2]", "Out (C++ [x].y)", "a|b*"} {
		exact := regexp.MustCompile(ExactPattern(name))
		if !exact.MatchString(name) || !exact.MatchString(strings.ToUpper(name)) || exact.MatchString(name+"x") {
			t.Fatalf("exact pattern for %q", name)
		}
		if p := DevicePattern(name); p != "" && !regexp.MustCompile(p).MatchString(name) {
			t.Fatalf("device pattern for %q", name)
		}
	}
	for _, name := range []string{"Volt ASIO", "()", "(Device)", "Speakers ( )"} {
		if p := DevicePattern(name); p != "" {
			t.Fatalf("device pattern for %q: %q", name, p)
		}
	}
}

func editDefault(t *testing.T, edits ...PriorityEdit) (Config, []byte) {
	t.Helper()
	raw := DefaultBytes()
	for _, e := range edits {
		var err error
		if raw, err = EditPriorities(raw, e); err != nil {
			t.Fatal(e, err)
		}
	}
	c, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	return c, raw
}

func TestEditPlaybackList(t *testing.T) {
	entry, _ := json.Marshal(Candidate{Driver: "wdm", Pattern: ExactPattern("Speakers (Studio <1>)")})
	c, raw := editDefault(t,
		PriorityEdit{List: ListPlayback, Op: "add", Value: string(entry)},
		PriorityEdit{List: ListPlayback, Op: "move", Index: 3, Value: "up"},
		PriorityEdit{List: ListPlayback, Op: "remove", Index: 0},
		PriorityEdit{List: ListPlayback, Op: "set", Index: 0, Field: "pattern", Value: "(?i)arena"},
	)
	got := []string{}
	for _, candidate := range c.Studio.Playback {
		got = append(got, candidate.Pattern)
	}
	want := []string{"(?i)arena", `(?i)^Speakers \(Studio <1>\)$`, "(?i)^Universal Audio Volt$"}
	if !slices.Equal(got, want) {
		t.Fatal(got)
	}
	if !strings.Contains(string(raw), "<1>") {
		t.Fatal("patterns must stay readable", string(raw))
	}
	// Untouched settings survive.
	if c.PollMS != 1000 || c.Studio.Voice == nil || !c.Studio.MovePlaybackRouting || len(c.Studio.ASIO) != 1 {
		t.Fatal(c)
	}
}

func TestEditInterfacesAndMicrophones(t *testing.T) {
	c, _ := editDefault(t,
		PriorityEdit{List: ListInterfaces, Op: "set", Index: 0, Field: "input:lav", Value: "0"},
		PriorityEdit{List: ListInterfaces, Op: "set", Index: 0, Field: "input:desk", Value: "3,4"},
		PriorityEdit{List: ListInterfaces, Op: "set", Index: 0, Field: "presence_pattern", Value: "(?i)volt"},
		PriorityEdit{List: ListMicrophones, Op: "add", Value: "desk"},
		PriorityEdit{List: ListMicrophones, Op: "move", Index: 2, Value: "up"},
	)
	if in := c.Studio.ASIO[0].Inputs; len(in) != 1 || in.Left("desk") != 3 || in.Right("desk") != 4 || in.Left("lav") != 0 || c.Studio.ASIO[0].PresencePattern != "(?i)volt" {
		t.Fatal(c.Studio.ASIO[0])
	}
	if !slices.Equal(c.Profiles.Microphones, []string{"lav", "desk", "webcam"}) {
		t.Fatal(c.Profiles.Microphones)
	}
}

func TestRejectedPriorityEdits(t *testing.T) {
	for _, e := range []PriorityEdit{
		{List: ListPlayback, Op: "set", Index: 0, Field: "pattern", Value: "(unclosed"},
		{List: ListWebcam, Op: "set", Index: 0, Field: "driver", Value: "asio"},
		{List: ListInterfaces, Op: "remove", Index: 0},
		{List: ListInterfaces, Op: "set", Index: 0, Field: "input:desk", Value: "65"},
		{List: ListInterfaces, Op: "set", Index: 0, Field: "input:webcam", Value: "1"},
		{List: ListInterfaces, Op: "set", Index: 0, Field: "input:desk", Value: "1,2,3"},
		{List: ListPlayback, Op: "remove", Index: 9},
		{List: ListPlayback, Op: "move", Index: 0, Value: "up"},
		{List: ListPlayback, Op: "add", Value: `{"driver":"wdm","pattern":"x","extra":1}`},
		{List: ListMicrophones, Op: "add", Value: "lav"},
		{List: ListMicrophones, Op: "add", Value: "vr:index"},
		{List: "outputs", Op: "add"},
		{List: ListPlayback, Op: "rename", Index: 0},
	} {
		if _, err := EditPriorities(DefaultBytes(), e); err == nil {
			t.Fatalf("accepted %+v", e)
		}
	}
}

// Older configurations omit profiles; the first edit writes the effective
// default priority out instead of an empty list.
func TestEditMicrophonesWithoutProfiles(t *testing.T) {
	var top map[string]json.RawMessage
	json.Unmarshal(DefaultBytes(), &top)
	delete(top, "profiles")
	raw, _ := json.Marshal(top)
	out, err := EditPriorities(raw, PriorityEdit{List: ListMicrophones, Op: "add", Value: "desk"})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := Decode(out)
	if !slices.Equal(c.Profiles.Microphones, []string{"lav", "webcam", "desk"}) {
		t.Fatal(c.Profiles.Microphones)
	}
}
