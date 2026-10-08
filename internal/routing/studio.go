package routing

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type Operation struct {
	Target      string        `json:"target,omitempty"`
	Device      *model.Device `json:"device,omitempty"`
	Parameter   string        `json:"parameter,omitempty"`
	Value       int           `json:"value,omitempty"`
	BeforeName  string        `json:"before_name,omitempty"`
	BeforeValue float32       `json:"before_value,omitempty"`
	Change      bool          `json:"change"`
}
type Topology struct {
	ASIOName        string           `json:"asio_name,omitempty"`
	ASIOUnavailable []string         `json:"asio_unavailable,omitempty"`
	MicStrips       []int            `json:"mic_strips,omitempty"`
	Recording       *RecordingStatus `json:"recording,omitempty"`
	Voice           *VoiceStatus     `json:"voice,omitempty"`
	Transition      []Operation      `json:"transition,omitempty"`
	ASIOActive      bool             `json:"asio_active"`
	PlaybackTarget  string           `json:"playback_target,omitempty"`
	Operations      []Operation      `json:"operations"`
	Unresolved      []string         `json:"unresolved,omitempty"`
	// WiredMics are the microphones wired for activity metering; nil when
	// activity metering is off.
	WiredMics []string `json:"wired_mics,omitempty"`
	// Outputs are the output slots in configuration order.
	Outputs []OutputStatus `json:"outputs,omitempty"`
	// PlaybackUnavailable explains a voice-profile plan without a playback output.
	PlaybackUnavailable []string `json:"playback_unavailable,omitempty"`
	InventoryKey        string   `json:"-"`
	// HeldDevices and HeldSends count changes the intent's pauses hold back.
	HeldDevices int `json:"held_devices,omitempty"`
	HeldSends   int `json:"held_sends,omitempty"`
}

func (t *Topology) Key() string {
	type desired struct {
		Target, Parameter string
		Device            *model.Device
		Value             int
	}
	v := struct {
		Active     bool
		Playback   string
		Ops        []desired
		Unresolved []string
	}{Active: t.ASIOActive, Playback: t.PlaybackTarget, Unresolved: t.Unresolved}
	for _, op := range t.Operations {
		v.Ops = append(v.Ops, desired{op.Target, op.Parameter, op.Device, op.Value})
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// Enumeration reordering must not invalidate a transaction.
func InventoryKey(s model.Snapshot) string {
	entries := []string{}
	for _, d := range s.Devices {
		b, _ := json.Marshal(d)
		entries = append(entries, string(b))
	}
	sort.Strings(entries)
	b, _ := json.Marshal(struct {
		Edition        int
		SteamVRRunning bool
		ElementRunning bool
		Devices        []string
	}{s.Edition, vrRunning(s), s.ElementRunning(), entries})
	return string(b)
}
func selectDevice(list []config.Candidate, direction string, devices []model.Device) (*model.Device, []string) {
	reasons := []string{}
	for _, c := range list {
		matches := []model.Device{}
		for _, d := range devices {
			if candidateMatch(c, direction, d) {
				matches = append(matches, d)
			}
		}
		if len(matches) == 1 {
			return &matches[0], reasons
		}
		reasons = append(reasons, fmt.Sprintf("%s pattern %q matched %d devices", direction, c.Pattern, len(matches)))
	}
	return nil, reasons
}
func buildStudio(c config.Config, s model.Snapshot) (Plan, error) {
	t := &Topology{InventoryKey: InventoryKey(s)}
	p := Plan{Edition: s.Edition, Topology: t}
	profile := c.Studio
	n, _ := model.Limits(s.Edition)
	for _, slot := range model.Slots(s.Edition) {
		if _, ok := s.Assignments[slot]; !ok {
			return p, fmt.Errorf("snapshot missing %s", slot)
		}
	}
	asio, selectedInterface, err := selectInterface(profile, s)
	if err != nil {
		return p, err
	}
	t.ASIOActive = asio != nil
	if asio != nil {
		t.ASIOName = asio.Name
		for n := range profile.ASIO {
			other, err := matchInterface(&profile.ASIO[n], s)
			if err == nil && other != nil && other.Name != asio.Name {
				t.ASIOUnavailable = append(t.ASIOUnavailable, other.Name)
			}
		}
	}
	ownsPlayback := func(name string) bool {
		// An output slot's device is never a playback output, even if it
		// matches a playback pattern.
		if name == "" || profile.OutputDevice(name) {
			return false
		}
		for _, c := range profile.Playback {
			if c.Regex.MatchString(name) {
				return true
			}
		}
		return false
	}
	ownsASIO := profile.OwnsASIO
	if t.ASIOActive && s.Assignments["A1"] != "" && !ownsASIO(s.Assignments["A1"]) && !ownsPlayback(s.Assignments["A1"]) && !profile.OutputDevice(s.Assignments["A1"]) {
		return p, fmt.Errorf("A1 is occupied by unmanaged device %q; cannot reserve it for ASIO", s.Assignments["A1"])
	}
	playback, reasons := selectDevice(profile.Playback, "output", playbackDevices(profile, s))
	if intent := c.VoiceIntent(); intent != nil && intent.PlaybackDevice != "" {
		for _, name := range PlaybackOptions(c, s) {
			if name != intent.PlaybackDevice {
				continue
			}
			if asio != nil && asio.Name == name {
				playback = asio
				break
			}
			for _, d := range s.Devices {
				if d.Name == name && d.Available && d.Driver == "wdm" && d.Direction == "output" {
					chosen := d
					playback = &chosen
					break
				}
			}
		}
	}
	if i := c.VoiceIntent(); i != nil && i.PreferVRPlayback {
		if d, _ := headsetDevice(c, s, "output", ""); d != nil {
			playback = d
		}
	}
	if c.ProfileResolved {
		playback = c.ProfilePlayback
	}
	if playback == nil {
		if c.ProfileResolved {
			reasons = append(reasons, "No eligible playback device in active profile")
		}
		// The voice profile still routes the mic stack without a playback output;
		// the reasons remain diagnostic rather than blocking apply or recovery.
		if profile.Voice != nil {
			t.PlaybackUnavailable = reasons
		} else {
			t.Unresolved = append(t.Unresolved, reasons...)
		}
	}
	oldBuses := []string{}
	if playback != nil && playback.Driver == "asio" {
		t.PlaybackTarget = "A1"
	}
	for i := 1; i <= n; i++ {
		target := fmt.Sprintf("A%d", i)
		current := s.Assignments[target]
		if ownsPlayback(current) || (i == 1 && ownsASIO(current)) {
			oldBuses = append(oldBuses, target)
		}
		if t.PlaybackTarget == "" && !(t.ASIOActive && i == 1) && (current == "" || ownsPlayback(current) || (i == 1 && ownsASIO(current))) {
			t.PlaybackTarget = target
		}
	}
	// Buses holding an output slot's device are not free for Playback, which
	// only takes the last slot's bus when nothing else is left.
	slots := holdOutputBuses(profile, s, n, t.ASIOActive)
	if t.PlaybackTarget == "" && playback != nil {
		t.PlaybackTarget = slots.evict()
	}
	if t.PlaybackTarget == "" {
		if profile.Voice == nil {
			return p, fmt.Errorf("no free hardware output for playback")
		}
		playback = nil
		t.Unresolved = append(t.Unresolved, "no free hardware output for playback")
	}
	// No playback device means no reconfiguration: avoids replacing the only
	// audible output while there is no destination to migrate its source sends.
	if playback == nil {
		if profile.Voice == nil {
			return p, nil
		}
		t.PlaybackTarget = ""
	}
	deviceOp := func(target string, d model.Device) {
		d.Direction = func() string { slot, _ := model.ParseSlot(target); return slot.Direction }()
		t.Operations = append(t.Operations, Operation{Target: target, Device: &d, BeforeName: s.Assignments[target], Change: s.Assignments[target] != d.Name})
	}
	numberOp := func(param string, value int) error {
		current, ok := s.Numbers[param]
		if !ok {
			return fmt.Errorf("snapshot missing %s", param)
		}
		for i := range t.Operations {
			if t.Operations[i].Parameter == param {
				t.Operations[i].Value = value
				t.Operations[i].Change = current != float32(value)
				return nil
			}
		}
		t.Operations = append(t.Operations, Operation{Parameter: param, Value: value, BeforeValue: current, Change: current != float32(value)})
		return nil
	}
	clear := model.Device{Driver: "wdm", Available: true}
	micActive := true
	if intent := c.VoiceIntent(); intent != nil {
		micActive = intent.MicActive()
	}
	// Disable the old input patch before installing a direct fallback mic.
	// Off disconnects ASIO inputs without releasing the A1 output device.
	if !t.ASIOActive || !micActive {
		for i := 0; i < 4; i++ {
			if e := numberOp(fmt.Sprintf("Patch.asio[%d]", i), 0); e != nil {
				return p, e
			}
		}
	}
	if t.ASIOActive {
		deviceOp("input:1", clear)
		deviceOp("input:2", clear)
		deviceOp("A1", *asio)
	} else if profile.Voice != nil {
		deviceOp("input:1", clear)
		deviceOp("input:2", clear)
	} else {
		mic, why := selectDevice(profile.FallbackMic, "input", s.Devices)
		if mic == nil {
			t.Unresolved = append(t.Unresolved, why...)
		} else {
			deviceOp("input:1", *mic)
		}
		deviceOp("input:2", clear)
	}
	if playback != nil && playback.Driver != "asio" {
		deviceOp(t.PlaybackTarget, *playback)
	}
	slots.place(c, t, ownsPlayback)
	for _, o := range t.Outputs {
		if o.State == OutputOK && s.Assignments[o.Bus] != o.Device {
			deviceOp(o.Bus, model.Device{Name: o.Device, Driver: "wdm", Available: true})
		}
	}
	for _, bus := range slots.duplicates {
		deviceOp(bus, clear)
	}
	if t.ASIOActive && micActive {
		desk, lav := selectedInterface.Inputs[0], selectedInterface.Inputs[1]
		// Activity metering wires only the checked microphones.
		if c.ProfileWired != nil {
			if !slices.Contains(c.ProfileWired, "desk") {
				desk = 0
			}
			if !slices.Contains(c.ProfileWired, "lav") {
				lav = 0
			}
		}
		for i, v := range []int{desk, desk, lav, lav} {
			if e := numberOp(fmt.Sprintf("Patch.asio[%d]", i), v); e != nil {
				return p, e
			}
		}
	}
	// Transfer the union of currently enabled playback sends. This also repairs
	// interrupted transitions where both old and new device slots are assigned.
	if playback != nil && profile.MovePlaybackRouting && len(oldBuses) > 0 {
		for strip := 0; strip < model.StripCount(s.Edition); strip++ {
			if profile.Voice != nil && (strip < 3 || strip == 6 || (c.VR != nil && strip == c.VR.Input-1)) {
				continue
			}
			enabled := 0
			for _, bus := range oldBuses {
				param := fmt.Sprintf("Strip[%d].%s", strip, bus)
				v, ok := s.Numbers[param]
				if !ok {
					return p, fmt.Errorf("snapshot missing %s", param)
				}
				if v != 0 {
					enabled = 1
				}
			}
			if e := numberOp(fmt.Sprintf("Strip[%d].%s", strip, t.PlaybackTarget), enabled); e != nil {
				return p, e
			}
			for _, bus := range oldBuses {
				if bus != t.PlaybackTarget {
					if e := numberOp(fmt.Sprintf("Strip[%d].%s", strip, bus), 0); e != nil {
						return p, e
					}
				}
			}
		}
	}
	// Source rules override migrated button states, even without device changes.
	for _, source := range profile.PlaybackSources {
		virtual := int(source[len(source)-1] - '1')
		strip := n + virtual
		enabled := 1
		if i := c.VoiceIntent(); i != nil && !i.Playback[source] {
			enabled = 0
		}
		if t.PlaybackTarget != "" {
			if e := numberOp(fmt.Sprintf("Strip[%d].%s", strip, t.PlaybackTarget), enabled); e != nil {
				return p, e
			}
		}
		for i := 1; i <= n; i++ {
			bus := fmt.Sprintf("A%d", i)
			if bus == t.PlaybackTarget {
				continue
			}
			if ownsPlayback(s.Assignments[bus]) || (i == 1 && t.ASIOActive) {
				if e := numberOp(fmt.Sprintf("Strip[%d].%s", strip, bus), 0); e != nil {
					return p, e
				}
			}
		}
	}
	// Each output slot receives only its switched-on playback sources.
	for _, o := range t.Outputs {
		if o.Bus == "" {
			continue
		}
		for _, source := range profile.PlaybackSources {
			strip := n + int(source[len(source)-1]-'1')
			if e := numberOp(fmt.Sprintf("Strip[%d].%s", strip, o.Bus), boolValue(o.Receives(source))); e != nil {
				return p, e
			}
		}
	}
	// Release former playback outputs last. Match ownership across restarts;
	// never clear unrelated occupied outputs.
	for _, bus := range oldBuses {
		if playback != nil && bus != t.PlaybackTarget && !(t.ASIOActive && bus == "A1") && t.OutputAt(bus) == nil {
			deviceOp(bus, clear)
		}
	}
	if profile.Voice != nil {
		return addVoice(c, s, p)
	}
	return p, nil
}
