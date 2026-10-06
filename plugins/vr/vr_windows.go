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
		// link is used by Start and then only by the polling goroutine, never concurrently.
		var link snoofer.ConnectionTracker
		observe := func() error {
			if runCtx.Err() != nil {
				return runCtx.Err()
			}
			names, err := process.Names()
			now := time.Now()
			if err != nil {
				link.Fail(err.Error(), now)
			}
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
			_ = s.Controls.Publish("vr", []snoofer.Control{
				{ID: "vr.status", Label: "SteamVR", Group: "VR", Kind: "status", Value: value, Available: true},
				steamVRReport(&link, policy, now),
			}, nil)
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
						now := time.Now()
						link.Fail(err.Error(), now)
						_ = s.Controls.Publish("vr", []snoofer.Control{
							{ID: "vr.status", Label: "SteamVR", Group: "VR", Kind: "status", Value: err.Error()},
							steamVRReport(&link, audio.VRPolicy{}, now),
						}, nil)
					}
				}
			}
		}()
		return i, nil
	}}
}

// steamVRReport describes vrserver.exe detection for the Third-party apps screen.
// An unknown observation keeps the previous headset profile but reports Unknown.
func steamVRReport(link *snoofer.ConnectionTracker, policy audio.VRPolicy, now time.Time) snoofer.Control {
	state, value, profile := snoofer.ConnectionUnknown, "Unknown", ""
	switch {
	case policy.Known && policy.Running:
		state, value, profile = snoofer.ConnectionConnected, "Running", "VR"
		link.Activity(now)
	case policy.Known:
		state, value, profile = snoofer.ConnectionDisconnected, "Not running", "Normal"
		link.Activity(now)
	}
	link.Observe(state, now)
	details := []snoofer.ConnectionDetail{{Label: "Interval", Value: "1s"}}
	if profile != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Audio profile", Value: profile})
	}
	return snoofer.Control{ID: "vr.app-steamvr", Label: "SteamVR", Group: "VR", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: link.Report("vrserver.exe (this session)", details...)}
}
