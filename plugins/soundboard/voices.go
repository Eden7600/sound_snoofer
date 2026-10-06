package soundboard

import (
	"errors"
	"time"

	"sound-snoofer/snoofer"
)

// maxVoices bounds simultaneous clips when overlap is on; a further clip
// stops the oldest voice.
const maxVoices = 8

// clipPlayer is one native playback graph. The pool reuses players across
// voices; all calls happen on the soundboard's locked worker thread.
type clipPlayer interface {
	Play(path string, muted bool) error
	Stop() error
	Poll() (bool, error)
	Close() error
}

type voice struct {
	player clipPlayer
	clip   string
	path   string
	ends   time.Time // Estimated; zero when the length is unknown.
}

// voicePool manages concurrent clips. Without overlap it behaves as a single
// voice: every new clip stops whatever is playing first.
type voicePool struct {
	open   func() (clipPlayer, error)
	max    int
	voices []voice // Playing, oldest first.
	idle   []clipPlayer
	opened int
}

func newVoicePool(open func() (clipPlayer, error)) *voicePool {
	return &voicePool{open: open, max: maxVoices}
}

// play starts a clip as a new voice, expected to end at ends (zero when
// unknown). On failure the player returns to the
// idle set and no voice is added.
func (p *voicePool) play(clip, path string, ends time.Time, overlap bool) error {
	if !overlap {
		if err := p.stopAll(); err != nil {
			return err
		}
	}
	if len(p.voices) >= p.max {
		oldest := p.voices[0]
		p.voices = p.voices[1:]
		p.idle = append(p.idle, oldest.player)
		if err := oldest.player.Stop(); err != nil {
			return err
		}
	}
	var player clipPlayer
	if n := len(p.idle); n > 0 {
		player, p.idle = p.idle[n-1], p.idle[:n-1]
	} else {
		var err error
		if player, err = p.open(); err != nil {
			return err
		}
		p.opened++
	}
	if err := player.Play(path, false); err != nil {
		p.idle = append(p.idle, player)
		return err
	}
	p.voices = append(p.voices, voice{player: player, clip: clip, path: path, ends: ends})
	return nil
}

// poll drops finished voices. A native polling error stops that voice and is
// returned after every voice has been checked.
func (p *voicePool) poll() error {
	var failures []error
	playing := p.voices[:0]
	for _, v := range p.voices {
		active, err := v.player.Poll()
		if err != nil {
			failures = append(failures, err, v.player.Stop())
		}
		if err == nil && active {
			playing = append(playing, v)
			continue
		}
		p.idle = append(p.idle, v.player)
	}
	p.voices = playing
	return errors.Join(failures...)
}

func (p *voicePool) stopAll() error {
	var failures []error
	for _, v := range p.voices {
		failures = append(failures, v.player.Stop())
		p.idle = append(p.idle, v.player)
	}
	p.voices = nil
	return errors.Join(failures...)
}

// close stops and releases every player.
func (p *voicePool) close() error {
	failures := []error{p.stopAll()}
	for _, player := range p.idle {
		failures = append(failures, player.Close())
	}
	p.idle = nil
	return errors.Join(failures...)
}

func (p *voicePool) playing(clip string) bool {
	for _, v := range p.voices {
		if v.clip == clip {
			return true
		}
	}
	return false
}

// timers lists playing clips with a known length, oldest first.
func (p *voicePool) timers() []snoofer.Timer {
	var out []snoofer.Timer
	for _, v := range p.voices {
		if !v.ends.IsZero() {
			out = append(out, snoofer.Timer{Control: v.clip, Ends: v.ends})
		}
	}
	return out
}

func (p *voicePool) active() bool { return len(p.voices) > 0 }

// loaded reports whether the native companion has been loaded at all.
func (p *voicePool) loaded() bool { return p.opened > 0 }

// paths lists files in use, which cache pruning must keep.
func (p *voicePool) paths() []string {
	out := make([]string, 0, len(p.voices))
	for _, v := range p.voices {
		out = append(out, v.path)
	}
	return out
}

func onOff(v bool) string {
	if v {
		return "On"
	}
	return "Off"
}
