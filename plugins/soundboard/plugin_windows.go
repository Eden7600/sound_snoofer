//go:build windows && amd64

package soundboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"sound-snoofer/internal/voicemeeter"
	"sound-snoofer/plugins/audio"
	"sound-snoofer/snoofer"
)

type Settings struct {
	Folder     string `json:"folder"`
	Renderer   string `json:"renderer"`
	Microphone bool   `json:"microphone"`
	Monitor    bool   `json:"monitor"`
	Overlap    bool   `json:"overlap,omitempty"` // Allow up to maxVoices simultaneous clips.
}
type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}
type command struct {
	clip          *clip
	toggleOverlap bool
}

// Plugin returns inert metadata; disabled plugins allocate no playback resources.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "soundboard", Label: "Soundboard", Requires: []string{"audio"}, Validate: validate, Defaults: snoofer.MarshalSettings(Settings{Folder: "Soundboard", Renderer: "DirectSound: Voicemeeter VAIO3 Input (VB-Audio Voicemeeter VAIO)", Microphone: true, Monitor: true}), Start: start}
}
func validate(raw json.RawMessage) error {
	var s Settings
	if err := snoofer.DecodeSettings(raw, &s); err != nil {
		return err
	}
	if strings.TrimSpace(s.Folder) == "" || strings.ContainsRune(s.Folder, 0) {
		return fmt.Errorf("soundboard folder is required")
	}
	if !strings.HasPrefix(s.Renderer, "DirectSound: ") || !strings.Contains(s.Renderer, "Voicemeeter VAIO3 Input") || strings.ContainsRune(s.Renderer, 0) {
		return fmt.Errorf("soundboard renderer must be an explicit DirectSound Voicemeeter VAIO3 Input")
	}
	if !s.Microphone && !s.Monitor {
		return fmt.Errorf("select at least one soundboard destination")
	}
	return nil
}
func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, deps map[string]snoofer.Instance) (snoofer.Instance, error) {
	if err := validate(raw); err != nil {
		return nil, err
	}
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return nil, err
	}
	// saved keeps the persisted form; settings.Folder is made absolute below.
	saved := settings
	audioPlugin, ok := deps["audio"].(*audio.Instance)
	if !ok {
		return nil, fmt.Errorf("audio dependency unavailable")
	}
	if !filepath.IsAbs(settings.Folder) {
		settings.Folder = filepath.Join(filepath.Dir(s.Path), settings.Folder)
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	routes := &audio.SoundboardRoutes{Microphone: settings.Microphone, Monitor: settings.Monitor}
	if err := audioPlugin.SetSoundboardRoutes(routes); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(i.done)
		defer s.Controls.Remove("soundboard")
		defer func() { i.err = errors.Join(i.err, audioPlugin.SetSoundboardRoutes(nil)) }()
		cacheFolder := filepath.Join(filepath.Dir(s.Path), "soundboard-cache", "peak-v1")
		library := filepath.Join(filepath.Dir(exe), "snoofer-soundboard.dll")
		prepare := func(ctx context.Context, c clip) (string, error) {
			return prepareClip(ctx, c, cacheFolder, func(ctx context.Context, source, destination string) error {
				return voicemeeter.NormalizeClip(ctx, library, source, destination)
			})
		}
		pending := &preparation{}
		defer func() { i.err = errors.Join(i.err, pending.close()) }()
		voices := newVoicePool(func() (clipPlayer, error) {
			return voicemeeter.OpenClipPlayer(library, settings.Renderer)
		})
		defer func() { i.err = errors.Join(i.err, voices.close()) }()
		commands := make(chan command, 1)
		clips, scanErr := catalogue(settings.Folder)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		scan := time.NewTicker(5 * time.Second)
		defer scan.Stop()
		failed, diagnostic := "", ""
		var link snoofer.ConnectionTracker // Native playback health, apart from clip feedback.
		lastClip := ""
		publish := func() {
			ready := audioPlugin.SoundboardReady()
			note := diagnostic
			if scanErr != nil {
				note = scanErr.Error()
			} else if ready != nil {
				note = ready.Error()
			}
			if note == "" {
				for _, c := range clips {
					if c.ArtworkError != "" {
						note = c.ArtworkError
						break
					}
				}
			}
			if !s.Live {
				note = "Preview"
			}
			controls := []snoofer.Control{
				{ID: "soundboard.status", Label: "Soundboard", Group: "Soundboard", Kind: "status", Value: note, Available: true},
				{ID: "soundboard.stop", Label: "Stop soundboard", ShortLabel: "Stop", Group: "Soundboard", Kind: "command", Icon: "soundboard-stop", Operations: []string{"press"}, Available: s.Live},
				{ID: "soundboard.overlap", Label: "Overlap clips", ShortLabel: "Overlap", Group: "Soundboard", Kind: "toggle", Icon: "soundboard-overlap", Value: onOff(settings.Overlap), Operations: []string{"press"}, Available: true},
			}
			volume, volumeHandler := audioPlugin.SoundboardVolume()
			controls = append(controls, volume, playbackReport(&link, s.Live, ready, voices.loaded(), settings.Renderer, lastClip, time.Now()))
			byID := map[string]clip{}
			for _, c := range clips {
				byID[c.ID] = c
				value, status := "Ready", ""
				if voices.playing(c.ID) {
					value = "Playing"
				}
				if pending.requested != nil && c.ID == pending.requested.ID {
					status = "Pending"
				}
				if c.ID == failed {
					status = diagnostic
				}
				controls = append(controls, snoofer.Control{Artwork: c.Artwork, ID: c.ID, Label: c.Label, Group: "Soundboard", Kind: "command", Icon: "soundboard-play", Value: value, Status: status, Operations: []string{"press"}, Available: s.Live && ready == nil && scanErr == nil})
			}
			if err := s.Controls.Publish("soundboard", controls, func(ctx context.Context, r snoofer.Request) error {
				if r.ID == volume.ID {
					return volumeHandler(ctx, r)
				}
				cmd := command{toggleOverlap: r.ID == "soundboard.overlap"}
				if r.ID != "soundboard.stop" && !cmd.toggleOverlap {
					c, ok := byID[r.ID]
					if !ok {
						return fmt.Errorf("clip unavailable")
					}
					cmd.clip = &c
				}
				select {
				case commands <- cmd:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				default:
					return fmt.Errorf("soundboard busy")
				}
			}); err != nil && runCtx.Err() == nil {
				diagnostic = err.Error()
			}
		}
		publish()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-scan.C:
				clips, scanErr = catalogue(settings.Folder)
				if scanErr == nil && s.Live {
					if err := pruneNormalized(cacheFolder, clips, voices.paths()); err != nil {
						diagnostic = err.Error()
					}
				}
			case <-ticker.C:
				if voices.active() {
					err := voices.poll()
					if err != nil {
						link.Fail(err.Error(), time.Now())
					} else if voices.active() {
						link.Activity(time.Now())
					}
					if err == nil {
						err = audioPlugin.SoundboardReady()
					}
					if err != nil {
						diagnostic = errors.Join(err, voices.stopAll()).Error()
					}
				}
			case result := <-pending.done:
				if pending.finish(result) {
					err := result.err
					if err == nil {
						err = runCtx.Err()
					}
					if err == nil {
						err = audioPlugin.SoundboardReady()
					}
					if err == nil {
						err = result.clip.unchanged()
					}
					if err == nil {
						// Opening or starting a native graph failed: a playback fault.
						if err = voices.play(result.clip.ID, result.path, settings.Overlap); err != nil {
							link.Fail(err.Error(), time.Now())
						} else {
							link.Activity(time.Now())
							lastClip = result.clip.Label
						}
					}
					if err != nil {
						failed = result.clip.ID
						diagnostic = err.Error()
					} else {
						failed = ""
					}
				}
			case cmd := <-commands:
				if runCtx.Err() != nil {
					return
				}
				diagnostic = ""
				failed = ""
				if cmd.toggleOverlap {
					next := saved
					next.Overlap = !saved.Overlap
					data, err := json.Marshal(next)
					if err == nil {
						err = s.SaveSettings("soundboard", raw, data)
					}
					if err != nil {
						diagnostic = err.Error()
					} else {
						raw, saved = data, next
						settings.Overlap = next.Overlap
					}
					publish()
					continue
				}
				var err error
				// A clip press with overlap keeps current voices; Stop or a
				// replacing press silences everything first.
				if cmd.clip == nil || !settings.Overlap {
					err = voices.stopAll()
				}
				pending.replace(nil)
				if cmd.clip != nil && err == nil {
					failed = cmd.clip.ID
					if !s.Live {
						err = fmt.Errorf("preview does not play clips")
					} else {
						err = audioPlugin.SoundboardReady()
					}
					if err == nil {
						err = cmd.clip.unchanged()
					}
					if err == nil {
						pending.replace(cmd.clip)
						failed = ""
					}
				}
				if err != nil {
					diagnostic = err.Error()
				}
			}
			pending.start(runCtx, prepare)
			publish()
		}
	}()
	return i, nil
}
func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return i.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
