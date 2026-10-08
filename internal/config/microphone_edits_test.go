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

func TestMicrophoneReadyAndSourceEdits(t *testing.T) {
	c, raw := editDefault(t,
		PriorityEdit{List: ListMics, Op: "set", Index: 1, Field: "ready", Value: "true"},
		// The desk becomes a device: no interface keeps channels for it.
		PriorityEdit{List: ListMics, Op: "set", Index: 0, Field: "source", Value: `{"driver":"wdm","id":"{yeti}","name":"Microphone (Yeti)"}`},
	)
	if !c.Studio.Microphones[1].Ready || c.Studio.Microphones[0].Ready {
		t.Fatal(c.Studio.Microphones)
	}
	if !c.Studio.Microphones[0].IsDevice() || c.Studio.ASIO[0].Inputs.Left("desk") != 0 || c.Studio.ASIO[0].Inputs.Left("lav") != 2 {
		t.Fatal(c.Studio.Microphones[0], c.Studio.ASIO[0].Inputs)
	}
	raw, err := EditPriorities(raw, PriorityEdit{List: ListMics, Op: "set", Index: 0, Field: "source", Value: "interface"})
	if err != nil {
		t.Fatal(err)
	}
	c, _ = Decode(raw)
	if c.Studio.Microphones[0].IsDevice() {
		t.Fatal("still a device")
	}
	// Channels for it may be mapped again.
	if _, err := EditPriorities(raw, PriorityEdit{List: ListInterfaces, Op: "set", Index: 0, Field: "input:desk", Value: "3,4"}); err != nil {
		t.Fatal(err)
	}
	for _, e := range []PriorityEdit{
		{List: ListMics, Op: "set", Index: 0, Field: "ready", Value: "maybe"},
		{List: ListMics, Op: "set", Index: 0, Field: "source", Value: `{"bogus":1}`},
	} {
		if _, err := EditPriorities(raw, e); err == nil {
			t.Fatalf("accepted %+v", e)
		}
	}
}

func TestActivityEdits(t *testing.T) {
	c, raw := editDefault(t,
		PriorityEdit{List: ListActivity, Op: "set", Field: "check", Value: "2"},
		PriorityEdit{List: ListActivity, Op: "set", Field: "silence_db", Value: "-60"},
		PriorityEdit{List: ListActivity, Op: "set", Field: "silent_after_s", Value: "5"},
	)
	if a := c.Profiles.Activity; a == nil || a.Check != 2 || a.SilenceDB != -60 || a.SilentAfterS != 5 {
		t.Fatal(c.Profiles.Activity)
	}
	for _, e := range []PriorityEdit{
		{List: ListActivity, Op: "set", Field: "check", Value: "5"},
		{List: ListActivity, Op: "set", Field: "silence_db", Value: "-5"},
		{List: ListActivity, Op: "set", Field: "silent_after_s", Value: "x"},
		{List: ListActivity, Op: "add", Value: "1"},
	} {
		if _, err := EditPriorities(raw, e); err == nil {
			t.Fatalf("accepted %+v", e)
		}
	}
	raw, err := EditPriorities(raw, PriorityEdit{List: ListActivity, Op: "set", Field: "check", Value: "0"})
	if err != nil {
		t.Fatal(err)
	}
	c, _ = Decode(raw)
	if c.Profiles.Activity != nil {
		t.Fatal("activity kept")
	}
	if _, err := EditPriorities(raw, PriorityEdit{List: ListActivity, Op: "set", Field: "silence_db", Value: "-60"}); err == nil {
		t.Fatal("threshold set with metering off")
	}
}
