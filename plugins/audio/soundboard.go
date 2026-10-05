package audio

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/controller"
	"sound-snoofer/snoofer"
)

// SoundboardRoutes selects destinations for the dedicated VAIO3 input.
type SoundboardRoutes = config.SoundboardRoutes

func validateSoundboard(reserved bool, c config.Config) error {
	if !reserved {
		return nil
	}
	if c.Studio == nil || c.Studio.Voice == nil {
		return fmt.Errorf("soundboard_input requires the studio voice profile")
	}
	if slices.Contains(c.Studio.PlaybackSources, "virtual:3") {
		return fmt.Errorf("VAIO3 is reserved for soundboard; remove virtual:3 from playback_sources")
	}
	return nil
}

// SetSoundboardRoutes lets the dependent plugin request routes without native writes.
// A nil policy clears reserved sends on the next reconciliation.
func (i *Instance) SetSoundboardRoutes(routes *SoundboardRoutes) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.soundboardReserved {
		return fmt.Errorf("enable audio.settings.soundboard_input to reserve VAIO3")
	}
	i.soundboard = nil
	if routes != nil {
		copy := *routes
		i.soundboard = &copy
	}
	select {
	case i.actions <- control.Refresh:
	default:
	}
	return nil
}
func (i *Instance) soundboardSnapshot() *config.SoundboardRoutes {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.soundboard
}

// SoundboardReady verifies observed sends, never just a successful setter.
func (i *Instance) SoundboardReady() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return soundboardReady(i.state, i.soundboard)
}
func soundboardReady(s control.State, routes *config.SoundboardRoutes) error {
	if routes == nil || !s.Live || !s.Connected || s.RecoveryPending || s.Error != "" || s.StateError != "" || time.Since(s.ObservedAt) > 3*time.Second {
		return fmt.Errorf("audio routing unavailable")
	}
	if s.Snapshot.Edition != 3 || s.Plan == nil || s.Plan.Topology == nil {
		return fmt.Errorf("soundboard requires Voicemeeter Potato")
	}
	target := s.Plan.Topology.PlaybackTarget
	if routes.Monitor && target == "" {
		return fmt.Errorf("playback output unavailable")
	}
	if routes.Monitor {
		for _, op := range s.Plan.Topology.Operations {
			if op.Target == target && op.Device != nil && s.Snapshot.Assignments[target] != op.Device.Name {
				return fmt.Errorf("playback device assignment pending")
			}
		}
	}
	desired := map[string]float32{"Strip[7].B2": 0, "Strip[7].B3": 0}
	if routes.Microphone {
		desired["Strip[7].B3"] = 1
	}
	for bus := 1; bus <= 5; bus++ {
		key := fmt.Sprintf("Strip[7].A%d", bus)
		desired[key] = 0
		if routes.Monitor && target == fmt.Sprintf("A%d", bus) {
			desired[key] = 1
		}
	}
	for key, want := range desired {
		if got, ok := s.Snapshot.Numbers[key]; !ok || got != want {
			return fmt.Errorf("soundboard routing pending: %s", key)
		}
	}
	return nil
}

// SoundboardVolume exposes the dedicated input through the existing audio actor.
// Its handler only queues work; the caller may safely use it under the controls lock.
func (i *Instance) SoundboardVolume() (snoofer.Control, func(context.Context, snoofer.Request) error) {
	i.mu.Lock()
	s := i.state
	ready := i.soundboardReserved && soundboardReady(s, i.soundboard) == nil
	i.mu.Unlock()
	value, known := s.Snapshot.Numbers["Strip[7].Gain"]
	known = known && !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
	display := "N/A"
	if known {
		display = fmt.Sprintf("%.1f dB", value)
	}
	c := snoofer.Control{ID: "soundboard.volume", Label: "Soundboard volume", ShortLabel: "Volume", Group: "Soundboard", Kind: "numeric", Value: display, Operations: []string{"adjust", "press"}, Available: ready && known, Epoch: s.Revision}
	action := control.Action{Kind: control.Gain, Target: "soundboard", Identity: controller.GainIdentity(s.Plan, s.Snapshot, "soundboard"), Revision: s.Revision, Origin: "soundboard"}
	return c, func(ctx context.Context, r snoofer.Request) error {
		if !c.Available {
			return fmt.Errorf("soundboard gain unavailable")
		}
		if r.Operation != "adjust" && r.Operation != "press" {
			return fmt.Errorf("unsupported volume operation")
		}
		a := action
		a.ResetGain = r.Operation == "press"
		a.Delta = float32(r.Delta)
		select {
		case i.actions <- a:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("audio command queue full")
		}
	}
}
