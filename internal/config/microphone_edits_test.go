package config

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestMicrophoneDefinitionEdits(t *testing.T) {
	yeti, _ := json.Marshal(Microphone{Name: "Yeti", Devices: []Candidate{{Driver: "wdm", ID: "{yeti}", Name: "Microphone (Yeti)"}}})
	c, raw := editDefault(t,
		// The first edit writes the legacy set out explicitly.
		PriorityEdit{List: ListMics, Op: "set", Index: 1, Field: "name", Value: "Lav"},
		PriorityEdit{List: ListMics, Op: "add", Value: string(yeti)},
		PriorityEdit{List: ListMics, Op: "move", Index: 3, Value: "up"},
		PriorityEdit{List: ListMicDevices + "webcam", Op: "add", Value: `{"driver":"wdm","pattern":"(?i)c920"}`},
	)
	if got := c.Studio.MicrophoneIDs(); !slices.Equal(got, []string{"desk", "lav", "yeti", "webcam"}) {
		t.Fatal(got)
	}
	if c.Studio.Microphones[1].Name != "Lav" || len(c.Studio.Microphones[3].Devices) != 2 || c.Studio.Microphones[2].Devices[0].ID != "{yeti}" {
		t.Fatal(c.Studio.Microphones)
	}
	if strings.Contains(string(raw), "fallback_mic") {
		t.Fatal("fallback_mic kept beside explicit microphones")
	}
	// Removing a microphone forgets its priority entry and channels.
	raw, err := EditPriorities(raw, PriorityEdit{List: ListMics, Op: "remove", Index: 1})
	if err != nil {
		t.Fatal(err)
	}
	c, _ = Decode(raw)
	if slices.Contains(c.Profiles.Microphones, "lav") || c.Studio.ASIO[0].Inputs.Left("lav") != 0 || c.Studio.ASIO[0].Inputs.Left("desk") != 1 {
		t.Fatal(c.Profiles.Microphones, c.Studio.ASIO[0].Inputs)
	}
	for _, e := range []PriorityEdit{
		{List: ListMicDevices + "desk", Op: "add", Value: `{"driver":"wdm","pattern":"x"}`},
		{List: ListMicDevices + "yeti", Op: "remove", Index: 0},
		{List: ListMics, Op: "set", Index: 0, Field: "name", Value: ""},
		{List: ListMics, Op: "remove", Index: 2},
	} {
		if _, err := EditPriorities(raw, e); err == nil {
			t.Fatalf("accepted %+v", e)
		}
	}
	if id := uniqueMicrophoneID(nil, "VR Mic"); id != "mic-vr-mic" {
		t.Fatal(id)
	}
}
