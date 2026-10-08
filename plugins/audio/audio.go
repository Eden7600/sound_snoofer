// Package audio adapts the existing serialized audio worker to Snoofer.
package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/controller"
	"sound-snoofer/internal/ownership"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/voicemeeter"
	"sound-snoofer/snoofer"
)

// Settings keeps the operational state path stable across the application rename.
type Settings struct {
	SoundboardInput bool            `json:"soundboard_input,omitempty"`
	Config          json.RawMessage `json:"config"`
	StatePath       string          `json:"state_path"`
	DLL             string          `json:"dll,omitempty"`
	// GainStepDB is the mixer gain change per dial detent; nil means 1 dB.
	GainStepDB *float64 `json:"gain_step_db,omitempty"`
}

// gainStep is the validated dB change per dial detent.
func (s Settings) gainStep() float32 {
	if s.GainStepDB == nil {
		return 1
	}
	return float32(*s.GainStepDB)
}

// Instance owns the audio actor and its snapshot publisher.
type Instance struct {
	soundboardReserved     bool
	soundboard             *config.SoundboardRoutes
	statePath, ownedPolicy string
	ownedInput             int
	stopError              error
	policy                 *config.ProfilePolicy
	cancel                 context.CancelFunc
	done                   chan struct{}
	actions                chan control.Action
	mu                     sync.Mutex
	state                  control.State

	// The echo canceller's wanted insert hook and its request generation.
	echoInsert     *voicemeeter.InsertHook
	echoGeneration uint64

	// Settings edits. settings and raw are the saved plugin settings (paths as
	// written); running is the prepared configuration the worker reloads.
	// Guarded by mu; only runEdits replaces them.
	edits    chan editRequest
	settings Settings
	raw      json.RawMessage
	running  config.Config
	editErr  string
	prepare  func(Settings) (config.Config, error)
	// The Routing view, rebuilt by the publisher goroutine only.
	viewData     json.RawMessage
	viewRaw      string
	viewAt       time.Time
	saveSettings func(string, json.RawMessage, json.RawMessage) error
}

// Plugin returns inert registration metadata.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "audio", Command: command, Validate: validateSettings, Label: "Audio", Defaults: snoofer.MarshalSettings(Settings{Config: config.DefaultBytes(), StatePath: "audio"}), Start: start}
}

func start(ctx context.Context, services snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return nil, err
	}
	if _, err := config.Decode(settings.Config); err != nil {
		return nil, err
	}
	if settings.StatePath == "" {
		return nil, fmt.Errorf("audio state_path is required to preserve operational journals")
	}
	statePath := settings.StatePath
	if !filepath.IsAbs(statePath) {
		statePath = filepath.Join(filepath.Dir(services.Path), statePath)
	}
	owned, err := loadPolicyOwnership(statePath)
	if err != nil {
		return nil, err
	}
	i := &Instance{done: make(chan struct{}), actions: make(chan control.Action, 8), edits: make(chan editRequest, 4)}
	i.statePath = statePath
	i.soundboardReserved = settings.SoundboardInput
	i.settings, i.raw, i.saveSettings = settings, raw, services.SaveSettings
	if owned != nil {
		data, _ := json.Marshal(owned)
		i.ownedPolicy = string(data)
		i.ownedInput = owned.Devices.Input
	}
	// prepare turns saved settings into the configuration the worker runs,
	// both at start and after an edit.
	i.prepare = func(settings Settings) (config.Config, error) {
		cfg, err := config.Decode(settings.Config)
		if err != nil {
			return cfg, err
		}
		if owned != nil {
			cfg.VR = &owned.Devices
			cfg.PolicyPlayback = owned.Playback
		}
		cfg.SoundboardReserved = settings.SoundboardInput
		cfg.SoundboardPolicy = i.soundboardSnapshot
		if cfg.Profiles == nil {
			// Older configurations omit profiles; see config.DefaultProfiles.
			cfg.Profiles = config.DefaultProfiles()
		}
		cfg.Policy = i.policySnapshot
		return cfg, nil
	}
	if i.running, err = i.prepare(settings); err != nil {
		return nil, err
	}
	cfg := config.LoadChoices(statePath, i.running)
	var initialLease func()
	if services.Live && cfg.StateError == "" {
		initialLease, err = ownership.Acquire()
		if err != nil {
			return nil, err
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	i.cancel = cancel
	states := make(chan control.State, 1)
	workerDone := make(chan struct{})
	started := make(chan error, 1)
	firstOpen := true
	deps := control.Dependencies{
		Meters: true,
		OnStop: func(err error) {
			i.stopError = err
			if initialLease != nil {
				initialLease()
				initialLease = nil
			}
		},
		Prepare: func(c config.Config) config.Config {
			if p := i.policySnapshot(); p != nil {
				devices := p.Devices
				c.VR = &devices
				intent := c.VoiceIntent()
				if intent != nil && intent.VRProfile == nil {
					choices := p.Choices
					intent.VRProfile = &choices
					c.Intent = intent
				}
			}
			return c
		},
		Open: func(path string) (control.Client, error) {
			client, err := voicemeeter.Open(path)
			if firstOpen {
				started <- err
				firstOpen = false
			}
			return client, err
		},
		Acquire: func() (func(), error) {
			if initialLease != nil {
				release := initialLease
				initialLease = nil
				return release, nil
			}
			return ownership.Acquire()
		},
		Load: func(string) (config.Config, error) {
			i.mu.Lock()
			running := i.running
			i.mu.Unlock()
			return config.LoadChoices(statePath, running), nil
		},
		Insert: i.insertRequest,
	}
	go control.Work(runCtx, cfg, statePath, settings.DLL, services.Live, deps, i.actions, states, workerDone)
	go i.runEdits(runCtx)
	go func() {
		defer close(i.done)
		defer services.Controls.Remove("audio")
		gainStep := settings.gainStep()
		reports := reporter{processor: cfg.ProcessorProcess()}
		for s := range states {
			i.mu.Lock()
			i.state = s
			i.mu.Unlock()
			snapshot := s
			published := append(controls(snapshot), slotControls(snapshot)...)
			published = append(published, i.priorityControls(snapshot)...)
			published = append(published, reports.reports(snapshot, time.Now())...)
			_ = services.Controls.Publish("audio", published, func(ctx context.Context, r snoofer.Request) error {
				if r.ID == "audio.priority-edit" || r.ID == "audio.output-edit" {
					return i.queueEdit(ctx, editRequest{control: r.ID, value: r.Value})
				}
				action, err := action(snapshot, r)
				if err != nil {
					return err
				}
				if action.Kind == control.Gain {
					action.Delta *= gainStep
				}
				select {
				case i.actions <- action:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				default:
					return fmt.Errorf("audio command queue full")
				}
			})
		}
		<-workerDone
	}()
	select {
	case err := <-started:
		if err != nil {
			cancel()
			stopCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			return nil, errors.Join(err, i.Stop(stopCtx))
		}
	case <-ctx.Done():
		cancel()
		stopCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		return nil, errors.Join(ctx.Err(), i.Stop(stopCtx))
	}
	return i, nil
}

// Stop waits for the native actor to relinquish ownership.
func (i *Instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return i.stopError
	case <-ctx.Done():
		return ctx.Err()
	}
}

// micPathControls only act on the microphone path; the deck hides them while
// the mic stack is off. The mic stack toggle itself is never hidden.
var micPathControls = map[string]bool{
	"mic-mute": true, "source": true, "mode": true, "monitor": true, "gain-mic": true,
	"record-mic": true, "record-tap": true,
	"normal-source": true, "normal-mode": true, "normal-monitor": true,
	"vr-profile-source": true, "vr-profile-mode": true, "vr-profile-monitor": true,
}

func controls(s control.State) []snoofer.Control {
	out := []snoofer.Control{{ID: "audio.health", Label: "Audio health", Group: "System", Kind: "status", Value: s.Health, Status: strings.TrimSpace(s.Error + " " + s.StateError + " " + s.Notice), Available: true}}
	if s.Intent == nil {
		return out
	}
	add := func(id, label, group, kind, value string, options []string, ops ...string) {
		status := s.Error
		if !s.Connected {
			status = "Disconnected"
		}
		out = append(out, snoofer.Control{ID: "audio." + id, Label: label, Group: group, Kind: kind, Value: value, Options: options, Operations: ops, Available: true, Status: status, Icon: id})
	}
	i := s.ActiveIntent
	if i == nil {
		i = s.Intent
	}
	interfaceName, playbackName, micName := "None", "Unavailable", "Unavailable"
	interfaceStatus := ""
	if s.Plan != nil && s.Plan.Topology != nil {
		t := s.Plan.Topology
		if t.Voice != nil {
			micName = t.Voice.Effective
			if label := s.ChoiceLabels[micName]; label != "" {
				micName = label
			}
			if micName == "off" {
				micName = "Off"
			}
		}
		if t.ASIOName != "" {
			interfaceName = t.ASIOName
			if s.Snapshot.Assignments["A1"] != interfaceName {
				interfaceStatus = "Pending"
			}
		}
		if t.PlaybackTarget != "" {
			playbackName = s.Snapshot.Assignments[t.PlaybackTarget]
		}
		if len(t.ASIOUnavailable) > 0 {
			add("asio-unavailable", "Unavailable interfaces", "Diagnostics", "status", strings.Join(t.ASIOUnavailable, ", ")+" · ASIO owned by "+t.ASIOName, nil)
		}
	}
	if !s.Connected {
		interfaceName, playbackName, micName = "Unknown", "Unknown", "Unknown"
	}
	add("interface", "Interface", "System", "status", interfaceName, nil)
	if s.Connected {
		out[len(out)-1].Status = interfaceStatus
	}
	add("playback-device", "Playback device", "System", "status", playbackName, nil)
	add("mic-device", "Microphone", "System", "status", micName, nil)
	activeTarget := s.Intent.Source
	activeOptions := s.MicOptions
	if s.Profile == "VR" && s.Intent.VRProfile != nil {
		activeTarget = s.Intent.VRProfile.Source
		activeOptions = s.VRSourceOptions
	}
	add("source", "Mic stack target", "Bindings", "selection", activeTarget, microphoneTargets(activeOptions, activeTarget), "set")
	add("mode", "Mic processing", "Bindings", "selection", i.Mode, []string{"direct", "element"}, "set", "press")
	add("monitor", "Monitor", "Bindings", "selection", i.Monitor, []string{"off", "pre", "post"}, "set", "press")
	add("output", "Playback", "Bindings", "selection", i.PlaybackDevice, s.OutputOptions, "set")
	for _, row := range []struct {
		id, label string
		value     bool
	}{
		{"mic-stack", "Mic stack", s.Intent.Enabled}, {"mic-mute", "Mic mute", i.MicMuted}, {"speaker-mute", "Playback mute", i.PlaybackMuted},
		{"defaults", "Protect Windows defaults", i.ProtectDefaults}, {"auto-recover", "Automatic recovery", i.AutoRecover},
	} {
		value := "Off"
		if row.value {
			value = "On"
		}
		add(row.id, row.label, "Shared audio", "toggle", value, nil, "press")
	}
	heldDevices, heldSends := 0, 0
	if s.Plan != nil && s.Plan.Topology != nil {
		heldDevices, heldSends = s.Plan.Topology.HeldDevices, s.Plan.Topology.HeldSends
	}
	for _, row := range []struct {
		id, label string
		value     bool
		held      int
	}{
		{"pause-devices", "Disable device manipulation", i.PauseDevices, heldDevices},
		{"pause-sends", "Disable send manipulation", i.PauseSends, heldSends},
	} {
		value := "Off"
		if row.value {
			value = "On"
		}
		add(row.id, row.label, "Shared audio", "toggle", value, nil, "press")
		if row.value && row.held > 0 {
			out[len(out)-1].Status = fmt.Sprintf("%d held", row.held)
		}
	}
	for source, enabled := range i.Playback {
		value := "Off"
		if enabled {
			value = "On"
		}
		add("playback:"+source, source+" playback", "Shared audio", "toggle", value, nil, "press")
	}
	if i.Recording != nil {
		for _, row := range []struct {
			id, label string
			value     bool
		}{
			{"record-mic", "Record microphone", i.Recording.MicEnabled}, {"record-computer", "Record computer", i.Recording.ComputerEnabled},
			{"record-loop", "Loop recording", i.Recording.Loop}, {"record-vst", "Recording to VST", i.Recording.ToVST},
		} {
			value := "Off"
			if row.value {
				value = "On"
			}
			add(row.id, row.label, "Recording", "toggle", value, nil, "press")
		}
		add("record-tap", "Mic stage", "Recording", "selection", i.Recording.MicTap, []string{"pre", "post"}, "press", "set")
		for _, id := range []string{"record-start", "record-stop", "snippet-play", "snippet-stop"} {
			add(id, strings.ReplaceAll(id, "-", " "), "Transport", "command", s.Recorder.State(), nil, "press")
		}
		add("record-toggle", "Record", "Transport", "command", s.Recorder.State(), nil, "press")
		for _, row := range tapeRows(s.Recorder) {
			add(row.id, row.label, "Transport", "command", row.value, nil, "press")
			out[len(out)-1].Available = row.available
			out[len(out)-1].Icon = row.icon
		}
	}
	for _, target := range []string{"playback", "mic"} {
		parameter := controller.GainTarget(s.Plan, target)
		value := "Unknown"
		if gain, ok := s.Snapshot.Numbers[parameter]; ok {
			value = fmt.Sprintf("%.1f dB", gain)
		}
		add("gain-"+target, map[string]string{"playback": "Playback", "mic": "Mic"}[target], "Shared audio", "numeric", value, nil, "adjust", "press")
		level, known := s.Levels[meterSource(s.Plan, target, parameter)]
		known = known && controller.GainIdentity(s.Plan, s.Snapshot, target) != "" && level >= 0 && !math.IsNaN(float64(level)) && !math.IsInf(float64(level), 0)
		db := -60.0
		if known && level > 0 {
			db = max(-60, 20*math.Log10(float64(level)))
		}
		out[len(out)-1].Meter = snoofer.Meter{Present: true, Known: known && s.Connected && !s.RecoveryPending, DB: db, At: s.LevelsAt}
	}
	add("engine-restart", "Restart audio engine", "Bindings", "command", "", nil, "press")
	if s.RestartConfirmation {
		add("engine-confirm", "Confirm audio restart (interrupts recording)", "System", "command", "Confirmation required", nil, "press")
	}
	normal := s.Intent
	normalOptions := microphoneTargets(s.MicOptions, normal.Source)
	add("normal-source", "Mic stack target", "Normal microphone", "selection", normal.Source, normalOptions, "set")
	add("normal-mode", "Processing", "Normal microphone", "selection", normal.Mode, []string{"direct", "element"}, "set")
	add("normal-monitor", "Monitoring", "Normal microphone", "selection", normal.Monitor, []string{"off", "pre", "post"}, "set")
	add("normal-output", "Playback", "Normal playback", "selection", normal.PlaybackDevice, s.OutputOptions, "set")
	if s.VRConfigured && normal.VRProfile != nil {
		p := normal.VRProfile
		add("vr-profile-source", "Mic stack target", "VR microphone", "selection", p.Source, microphoneTargets(s.VRSourceOptions, p.Source), "set")
		add("vr-profile-mode", "Processing", "VR microphone", "selection", p.Mode, []string{"direct", "element"}, "set")
		add("vr-profile-monitor", "Monitoring", "VR microphone", "selection", p.Monitor, []string{"off", "pre", "post"}, "set")
		add("vr-profile-output", "Playback", "VR playback", "selection", p.Playback, s.VROutputOptions, "set")
	}
	for n := range out {
		out[n].Epoch = s.Revision
		out[n].SurfaceOnly = out[n].Group == "Transport" || out[n].Group == "Bindings" || out[n].Group == "Diagnostics" || out[n].ID == "audio.playback-device" || out[n].ID == "audio.mic-device"
		out[n].OptionLabels = map[string]string{"auto": "Automatic", "": "Automatic"}
		for id, label := range s.ChoiceLabels {
			out[n].OptionLabels[id] = label
		}
		// Activity metering marks microphones Auto currently skips.
		for _, id := range s.SilentMics {
			if label, ok := out[n].OptionLabels[id]; ok {
				out[n].OptionLabels[id] = label + " · Silent"
			}
		}
		key := strings.TrimPrefix(out[n].ID, "audio.")
		out[n].ShortLabel = shortLabel(key)
		switch key {
		case "mode", "normal-mode", "vr-profile-mode":
			out[n].Icon = "mode-" + out[n].Value
		case "record-tap":
			out[n].Icon = "tap-" + out[n].Value
		case "normal-monitor", "vr-profile-monitor":
			out[n].Icon = "monitor"
		}
		if key == "mic-stack" {
			out[n].Group = "Mic stack"
		}
		if key == "mic-stack" {
			out[n].Icon = "mic-stack"
			if !s.Intent.Enabled {
				out[n].Icon = "mic-stack-off"
			}
		}
		if m, handled := control.ObserveMute(s, key); handled {
			if !m.Known {
				out[n].Status = "Unknown"
			} else if m.Pending {
				out[n].Status = "Pending"
			} else {
				out[n].Status = ""
			}
			if m.Known && m.Muted {
				out[n].Icon = key + "-muted"
			}
		}
		if f, ok := s.Feedback[key]; ok {
			if f.Kind == control.NoticePending {
				out[n].Status = "Pending"
			}
			if f.Kind == control.NoticeError {
				out[n].Status = "Failed"
			}
		}
		if key == "record-toggle" && s.Recorder.State() == "Recording" {
			out[n].Icon = "record-stop"
			out[n].Label = "Stop recording"
			out[n].ShortLabel = "Stop rec"
		}
		out[n].EnterOnly = out[n].ID == "audio.engine-confirm"
		if s.Profile == "VR" && strings.HasPrefix(out[n].Group, "Normal") {
			out[n].Subdued = true
			out[n].Status = "VR override"
		}
		if out[n].Group == "Bindings" {
			out[n].Label += " · " + s.Profile
		}
		// With the mic stack off, mic-path controls have no use on control
		// surfaces; the GUI still shows them, so availability is unchanged.
		out[n].Hidden = !s.Intent.Enabled && micPathControls[key]
		if out[n].Kind == "numeric" || out[n].Group == "Transport" {
			out[n].Available = s.Connected && s.Live
			if strings.HasPrefix(key, "gain-") {
				out[n].Available = out[n].Available && controller.GainIdentity(s.Plan, s.Snapshot, strings.TrimPrefix(key, "gain-")) != ""
			}
		}
	}
	return out
}

func action(s control.State, r snoofer.Request) (control.Action, error) {
	key := strings.TrimPrefix(r.ID, "audio.")
	activeKey := key
	if key == "source" || key == "output" || key == "mode" || key == "monitor" {
		if s.Profile == "VR" {
			key = "vr-profile-" + key
		} else {
			key = "normal-" + key
		}
	}
	a := control.Action{Origin: "snoofer", Revision: s.Revision, Kind: control.Edit, Row: key, Value: r.Value}
	if strings.HasPrefix(key, "gain-") {
		target := strings.TrimPrefix(key, "gain-")
		if r.Operation == "adjust" {
			a.Kind = control.Gain
			a.Target = target
			a.Identity = controller.GainIdentity(s.Plan, s.Snapshot, target)
			a.Delta = float32(r.Delta)
			return a, nil
		}
		key = map[string]string{"playback": "speaker-mute", "mic": "mic-mute"}[target]
		a.Row = key
		activeKey = key
	}
	if strings.HasPrefix(key, "slot-") {
		return slotAction(s, key, r, a)
	}
	switch key {
	case "engine-restart", "engine-confirm":
		a.Kind = control.Restart
		a.Confirm = key == "engine-confirm"
		return a, nil
	case "record-start":
		a.Kind = control.RecordStart
		return a, nil
	case "record-stop", "snippet-stop":
		a.Kind = control.RecordStop
		return a, nil
	case "snippet-play":
		a.Kind = control.SnippetPlay
		return a, nil
	case "tape-play", "tape-stop", "tape-rew", "tape-ff":
		a.Kind = control.Tape
		a.Target = strings.TrimPrefix(key, "tape-")
		return a, nil
	case "record-toggle":
		switch s.Recorder.State() {
		case "Stopped":
			a.Kind = control.RecordStart
		case "Recording":
			a.Kind = control.RecordStop
		default:
			return a, fmt.Errorf("recorder state is not actionable")
		}
		return a, nil
	case "speaker-mute":
		return control.ToggleMute(s, key), nil
	}
	if r.Operation == "press" {
		for _, c := range controls(s) {
			if c.ID == "audio."+activeKey {
				if c.Kind == "toggle" {
					a.Value = fmt.Sprint(c.Value != "On")
				} else {
					for n, v := range c.Options {
						if v == c.Value {
							a.Value = c.Options[(n+1)%len(c.Options)]
							break
						}
					}
				}
			}
		}
	}
	return a, nil
}

// Targets preserve a missing saved choice, but Off belongs to enablement only.
func microphoneTargets(options []string, saved string) []string {
	result := []string{"auto"}
	for _, id := range append(slices.Clone(options), saved) {
		if id != "" && id != "off" && !slices.Contains(result, id) {
			result = append(result, id)
		}
	}
	return result
}

// meterSource is the level reading for a gain target's meter. While Element
// processes the mic, the mic meter shows its return on the AUX strip.
func meterSource(p *routing.Plan, target, gainParameter string) string {
	if target == "mic" && p != nil && p.Topology != nil && p.Topology.Voice != nil && p.Topology.Voice.EffectiveMode == "element" {
		return "Strip[6].Gain"
	}
	return gainParameter
}
