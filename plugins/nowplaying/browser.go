package nowplaying

import (
	"time"

	"sound-snoofer/internal/browserbridge"
)

type browserSession = browserbridge.MediaSession
type browserUpdate = browserbridge.MediaUpdate
type browserCommand = browserbridge.MediaCommand

const maxBrowserSessions = 64

var (
	errBrowserChanged = browserbridge.ErrConnectionChanged
	errBrowserBusy    = browserbridge.ErrBusy
)

func validBrowserSessions(sessions []browserSession) error {
	return browserbridge.ValidateMediaSessions(sessions)
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
		CanToggle: !s.ReloadRequired, CanPlay: !s.ReloadRequired, CanPause: !s.ReloadRequired, CanNext: s.CanNext && !s.ReloadRequired, CanPrev: s.CanPrev && !s.ReloadRequired, CanSeek: s.CanSeek && s.DurationMs > 0 && !s.ReloadRequired, CanMute: true, Muted: s.Muted,
		Track: s.Track, Rating: s.Rating, CanRate: s.CanRate && !s.ReloadRequired, ReloadRequired: s.ReloadRequired}
	if s.Site == "" {
		out.App = browser
	}
	if s.UpdatedMs > 0 {
		out.Updated = time.UnixMilli(s.UpdatedMs)
	}
	return out
}
