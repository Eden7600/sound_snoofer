// Package mediasessions reads and controls Windows media sessions (the
// system media transport controls) through the snoofer-media companion.
package mediasessions

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Session is one app's media session as Windows reports it.
type Session struct {
	ID         string  `json:"id"`  // AppUserModelID, plus "#n" when an app has several.
	App        string  `json:"app"` // AppUserModelID.
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Album      string  `json:"album"`
	Status     string  `json:"status"` // playing, paused, stopped, changing, closed or opened.
	PositionMs int64   `json:"positionMs"`
	DurationMs int64   `json:"durationMs"`
	UpdatedMs  int64   `json:"updatedMs"` // Unix ms when the position was sampled; 0 if unknown.
	Rate       float64 `json:"rate"`
	CanPlay    bool    `json:"canPlay"`
	CanPause   bool    `json:"canPause"`
	CanNext    bool    `json:"canNext"`
	CanPrev    bool    `json:"canPrev"`
	CanSeek    bool    `json:"canSeek"`
	Current    bool    `json:"current"` // The session media keys control.
	ArtKey     string  `json:"artKey"`  // Changes with the track; empty without artwork.
}

// Op is a session command.
type Op int

const (
	Play Op = iota + 1
	Pause
	Toggle
	Next
	Previous
	Seek // Value is the absolute position in milliseconds.
)

// ErrDeclined reports that the player refused a command it received.
var ErrDeclined = errors.New("player declined the command")

func parseSnapshot(data []byte) ([]Session, error) {
	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("media sessions snapshot: %w", err)
	}
	return sessions, nil
}
