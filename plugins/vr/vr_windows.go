//go:build windows

// Package vr owns SteamVR detection and depends directly on audio's policy API.
package vr

import (
	"context"
	"encoding/json"
	"time"

	"sound-snoofer/internal/process"
	"sound-snoofer/plugins/audio"
	"sound-snoofer/snoofer"
)

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Plugin returns the VR policy provider.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "vr", Validate: func(raw json.RawMessage) error {
		var p audio.VRPolicy
		if err := snoofer.DecodeSettings(raw, &p); err != nil {
			return err
		}
		return p.Validate()
	}, Label: "VR", Defaults: json.RawMessage(`{"devices":{"input":4,"headsets":[]},"microphones":["normal"],"playback":[{"driver":"normal","pattern":""}],"choices":{"source":"auto","mode":"direct","monitor":"off"}}`), Requires: []string{"audio"}, Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, deps map[string]snoofer.Instance) (snoofer.Instance, error) {
		var policy audio.VRPolicy
		if err := snoofer.DecodeSettings(raw, &policy); err != nil {
			return nil, err
		}
		if err := policy.Validate(); err != nil {
			return nil, err
		}
		target := deps["audio"].(*audio.Instance)
		runCtx, cancel := context.WithCancel(ctx)
		i := &instance{cancel: cancel, done: make(chan struct{})}
		observe := func() error {
			if runCtx.Err() != nil {
				return runCtx.Err()
			}
			names, err := process.Names()
			policy.Known = err == nil
			if err == nil {
				policy.Running = process.Contains(names, "vrserver.exe")
			}
			if runCtx.Err() != nil {
				return runCtx.Err()
			}
			if err := target.SetVRPolicy(policy); err != nil {
				return err
			}
			value := "Stopped"
			if policy.Running {
				value = "Running"
			}
			if !policy.Known {
				value = "Unknown — retaining last profile"
			}
			_ = s.Controls.Publish("vr", []snoofer.Control{{ID: "vr.status", Label: "SteamVR", Group: "VR", Kind: "status", Value: value, Available: true}}, nil)
			return nil
		}
		if err := observe(); err != nil {
			cancel()
			return nil, err
		}
		go func() {
			defer close(i.done)
			defer s.Controls.Remove("vr")
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-runCtx.Done():
					return
				case <-ticker.C:
					if err := observe(); err != nil {
						_ = s.Controls.Publish("vr", []snoofer.Control{{ID: "vr.status", Label: "SteamVR", Group: "VR", Kind: "status", Value: err.Error()}}, nil)
					}
				}
			}
		}()
		return i, nil
	}}
}
