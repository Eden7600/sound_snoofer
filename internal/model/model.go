package model

import (
	"fmt"
	"strconv"
	"strings"
)

type Device struct {
	Name      string `json:"name"`
	Direction string `json:"direction"`
	Driver    string `json:"driver"`
	ID        string `json:"hardware_id,omitempty"`
	Available bool   `json:"available"`
}

// CallbackStatus reports processing progress, never proof of audible output.
type CallbackStatus struct {
	Active                                     bool
	Buffers, Synced, Starting, Ending, Changes uint32
	Error                                      string
}

type Snapshot struct {
	Callback    *CallbackStatus    `json:"callback,omitempty"`
	SteamVR     *ProcessStatus     `json:"steamvr,omitempty"`
	Element     *ProcessStatus     `json:"element,omitempty"`
	Recorder    *RecorderSnapshot  `json:"recorder,omitempty"`
	Edition     int                `json:"edition"`
	Devices     []Device           `json:"devices"`
	Assignments map[string]string  `json:"assignments"`
	Numbers     map[string]float32 `json:"parameters,omitempty"`
}

func StripCount(edition int) int {
	if edition == 2 {
		return 5
	}
	if edition == 3 {
		return 8
	}
	return 0
}

type Slot struct {
	Direction string
	Index     int
}

// Slot names use one-based physical inputs (input:1) and hardware buses (A1).
func ParseSlot(s string) (Slot, error) {
	var direction, number string
	if strings.HasPrefix(s, "input:") {
		direction, number = "input", strings.TrimPrefix(s, "input:")
	} else if strings.HasPrefix(s, "A") {
		direction, number = "output", strings.TrimPrefix(s, "A")
	}
	n, err := strconv.Atoi(number)
	if err != nil || n < 1 || n > 5 || strconv.Itoa(n) != number {
		return Slot{}, fmt.Errorf("invalid hardware slot %q (use input:1..5 or A1..5)", s)
	}
	return Slot{direction, n - 1}, nil
}

func (s Slot) Parameter(suffix string) string {
	prefix := "Strip"
	if s.Direction == "output" {
		prefix = "Bus"
	}
	return fmt.Sprintf("%s[%d].device.%s", prefix, s.Index, suffix)
}

func Limits(edition int) (int, error) {
	switch edition {
	case 2:
		return 3, nil
	case 3:
		return 5, nil
	}
	return 0, fmt.Errorf("unsupported Voicemeeter edition %d: routing requires Banana (2) or Potato (3)", edition)
}

func Slots(edition int) []string {
	n, err := Limits(edition)
	if err != nil {
		return nil
	}
	out := make([]string, 0, n*2)
	for i := 1; i <= n; i++ {
		out = append(out, fmt.Sprintf("input:%d", i), fmt.Sprintf("A%d", i))
	}
	return out
}
