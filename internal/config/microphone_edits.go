package config

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Microphone definition lists: ListMics edits the microphones themselves
// (add, remove, move, set name); ListMicDevices+<id> edits a device
// microphone's device priority like any other device list.
const (
	ListMics       = "mics"
	ListMicDevices = "mic-devices:"
)

// editMicrophoneDefinitions edits studio.microphones. A configuration still
// in the legacy form is first written out explicitly, replacing fallback_mic.
func editMicrophoneDefinitions(raw []byte, top map[string]json.RawMessage, e PriorityEdit) error {
	c, err := Decode(raw)
	if err != nil {
		return err
	}
	if c.Studio == nil {
		return fmt.Errorf("configuration has no studio profile")
	}
	studio := map[string]json.RawMessage{}
	if err := json.Unmarshal(top["studio"], &studio); err != nil {
		return fmt.Errorf("studio: %w", err)
	}
	mics := slices.Clone(c.Studio.Mics())
	delete(studio, "fallback_mic")

	if id, ok := strings.CutPrefix(e.List, ListMicDevices); ok {
		_, n, found := c.Studio.Microphone(id)
		if !found || !mics[n].IsDevice() {
			return fmt.Errorf("%s is not a device microphone", id)
		}
		devices, err := editList(slices.Clone(mics[n].Devices), e, parseEntry[Candidate], setCandidateField)
		if err != nil {
			return err
		}
		mics[n].Devices = devices
	} else {
		var removed string
		if e.Op == "remove" && e.Index >= 0 && e.Index < len(mics) {
			removed = mics[e.Index].ID
		}
		mics, err = editList(mics, e, func(value string) (Microphone, error) {
			m, err := parseEntry[Microphone](value)
			m.Name = strings.TrimSpace(m.Name)
			m.ID = uniqueMicrophoneID(mics, m.Name)
			return m, err
		}, func(m *Microphone, field, value string) error {
			if field != "name" {
				return fmt.Errorf("unknown field %q", field)
			}
			m.Name = strings.TrimSpace(value)
			return nil
		})
		if err != nil {
			return err
		}
		if removed != "" {
			if err := forgetMicrophone(top, studio, removed); err != nil {
				return err
			}
		}
	}
	if err := setKey(studio, "microphones", mics); err != nil {
		return err
	}
	return setKey(top, "studio", studio)
}

// forgetMicrophone removes references to a removed microphone: its
// priority entry, interface channels and a voice source naming it.
func forgetMicrophone(top, studio map[string]json.RawMessage, id string) error {
	var interfaces []map[string]json.RawMessage
	if err := unmarshalList(studio["asio"], &interfaces); err != nil {
		return err
	}
	for _, a := range interfaces {
		var inputs MicInputs
		if raw, ok := a["inputs"]; ok {
			if err := json.Unmarshal(raw, &inputs); err != nil {
				return err
			}
			delete(inputs, id)
			if err := setKey(a, "inputs", inputs); err != nil {
				return err
			}
		}
	}
	if interfaces != nil {
		if err := setKey(studio, "asio", interfaces); err != nil {
			return err
		}
	}
	var voice map[string]json.RawMessage
	if raw, ok := studio["voice"]; ok && json.Unmarshal(raw, &voice) == nil {
		if string(voice["source"]) == `"`+id+`"` {
			voice["source"] = json.RawMessage(`"auto"`)
			if err := setKey(studio, "voice", voice); err != nil {
				return err
			}
		}
	}
	profiles := map[string]json.RawMessage{}
	if raw, ok := top["profiles"]; ok {
		if err := json.Unmarshal(raw, &profiles); err != nil {
			return err
		}
		var priority []string
		if err := unmarshalList(profiles["microphones"], &priority); err != nil {
			return err
		}
		priority = slices.DeleteFunc(priority, func(p string) bool { return p == id })
		if len(priority) == 0 {
			return fmt.Errorf("the microphone priority needs another microphone first")
		}
		if err := setKey(profiles, "microphones", priority); err != nil {
			return err
		}
		return setKey(top, "profiles", profiles)
	}
	return nil
}

// uniqueMicrophoneID derives a stable ID from a name, avoiding reserved
// choices; renames keep it so saved choices stay attached.
func uniqueMicrophoneID(mics []Microphone, name string) string {
	base := strings.Trim(nonID.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if base == "" || base == "off" || base == "auto" || base == "normal" || strings.HasPrefix(base, "vr") {
		base = strings.Trim("mic-"+base, "-")
	}
	if len(base) > 20 {
		base = base[:20]
	}
	id := base
	for n := 2; slices.ContainsFunc(mics, func(m Microphone) bool { return m.ID == id }); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}
