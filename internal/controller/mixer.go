package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

// ErrMixerPending blocks dependent routes while native mute readback catches up.
var ErrMixerPending = errors.New("mixer readback pending")

type MixerBackend interface{ SetMixer(string, float32) error }
type MuteOwnership struct {
	Baseline float32 `json:"baseline"`
}
type Mixer struct {
	Path   string
	Owned  map[string]MuteOwnership
	Error  error
	loaded bool
}

func (m *Mixer) load() error {
	if m.loaded {
		return m.Error
	}
	m.loaded = true
	m.Owned = map[string]MuteOwnership{}
	if m.Path == "" {
		return nil
	}
	b, e := os.ReadFile(m.Path)
	if os.IsNotExist(e) {
		return nil
	}
	if e == nil {
		e = json.Unmarshal(b, &m.Owned)
	}
	if m.Owned == nil {
		m.Owned = map[string]MuteOwnership{}
	}
	for p, v := range m.Owned {
		if !validMute(p) || (v.Baseline != 0 && v.Baseline != 1) {
			e = fmt.Errorf("invalid mute ownership journal")
		}
	}
	m.Error = e
	return e
}
func validMute(p string) bool {
	for _, prefix := range []string{"Strip", "Bus"} {
		for n := 0; n < 8; n++ {
			if p == fmt.Sprintf("%s[%d].Mute", prefix, n) {
				return true
			}
		}
	}
	return false
}
func (m *Mixer) save() error {
	if m.Path == "" {
		return nil
	}
	return config.WriteJournal(m.Path, m.Owned)
}

// Reconcile establishes new mutes before routing, then releases obsolete ownership only after routes settle.
func (m *Mixer) Reconcile(b Backend, c config.Config, p routing.Plan, s model.Snapshot, release bool) error {
	if err := m.load(); err != nil {
		return err
	}
	i := c.VoiceIntent()
	if i == nil {
		return nil
	}
	wanted := map[string]bool{}
	for _, key := range []string{"mic-mute", "speaker-mute"} {
		parameters, requested, _ := routing.MuteTargets(i, &p, key)
		if requested || key == "speaker-mute" {
			for _, parameter := range parameters {
				wanted[parameter] = requested
			}
		}
	}
	keys := []string{}
	for p := range wanted {
		keys = append(keys, p)
	}
	if release {
		for p := range m.Owned {
			if _, active := wanted[p]; !active {
				keys = append(keys, p)
			}
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return nil
	}
	api, ok := b.(MixerBackend)
	if !ok {
		return fmt.Errorf("native mute unavailable")
	}
	for _, param := range keys {
		current, ok := s.Numbers[param]
		if !ok {
			return fmt.Errorf("mute observation unavailable: %s", param)
		}
		owner, owned := m.Owned[param]
		requested, active := wanted[param]
		if active && !owned {
			owner = MuteOwnership{Baseline: current}
			m.Owned[param] = owner
			if err := m.save(); err != nil {
				delete(m.Owned, param)
				return err
			}
		}
		value := float32(0)
		if requested {
			value = 1
		}
		if !active {
			value = owner.Baseline
			if !strings.HasPrefix(param, "Bus[") && current != 1 {
				value = current
			}
		}
		if current != value {
			if err := api.SetMixer(param, value); err != nil {
				return err
			}
			var next model.Snapshot
			var err error
			if fast, ok := b.(ParameterBackend); ok {
				next, err = fast.ParameterSnapshot()
			} else {
				next, err = b.Snapshot()
			}
			if err != nil {
				return err
			}
			v, known := next.Numbers[param]
			if !known || v != value {
				return fmt.Errorf("%w: %s", ErrMixerPending, param)
			}
		}
		if !active {
			delete(m.Owned, param)
			if err := m.save(); err != nil {
				m.Owned[param] = owner
				return err
			}
		}
	}
	return nil
}
func GainTarget(p *routing.Plan, target string) string {
	switch target {
	case "soundboard":
		if p != nil && p.Edition == 3 {
			return "Strip[7].Gain"
		}
	case "playback":
		if p != nil && p.Topology != nil {
			bus := p.Topology.PlaybackTarget
			if len(bus) == 2 && bus[0] == 'A' && bus[1] >= '1' && bus[1] <= '5' {
				return fmt.Sprintf("Bus[%d].Gain", bus[1]-'1')
			}
		}
	case "mic":
		if p != nil && p.Topology != nil && p.Topology.Voice != nil && p.Topology.Voice.Strip >= 0 {
			return fmt.Sprintf("Strip[%d].Gain", p.Topology.Voice.Strip)
		}
	}
	return ""
}
func GainIdentity(p *routing.Plan, s model.Snapshot, target string) string {
	param := GainTarget(p, target)
	if param == "" {
		return ""
	}
	if target == "mic" {
		v := p.Topology.Voice
		name := s.Assignments[fmt.Sprintf("input:%d", v.Strip+1)]
		if v.Strip < 2 && p.Topology.ASIOActive {
			name = s.Assignments["A1"]
		}
		return param + "|" + v.Effective + "|" + name
	}
	if target == "playback" {
		target = p.Topology.PlaybackTarget
		for _, op := range p.Topology.Operations {
			if op.Target == target && op.Device != nil && op.Device.Name != s.Assignments[target] {
				return ""
			}
		}
		if s.Assignments[target] == "" {
			return ""
		}
	}
	return param + "|" + s.Assignments[target]
}
func (c *Controller) Gain(ctx context.Context, target, identity string, delta float32, live bool) error {
	return c.changeGain(ctx, target, identity, delta, false, live)
}

// ResetGain sets an absolute 0 dB, independent of queued increments or old UI values.
func (c *Controller) ResetGain(ctx context.Context, target, identity string, live bool) error {
	return c.changeGain(ctx, target, identity, 0, true, live)
}

func (c *Controller) changeGain(ctx context.Context, target, identity string, delta float32, reset, live bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if math.IsNaN(float64(delta)) || math.IsInf(float64(delta), 0) || delta < -127 || delta > 127 {
		return fmt.Errorf("invalid gain increment")
	}
	if target == "soundboard" && (!c.Config.SoundboardReserved || c.Config.SoundboardPolicy == nil || c.Config.SoundboardPolicy() == nil) {
		return fmt.Errorf("soundboard input is not active")
	}
	s, e := c.observe(true)
	if e != nil {
		return e
	}
	p, e := routing.Build(c.planConfig(), s)
	if e != nil {
		return e
	}
	if identity == "" || identity != GainIdentity(&p, s, target) {
		return fmt.Errorf("gain target changed; try again")
	}
	param := GainTarget(&p, target)
	v, ok := s.Numbers[param]
	if !ok {
		return fmt.Errorf("gain unavailable")
	}
	if !live {
		return fmt.Errorf("preview: gain not applied")
	}
	api, ok := c.Backend.(MixerBackend)
	if !ok {
		return fmt.Errorf("gain API unavailable")
	}
	value := max(float32(-60), min(float32(12), v+delta))
	if reset {
		value = 0
	}
	if e = api.SetMixer(param, value); e != nil {
		return e
	}
	deadline := c.Clock.Now().Add(100 * time.Millisecond)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		s, e = c.observe(true)
		if e != nil {
			return e
		}
		if got, ok := s.Numbers[param]; ok && math.Abs(float64(got-value)) <= 0.01 {
			return nil
		}
		remaining := deadline.Sub(c.Clock.Now())
		if remaining <= 0 {
			return fmt.Errorf("%s gain unverified", strings.TrimSuffix(param, ".Gain"))
		}
		if e = c.Clock.Wait(ctx, min(10*time.Millisecond, remaining)); e != nil {
			return e
		}
	}
}
