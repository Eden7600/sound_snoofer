package nowplaying

import (
	"fmt"
	"time"
)

// browserSession is one tab or frame as the extension reports it.
type browserSession struct {
	ID         string  `json:"id"`       // "tab:frame".
	Instance   string  `json:"instance"` // Content-port lifetime within this connection.
	Tab        int     `json:"tab"`
	Site       string  `json:"site"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Album      string  `json:"album"`
	Art        string  `json:"art,omitempty"` // Base64 PNG, sent when ArtKey changes.
	ArtKey     string  `json:"artKey"`
	State      string  `json:"state"` // playing or paused.
	PositionMs int64   `json:"positionMs"`
	DurationMs int64   `json:"durationMs"`
	UpdatedMs  int64   `json:"updatedMs"`
	Rate       float64 `json:"rate"`
	CanNext    bool    `json:"canNext"`
	CanPrev    bool    `json:"canPrev"`
	CanSeek    bool    `json:"canSeek"`
	Muted      bool    `json:"muted"`
}

// browserUpdate replaces everything known about one browser. A disconnected
// update removes the browser.
type browserUpdate struct {
	Browser    string
	Version    string
	Connected  bool
	Generation uint64
	Sessions   []browserSession
}

// browserCommand is sent to the extension.
type browserCommand struct {
	Type       string `json:"type"` // "command".
	ID         string `json:"id"`
	Instance   string `json:"instance"`
	Generation uint64 `json:"-"`     // Captured bridge connection, never resolved by browser name alone.
	Op         string `json:"op"`    // play, pause, toggle, next, prev, seek or mute.
	Value      int64  `json:"value"` // Explicit 0/1 for toggle/mute, milliseconds for seek.
}

// maxBrowserSessions bounds what one browser may report.
const maxBrowserSessions = 64

func validBrowserSessions(sessions []browserSession) error {
	if len(sessions) > maxBrowserSessions {
		return fmt.Errorf("browser reported %d sessions; at most %d", len(sessions), maxBrowserSessions)
	}
	for _, s := range sessions {
		if s.ID == "" || len(s.ID) > 64 || s.Instance == "" || len(s.Instance) > 64 || len(s.Art) > 128*1024 {
			return fmt.Errorf("invalid browser session")
		}
	}
	return nil
}

// fromBrowser converts a tab; art is the cached thumbnail for its ArtKey.
func fromBrowser(browser string, s browserSession, art string) session {
	status := "Paused"
	if s.State == "playing" {
		status = "Playing"
	}
	title := s.Title
	out := session{Key: browser + ":" + s.ID, Source: browser, ID: s.ID, Instance: s.Instance, App: browser + " · " + s.Site, Title: title, Artist: s.Artist, Album: s.Album, Art: art,
		Status: status, PositionMs: s.PositionMs, DurationMs: s.DurationMs, Rate: s.Rate,
		CanToggle: true, CanPlay: true, CanPause: true, CanNext: s.CanNext, CanPrev: s.CanPrev, CanSeek: s.CanSeek && s.DurationMs > 0, CanMute: true, Muted: s.Muted}
	if s.Site == "" {
		out.App = browser
	}
	if s.UpdatedMs > 0 {
		out.Updated = time.UnixMilli(s.UpdatedMs)
	}
	return out
}
