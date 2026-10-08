package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Editable priority lists and the configuration keys that hold them.
const (
	ListInterfaces  = "interfaces"
	ListPlayback    = "playback"
	ListWebcam      = "webcam"
	ListMicrophones = "microphones"
)

var studioListKeys = map[string]string{ListInterfaces: "asio", ListPlayback: "playback", ListWebcam: "fallback_mic"}

// PriorityEdit is one change to an editable priority list. Index addresses an
// existing entry; Value carries the new entry (JSON for device lists, an ID
// for microphones), a direction for move, or a field value for set.
type PriorityEdit struct {
	List  string `json:"list"`
	Op    string `json:"op"`
	Index int    `json:"index"`
	Field string `json:"field,omitempty"`
	Value string `json:"value,omitempty"`
}

// ExactPattern matches exactly name, ignoring case.
func ExactPattern(name string) string {
	return "(?i)^" + regexp.QuoteMeta(name) + "$"
}

// DevicePattern matches the device part of a Windows endpoint name shaped
// like "Endpoint (Device)" anywhere in a name, ignoring case, so the entry
// survives Windows renaming the endpoint. It is empty for other names.
func DevicePattern(name string) string {
	name = strings.TrimSpace(name)
	open := strings.LastIndex(name, "(")
	if open <= 0 || !strings.HasSuffix(name, ")") {
		return ""
	}
	inner := strings.TrimSpace(name[open+1 : len(name)-1])
	if inner == "" {
		return ""
	}
	return "(?i)" + regexp.QuoteMeta(inner)
}

// EditPriorities applies e to the audio configuration raw and returns the
// edited configuration. Only the edited list is rewritten; the result must
// decode and validate, otherwise the edit is refused.
func EditPriorities(raw []byte, e PriorityEdit) ([]byte, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("configuration: %w", err)
	}
	var err error
	if e.List == ListMicrophones {
		err = editMicrophones(top, e)
	} else if key, ok := studioListKeys[e.List]; ok {
		err = editStudioList(top, key, e)
	} else {
		err = fmt.Errorf("unknown list %q", e.List)
	}
	if err != nil {
		return nil, err
	}
	out, err := encode(top, "  ")
	if err != nil {
		return nil, err
	}
	if _, err := Decode(out); err != nil {
		return nil, err
	}
	return out, nil
}

func editMicrophones(top map[string]json.RawMessage, e PriorityEdit) error {
	profiles := map[string]json.RawMessage{}
	if raw, ok := top["profiles"]; ok {
		if err := json.Unmarshal(raw, &profiles); err != nil {
			return fmt.Errorf("profiles: %w", err)
		}
	}
	list := slices.Clone(DefaultProfiles().Microphones)
	if raw, ok := profiles["microphones"]; ok {
		list = nil
		if err := json.Unmarshal(raw, &list); err != nil {
			return fmt.Errorf("profiles.microphones: %w", err)
		}
	}
	if e.Op == "add" && slices.Contains(list, e.Value) {
		return fmt.Errorf("%s is already in the list", e.Value)
	}
	list, err := editList(list, e, func(value string) (string, error) { return value, nil }, nil)
	if err != nil {
		return err
	}
	if err := setKey(profiles, "microphones", list); err != nil {
		return err
	}
	return setKey(top, "profiles", profiles)
}

func editStudioList(top map[string]json.RawMessage, key string, e PriorityEdit) error {
	studio := map[string]json.RawMessage{}
	raw, ok := top["studio"]
	if !ok {
		return fmt.Errorf("configuration has no studio profile")
	}
	if err := json.Unmarshal(raw, &studio); err != nil {
		return fmt.Errorf("studio: %w", err)
	}
	var err error
	if key == "asio" {
		var list []ASIOInterface
		if err = unmarshalList(studio[key], &list); err != nil {
			return err
		}
		if list, err = editList(list, e, parseEntry[ASIOInterface], setInterfaceField); err != nil {
			return err
		}
		err = setKey(studio, key, list)
	} else {
		var list []Candidate
		if err = unmarshalList(studio[key], &list); err != nil {
			return err
		}
		if list, err = editList(list, e, parseEntry[Candidate], setCandidateField); err != nil {
			return err
		}
		err = setKey(studio, key, list)
	}
	if err != nil {
		return err
	}
	return setKey(top, "studio", studio)
}

// editList applies an add, remove, move or set to list.
func editList[T any](list []T, e PriorityEdit, parse func(string) (T, error), set func(*T, string, string) error) ([]T, error) {
	if e.Op != "add" && (e.Index < 0 || e.Index >= len(list)) {
		return nil, fmt.Errorf("no entry %d", e.Index+1)
	}
	switch e.Op {
	case "add":
		entry, err := parse(e.Value)
		if err != nil {
			return nil, err
		}
		return append(list, entry), nil
	case "remove":
		if len(list) == 1 {
			return nil, fmt.Errorf("the list needs at least one entry")
		}
		return slices.Delete(list, e.Index, e.Index+1), nil
	case "move":
		to := e.Index - 1
		if e.Value == "down" {
			to = e.Index + 1
		}
		if to < 0 || to >= len(list) {
			return nil, fmt.Errorf("cannot move entry %d %s", e.Index+1, e.Value)
		}
		list[e.Index], list[to] = list[to], list[e.Index]
		return list, nil
	case "set":
		if set == nil {
			return nil, fmt.Errorf("entries in this list cannot be edited")
		}
		if err := set(&list[e.Index], e.Field, e.Value); err != nil {
			return nil, err
		}
		return list, nil
	}
	return nil, fmt.Errorf("unknown edit %q", e.Op)
}

func parseEntry[T any](value string) (T, error) {
	var entry T
	d := json.NewDecoder(strings.NewReader(value))
	d.DisallowUnknownFields()
	if err := d.Decode(&entry); err != nil {
		return entry, fmt.Errorf("invalid entry: %w", err)
	}
	return entry, nil
}

func setCandidateField(c *Candidate, field, value string) error {
	switch field {
	case "pattern":
		c.Pattern = value
	case "driver":
		c.Driver = value
	default:
		return fmt.Errorf("unknown field %q", field)
	}
	return nil
}

func setInterfaceField(a *ASIOInterface, field, value string) error {
	switch field {
	case "asio_pattern":
		a.ASIOPattern = value
	case "presence_pattern":
		a.PresencePattern = value
	case "desk", "lav":
		channel, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return fmt.Errorf("%s channel must be a number", field)
		}
		if field == "desk" {
			a.Inputs[0] = channel
		} else {
			a.Inputs[1] = channel
		}
	default:
		return fmt.Errorf("unknown field %q", field)
	}
	return nil
}

func unmarshalList(raw json.RawMessage, list any) error {
	if raw == nil {
		return nil
	}
	return json.Unmarshal(raw, list)
}

func setKey(m map[string]json.RawMessage, key string, value any) error {
	b, err := encode(value, "")
	if err != nil {
		return err
	}
	m[key] = b
	return nil
}

// encode marshals value without HTML escaping, keeping patterns readable in
// the saved file.
func encode(value any, indent string) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", indent)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}
