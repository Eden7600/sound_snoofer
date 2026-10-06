package nowplaying

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"sound-snoofer/internal/mediasessions"
)

// session is a media session from either source, in one shape.
type session struct {
	Key        string // Source and source ID; stable while the session lives.
	Source     string // "windows" or the browser name.
	ID         string // The source's own ID.
	App        string // Player or site, for display.
	Title      string
	Artist     string
	Album      string
	Art        string // 64 px base64 PNG, or empty.
	Status     string // Playing, Paused or Stopped.
	PositionMs int64
	DurationMs int64
	Updated    time.Time // When PositionMs was sampled; zero if unknown.
	Rate       float64
	CanToggle  bool
	CanNext    bool
	CanPrev    bool
	CanSeek    bool
	CanMute    bool // Browser tabs only.
	Muted      bool
}

func (s session) playing() bool { return s.Status == "Playing" }

// controlID is stable for the session's life.
func controlID(key string) string {
	return fmt.Sprintf("nowplaying.s-%x", sha256.Sum256([]byte(key)))[:len("nowplaying.s-")+12]
}

// position interpolates the playback position at now, clamped to the length.
func (s session) position(now time.Time) int64 {
	p := s.PositionMs
	if s.playing() && !s.Updated.IsZero() {
		rate := s.Rate
		if rate <= 0 {
			rate = 1
		}
		p += int64(float64(now.Sub(s.Updated).Milliseconds()) * rate)
	}
	if s.DurationMs > 0 {
		p = min(p, s.DurationMs)
	}
	return max(0, p)
}

// clock formats milliseconds as m:ss, or h:mm:ss from an hour.
func clock(ms int64) string {
	seconds := max(0, ms/1000)
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// progress is the dial value: position and length, or position alone.
func (s session) progress(now time.Time) string {
	if s.DurationMs <= 0 {
		return clock(s.position(now))
	}
	return clock(s.position(now)) + " / " + clock(s.DurationMs)
}

// appName makes an AppUserModelID readable: "Spotify.exe" is Spotify and
// "Microsoft.ZuneMusic_8wekyb3d8bbwe!Microsoft.ZuneMusic" is Media Player.
func appName(aumid string) string {
	name := aumid
	if _, after, ok := strings.Cut(name, "!"); ok {
		name = after
	}
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".exe"), ".EXE")
	switch strings.ToLower(name) {
	case "microsoft.zunemusic":
		return "Media Player"
	case "msedge":
		return "Edge"
	}
	if i := strings.LastIndex(name, "."); i >= 0 && i < len(name)-1 && strings.Contains(name[:i], ".") {
		name = name[i+1:]
	}
	return strings.TrimPrefix(name, "Microsoft.")
}

func windowsStatus(status string) string {
	switch status {
	case "playing":
		return "Playing"
	case "paused":
		return "Paused"
	}
	return "Stopped"
}

// fromWindows converts a Windows session; art comes from the caller's cache.
func fromWindows(s mediasessions.Session, art string) session {
	out := session{Key: "windows:" + s.ID, Source: "windows", ID: s.ID, App: appName(s.App), Title: s.Title, Artist: s.Artist, Album: s.Album, Art: art,
		Status: windowsStatus(s.Status), PositionMs: s.PositionMs, DurationMs: s.DurationMs, Rate: s.Rate,
		CanToggle: s.CanPlay || s.CanPause, CanNext: s.CanNext, CanPrev: s.CanPrev, CanSeek: s.CanSeek && s.DurationMs > 0}
	if s.UpdatedMs > 0 {
		out.Updated = time.UnixMilli(s.UpdatedMs)
	}
	return out
}

// browserApps maps a browser's name to the AppUserModelID its single Windows
// session uses; that session is hidden while the browser's extension is
// connected, because the extension reports each tab instead.
var browserApps = map[string]string{"brave": "brave", "chrome": "chrome", "edge": "msedge"}

// shadowed reports whether a Windows session belongs to a connected browser.
func shadowed(app string, connected map[string]bool) bool {
	app = strings.ToLower(app)
	for browser := range connected {
		if id, ok := browserApps[strings.ToLower(browser)]; ok && (app == id || strings.HasPrefix(app, id+".")) {
			return true
		}
	}
	return false
}
