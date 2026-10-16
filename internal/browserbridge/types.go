// Package browserbridge owns the local extension transport shared by Media and Meet.
package browserbridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrConnectionChanged = errors.New("browser connection changed; try again")
	ErrUnavailable       = errors.New("browser feature unavailable")
	ErrBusy              = errors.New("browser commands busy")
)

// Status describes the actual shared listener, independently of either provider.
type Status struct {
	Port           int
	Error, Refused string
}

// MediaSession retains the protocol-2 Media report, including optional ratings.
type MediaSession struct {
	ID             string  `json:"id"`
	Instance       string  `json:"instance"`
	Tab            int     `json:"tab"`
	Site           string  `json:"site"`
	Title          string  `json:"title"`
	Artist         string  `json:"artist"`
	Album          string  `json:"album"`
	Art            string  `json:"art,omitempty"`
	ArtKey         string  `json:"artKey"`
	State          string  `json:"state"`
	PositionMs     int64   `json:"positionMs"`
	DurationMs     int64   `json:"durationMs"`
	UpdatedMs      int64   `json:"updatedMs"`
	Rate           float64 `json:"rate"`
	CanNext        bool    `json:"canNext"`
	CanPrev        bool    `json:"canPrev"`
	CanSeek        bool    `json:"canSeek"`
	Muted          bool    `json:"muted"`
	Track          string  `json:"track,omitempty"`
	Rating         string  `json:"rating,omitempty"`
	CanRate        bool    `json:"canRate,omitempty"`
	ReloadRequired bool    `json:"reloadRequired,omitempty"`
}

// MediaUpdate replaces the browser's complete Media snapshot.
type MediaUpdate struct {
	Browser, Version string
	Connected        bool
	Generation       uint64
	Sessions         []MediaSession
}

// MediaCommand preserves the existing submitted-only Media wire contract.
type MediaCommand struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Instance   string `json:"instance"`
	Generation uint64 `json:"-"`
	Op         string `json:"op"`
	Value      int64  `json:"value"`
	Track      string `json:"track,omitempty"`
	Rating     string `json:"rating,omitempty"`
}

// ValidateMediaSessions preserves the bounds of the existing browser transport.
func ValidateMediaSessions(sessions []MediaSession) error {
	if len(sessions) > 64 {
		return fmt.Errorf("browser reported %d sessions; at most 64", len(sessions))
	}
	for _, s := range sessions {
		if s.ID == "" || len(s.ID) > 64 || s.Instance == "" || len(s.Instance) > 64 || len(s.Art) > 128*1024 {
			return errors.New("invalid browser session")
		}
		known := s.Rating == "none" || s.Rating == "like" || s.Rating == "dislike"
		if len(s.Track) > 128 || s.Rating != "" && s.Rating != "unknown" && !known || s.CanRate && (s.Track == "" || !known) {
			return errors.New("invalid browser rating")
		}
	}
	return nil
}

// MeetControl separates unknown state from a known false value.
type MeetControl struct {
	Identity  uint64 `json:"identity,string,omitempty"`
	Value     *bool  `json:"value"`
	Revision  uint64 `json:"revision,string"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// MeetCall identifies one observed document and joined-call lifetime.
type MeetCall struct {
	Tab      int64        `json:"tab"`
	Instance string       `json:"instance"`
	Sample   uint64       `json:"sample,string"`
	Call     string       `json:"call"`
	State    string       `json:"state"`
	Mic      MeetControl  `json:"mic"`
	Camera   MeetControl  `json:"camera"`
	Share    *MeetControl `json:"share,omitempty"`
	Leave    *MeetControl `json:"leave,omitempty"`
}

// MeetUpdate is immutable after delivery. Received belongs to the local owner.
type MeetUpdate struct {
	Browser, Version     string
	Connected, Supported bool
	Generation           uint64
	Received             time.Time
	Calls                []MeetCall
}

// MeetCommand captures explicit intent; the owner adds its unique wire ID.
type MeetCommand struct {
	Generation uint64 `json:"-"`
	Tab        int64  `json:"tab"`
	Instance   string `json:"instance"`
	Call       string `json:"call"`
	Control    string `json:"control"`
	Revision   uint64 `json:"revision,string"`
	Value      bool   `json:"value"`
}

// MeetResult settles exactly one accepted command without changing observations.
type MeetResult struct {
	ID       string `json:"id"`
	Tab      int64  `json:"tab"`
	Instance string `json:"instance"`
	Call     string `json:"call"`
	Control  string `json:"control"`
	Revision uint64 `json:"revision,string"`
	Value    bool   `json:"value"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

func counter(s string) bool {
	if s == "" || len(s) > 20 || s[0] == '0' {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && n > 0
}

func validCommand(c MeetCommand) bool {
	return c.Generation > 0 && c.Tab >= 0 && c.Tab <= 9007199254740991 && counter(c.Instance) && counter(c.Call) && c.Revision > 0 && (c.Control == "mic" || c.Control == "camera" || c.Control == "share" || c.Control == "leave" && c.Value)
}

// Presence is checked before ordinary decoding so omitted false/null fields cannot
// silently become known false controls or successful false-valued results.
func required(data []byte, fields ...string) error {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	for _, field := range fields {
		if len(values[field]) == 0 {
			return fmt.Errorf("missing %s", field)
		}
	}
	return nil
}

func (c *MeetControl) UnmarshalJSON(data []byte) error {
	if err := required(data, "value", "revision", "available", "reason"); err != nil {
		return err
	}
	type plain MeetControl
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var presence struct {
		Identity  json.RawMessage `json:"identity"`
		Available *bool           `json:"available"`
		Revision  string          `json:"revision"`
		Reason    *string         `json:"reason"`
	}
	if err := json.Unmarshal(data, &presence); err != nil {
		return err
	}
	if presence.Available == nil || presence.Reason == nil || !counter(presence.Revision) {
		return errors.New("invalid Meet control")
	}
	if len(presence.Identity) > 0 {
		var identity string
		if err := json.Unmarshal(presence.Identity, &identity); err != nil || identity != "0" && !counter(identity) {
			return errors.New("invalid Meet control identity")
		}
	}
	if value.Available {
		if value.Value == nil || value.Reason != "" {
			return errors.New("invalid available Meet control")
		}
	} else if value.Reason != "unknown" && value.Reason != "unavailable" && value.Reason != "not-in-call" && value.Reason != "reload-required" {
		return errors.New("invalid Meet control reason")
	}
	*c = MeetControl(value)
	return nil
}

func decodeCalls(data []byte) ([]MeetCall, error) {
	if len(data) > 64*1024 {
		return nil, errors.New("Meet state exceeds 64 KiB")
	}
	var envelope struct {
		Calls []json.RawMessage `json:"calls"`
	}
	if err := required(data, "calls"); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if envelope.Calls == nil {
		return nil, errors.New("Meet calls must be an array")
	}
	if len(envelope.Calls) > 64 {
		return nil, errors.New("Meet reports at most 64 tabs")
	}
	seen := map[int64]bool{}
	result := make([]MeetCall, 0, len(envelope.Calls))
	for _, raw := range envelope.Calls {
		if err := required(raw, "tab", "instance", "sample", "call", "state", "mic", "camera"); err != nil {
			return nil, err
		}
		var c MeetCall
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		var tab struct {
			Tab    *int64  `json:"tab"`
			Sample string  `json:"sample"`
			Call   *string `json:"call"`
		}
		_ = json.Unmarshal(raw, &tab) // Full decoding above already checked the number type.
		if tab.Tab == nil || c.Tab < 0 || c.Tab > 9007199254740991 || seen[c.Tab] || !counter(c.Instance) || !counter(tab.Sample) || tab.Call == nil || c.Mic.Revision == 0 || c.Camera.Revision == 0 {
			return nil, errors.New("invalid Meet tab")
		}
		var optional map[string]json.RawMessage
		_ = json.Unmarshal(raw, &optional) // Full decoding above checked the object.
		for _, name := range []string{"share", "leave"} {
			if value, present := optional[name]; present && string(value) == "null" {
				return nil, errors.New("invalid optional Meet control")
			}
		}
		if c.State == "joined" {
			if !counter(c.Call) {
				return nil, errors.New("invalid Meet call")
			}
		} else if (c.State != "prejoin" && c.State != "unknown") || c.Call != "" || c.Mic.Available || c.Camera.Available || c.Share != nil && c.Share.Available || c.Leave != nil && c.Leave.Available {
			return nil, errors.New("invalid Meet call state")
		}
		seen[c.Tab] = true
		result = append(result, c)
	}
	return result, nil
}

func decodeResult(data []byte) (MeetResult, error) {
	var r MeetResult
	if len(data) > 2048 {
		return r, errors.New("Meet result exceeds 2 KiB")
	}
	if err := required(data, "id", "tab", "instance", "call", "control", "revision", "value", "status", "reason"); err != nil {
		return r, err
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, err
	}
	var presence struct {
		Tab      *int64  `json:"tab"`
		Value    *bool   `json:"value"`
		Revision string  `json:"revision"`
		Reason   *string `json:"reason"`
	}
	if err := json.Unmarshal(data, &presence); err != nil {
		return r, err
	}
	if presence.Tab == nil || presence.Value == nil || presence.Reason == nil || !counter(presence.Revision) || !counter(r.ID) || !validCommand(MeetCommand{Generation: 1, Tab: r.Tab, Instance: r.Instance, Call: r.Call, Control: r.Control, Revision: r.Revision, Value: r.Value}) {
		return r, errors.New("invalid Meet result")
	}
	valid := r.Status == "observed" && r.Reason == "" || r.Status == "submitted" && r.Reason == "choose-source" && r.Control == "share" && r.Value || r.Status == "rejected" && (r.Reason == "stale" || r.Reason == "unavailable" || r.Reason == "busy" || r.Reason == "disabled") || r.Status == "unknown" && (r.Reason == "not-confirmed" || r.Reason == "disconnected" || r.Reason == "stopped")
	if !valid {
		return r, errors.New("invalid Meet result status")
	}
	return r, nil
}
