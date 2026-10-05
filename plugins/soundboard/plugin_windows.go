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
}
type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}
type command struct{ clip *clip }

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
		var player *voicemeeter.ClipPlayer
		defer func() {
			if player != nil {
				i.err = errors.Join(i.err, player.Close())
			}
		}()
		cacheFolder := filepath.Join(filepath.Dir(s.Path), "soundboard-cache", "peak-v1")
		library := filepath.Join(filepath.Dir(exe), "snoofer-soundboard.dll")
		prepare := func(ctx context.Context, c clip) (string, error) {
			return prepareClip(ctx, c, cacheFolder, func(ctx context.Context, source, destination string) error {
				return voicemeeter.NormalizeClip(ctx, library, source, destination)
			})
		}
		pending := &preparation{}
		defer func() { i.err = errors.Join(i.err, pending.close()) }()
		playingPath := ""
		commands := make(chan command, 1)
		clips, scanErr := catalogue(settings.Folder)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		scan := time.NewTicker(5 * time.Second)
		defer scan.Stop()
		current, failed, diagnostic := "", "", ""
		stop := func() error {
			current = ""
			playingPath = ""
			if player != nil {
				return player.Stop()
			}
			return nil
		}
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
			}
			volume, volumeHandler := audioPlugin.SoundboardVolume()
			controls = append(controls, volume)
			byID := map[string]clip{}
			for _, c := range clips {
				byID[c.ID] = c
				value, status := "Ready", ""
				if c.ID == current {
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
				cmd := command{}
				if r.ID != "soundboard.stop" {
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
					if err := pruneNormalized(cacheFolder, clips, playingPath); err != nil {
						diagnostic = err.Error()
					}
				}
			case <-ticker.C:
				if current != "" {
					active, err := player.Poll()
					if err == nil {
						err = audioPlugin.SoundboardReady()
					}
					if err != nil {
						failed = current
						diagnostic = errors.Join(err, stop()).Error()
					} else if !active {
						current = ""
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
					if err == nil && player == nil {
						player, err = voicemeeter.OpenClipPlayer(library, settings.Renderer)
					}
					if err == nil {
						err = player.Play(result.path, false)
					}
					if err != nil {
						failed = result.clip.ID
						diagnostic = err.Error()
					} else {
						current = result.clip.ID
						playingPath = result.path
						failed = ""
					}
				}
			case cmd := <-commands:
				if runCtx.Err() != nil {
					return
				}
				diagnostic = ""
				failed = ""
				err := stop()
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
