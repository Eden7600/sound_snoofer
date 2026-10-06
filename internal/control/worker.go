package control

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/controller"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/windowsaudio"
)

type Client interface {
	controller.Backend
	Close() error
}
type Dependencies struct {
	Meters        bool
	OnStop        func(error)
	Prepare       func(config.Config) config.Config
	StartDefaults func(context.Context) (chan windowsaudio.Request, chan windowsaudio.Result, <-chan struct{})
	Save          func(string, config.Config, *config.Intent, string) (string, error)
	Open          func(string) (Client, error)
	Acquire       func() (func(), error)
	Load          func(string) (config.Config, error)
}
type State struct {
	Levels                           map[string]float32
	LevelsAt                         time.Time
	ActiveIntent                     *config.Intent
	VRSourceOptions, VROutputOptions []string
	Profile                          string
	VRConfigured                     bool
	Feedback                         map[string]Feedback
	noticeRevision                   uint64
	// ObservedAt advances only on successful native snapshots; PublishedAt tracks worker progress.
	ObservedAt                          time.Time
	PublishedAt                         time.Time
	ConfigPath                          string
	ChoiceLabels                        map[string]string
	VRMicAvailable, VRPlaybackAvailable bool
	DefaultKind                         windowsaudio.StatusKind
	DefaultsDetail                      windowsaudio.Result // Latest full defaults observation, for diagnostics.
	Remote                              model.RemoteInfo    // Native Remote API identity, for diagnostics.
	Stalled                             bool                // A qualified callback stall: the engine is not processing audio.
	RecoveryOutcome                     string
	RecoveryPending                     bool
	VRMic                               string
	VRPlayback                          string
	Acks                                map[string]Ack
	Health                              string
	Defaults                            string
	RestartConfirmation                 bool

	MicOptions    []string
	OutputOptions []string
	EditAck       uint64
	EditError     string
	Recorder      *model.RecorderSnapshot
	Intent        *config.Intent
	Revision      uint64
	StateError    string
	Live          bool
	Connected     bool
	Error         string
	Notice        string
	NoticeKind    NoticeKind
	NoticeUntil   time.Time
	Snapshot      model.Snapshot
	Plan          *routing.Plan
}

// NeedsAttention reports an active diagnostic without parsing its presentation text.
func (s State) NeedsAttention() bool {
	return s.Error != "" || s.StateError != "" || (s.Notice != "" && s.NoticeKind == NoticeError)
}

type ActionKind int

const (
	toggleLive ActionKind = iota
	reload
	refresh
	editRule
	resetChoices
	startRecording
	stopRecording
	playSnippet
	gain
	restartEngine
)

type Action struct {
	ResetGain bool
	Origin    string
	Target    string
	Identity  string
	Delta     float32
	Confirm   bool

	ID         uint64
	Edits      []SettingEdit
	Kind       ActionKind
	Row, Value string
	Revision   uint64
}

var ToggleLive = Action{Kind: toggleLive}
var Reload = Action{Kind: reload}
var Refresh = Action{Kind: refresh}

type observed struct {
	Client
	snapshot   model.Snapshot
	readError  error
	observedAt time.Time
}

func (o *observed) Snapshot() (model.Snapshot, error) {
	s, e := o.Client.Snapshot()
	o.readError = e
	if e == nil {
		o.snapshot = s
		o.observedAt = time.Now()
	}
	return s, e
}
func (o *observed) ParameterSnapshot() (model.Snapshot, error) {
	b, ok := o.Client.(controller.ParameterBackend)
	if !ok {
		return o.Snapshot()
	}
	s, e := b.ParameterSnapshot()
	o.readError = e
	if e == nil {
		o.snapshot = s
		o.observedAt = time.Now()
	}
	return s, e
}
func (o *observed) SetNumber(p string, v int) error {
	b, ok := o.Client.(interface{ SetNumber(string, int) error })
	if !ok {
		return fmt.Errorf("backend does not support numeric routing")
	}
	return b.SetNumber(p, v)
}
func (o *observed) Recorder() (model.RecorderSnapshot, error) {
	b, ok := o.Client.(controller.RecorderBackend)
	if !ok {
		return model.RecorderSnapshot{}, fmt.Errorf("recorder API unavailable")
	}
	r, e := b.Recorder()
	o.snapshot.Recorder = &r
	if e != nil {
		o.snapshot.Recorder = &model.RecorderSnapshot{Error: e.Error()}
	}
	return r, e
}
func (o *observed) SetRecorder(p string, v int) error {
	b, ok := o.Client.(controller.RecorderBackend)
	if !ok {
		return fmt.Errorf("recorder API unavailable")
	}
	return b.SetRecorder(p, v)
}

// The actor exclusively owns DLL calls, controller state and writer ownership.
// State messages contain fresh snapshots; no shared mutable maps reach the UI.
func Work(ctx context.Context, cfg config.Config, path, dll string, live bool, deps Dependencies, actions <-chan Action, states chan State, done chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)
	defer close(states)
	var cleanupError error
	defer func() {
		if deps.OnStop != nil {
			deps.OnStop(cleanupError)
		}
	}()
	defaultCtx, defaultCancel := context.WithCancel(ctx)
	defer defaultCancel()
	startDefaults := deps.StartDefaults
	if startDefaults == nil {
		startDefaults = windowsaudio.Start
	}
	defaultRequests, defaultResults, defaultDone := startDefaults(defaultCtx)
	revokeDefaults := func() error {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
		defer stopCancel()
		return windowsaudio.Revoke(stopCtx, defaultRequests, defaultDone)
	}
	state := State{Acks: map[string]Ack{}}
	seen := map[string]uint64{}
	if deps.Save == nil {
		deps.Save = config.SaveIntent
	}
	revision := uint64(1)
	var release func()
	var backend *observed
	configureMonitor := func(enable bool) error {
		if backend == nil {
			return nil
		}
		if m, ok := backend.Client.(interface{ SetMonitoring(bool) error }); ok {
			return m.SetMonitoring(enable)
		}
		if enable {
			return fmt.Errorf("callback monitoring unsupported by this backend")
		}
		return nil
	}
	defer func() {
		err := revokeDefaults()
		cleanupError = errors.Join(cleanupError, err)
		if err == nil && release != nil {
			release()
		}
		defaultCancel()
	}()
	setLive := func(enable bool) {
		if enable == state.Live {
			return
		}
		if enable {
			if cfg.StateError != "" {
				state.SetNotice(cfg.StateError, NoticeError, time.Now())
				return
			}
			r, e := deps.Acquire()
			if e != nil {
				state.SetNotice(e.Error(), NoticeError, time.Now())
				return
			}
			release = r
		} else if release != nil {
			if err := configureMonitor(false); err != nil {
				state.SetNotice(err.Error(), NoticeError, time.Now())
				return
			}
			if err := revokeDefaults(); err != nil {
				state.SetNotice(err.Error(), NoticeError, time.Now())
				return
			}
			release()
			release = nil
		}
		state.Live = enable
		state.SetNotice("", NoticeSuccess, time.Now())
	}
	setLive(live)
	defer func() {
		if backend != nil {
			if err := backend.Close(); err != nil {
				cleanupError = errors.Join(cleanupError, err)
				// Retain ownership when native callback cleanup is uncertain.
				release = nil
			}
		}
	}()
	recovery := newRecovery(path)
	mixer := &controller.Mixer{Path: path + ".mutes.json"}
	var ctl *controller.Controller
	var confirmRevision uint64
	reset := func() {
		if backend == nil {
			return
		}
		prepared := ctl != nil && ctl.RecorderPrepared
		ctl = &controller.Controller{Mixer: mixer, RecorderPrepared: prepared, Backend: backend, Config: cfg, Clock: controller.RealClock{}, Emit: func(e controller.Event) {
			if e.Kind == "error" {
				state.Error = e.Message
			}
		}}
	}
	sampleLevels := func() {
		if !deps.Meters {
			return
		}
		state.Levels = nil
		if backend != nil && state.Connected && !state.RecoveryPending {
			if reader, ok := backend.Client.(interface{ GainLevels(int) map[string]float32 }); ok {
				strip := -1
				if state.Plan != nil && state.Plan.Topology != nil && state.Plan.Topology.Voice != nil {
					strip = state.Plan.Topology.Voice.Strip
				}
				state.Levels = reader.GainLevels(strip)
			}
		}
		state.LevelsAt = time.Now()

	}
	emit := func() {
		select {
		case states <- state:
		default:
			select {
			case <-states:
			default:
			}
			select {
			case states <- state:
			case <-ctx.Done():
			}
		}
	}
	publish := func() {
		sampleLevels() // Routing observations may change the meter's source.
		state.pruneFeedback(time.Now())
		state.PublishedAt = time.Now()
		state.ConfigPath = path
		state.ChoiceLabels = map[string]string{}
		if cfg.VR != nil {
			for _, h := range cfg.VR.Headsets {
				state.ChoiceLabels["vr:"+h.ID] = h.Label
			}
		}
		if backend != nil {
			state.ObservedAt = backend.observedAt
		}
		r := windowsaudio.Request{}
		if i := cfg.VoiceIntent(); i != nil {
			r.Enabled = i.ProtectDefaults
			r.Live = state.Live
		}
		if cfg.VR != nil {
			r.Playback = cfg.VR.PlaybackDefault
			r.Capture = cfg.VR.CaptureDefault
		}
		if cfg.WindowsDefaults != nil {
			r.Playback = cfg.WindowsDefaults.Playback
			r.Capture = cfg.WindowsDefaults.Capture
		}
		windowsaudio.Update(defaultRequests, r)

		select {
		case v, ok := <-defaultResults:
			if ok {
				state.Defaults = v.Status
				state.DefaultKind = v.Kind
				state.DefaultsDetail = v
			}
		default:
		}

		state.MicOptions = []string{"off"}
		state.OutputOptions = []string{""}
		if state.Connected {
			state.MicOptions = routing.MicrophoneOptions(cfg, state.Snapshot)
			state.OutputOptions = routing.PlaybackOptions(cfg, state.Snapshot)
		}
		state.VRMicAvailable = false
		state.VRPlaybackAvailable = false
		state.VRMic = "No eligible headset mic"
		state.VRPlayback = "No eligible headset playback"
		for _, option := range state.MicOptions {
			if strings.HasPrefix(option, "vr:") {
				state.VRMic = "Available"
				state.VRMicAvailable = true
				break
			}
		}
		if cfg.VR == nil || len(cfg.VR.Headsets) == 0 {
			state.VRMic = "Configure headset matchers"
			state.VRPlayback = "Configure headset matchers"
		} else {
			for _, d := range routing.VRDevices(cfg, state.Snapshot).Devices {
				if d.Available && d.Direction == "output" && d.Driver == "wdm" {
					for _, h := range cfg.VR.Headsets {
						if h.PlaybackRegex != nil && h.PlaybackRegex.MatchString(d.Name) {
							state.VRPlayback = "Available"
							state.VRPlaybackAvailable = true
						}
					}
				}
			}
		}
		state.Intent = cfg.VoiceIntent()
		state.ActiveIntent = routing.ProfileConfig(cfg, state.Snapshot).VoiceIntent()
		state.VRSourceOptions = []string{"auto", "off"}
		if cfg.VR != nil {
			for _, h := range cfg.VR.Headsets {
				state.VRSourceOptions = append(state.VRSourceOptions, "vr:"+h.ID)
			}
			vrSnapshot := state.Snapshot
			vrSnapshot.SteamVR = &model.ProcessStatus{Known: true, Running: true}
			vrConfig := cfg
			vrConfig.ProfileRunning = true
			if cfg.Policy != nil {
				if policy := cfg.Policy(); policy != nil {
					studio := *cfg.Studio
					studio.Playback = slices.Clone(studio.Playback)
					for _, candidate := range policy.Playback {
						if candidate.Driver != "normal" {
							studio.Playback = append(studio.Playback, candidate)
						}
					}
					vrConfig.Studio = &studio
				}
			}
			state.VRSourceOptions = append(state.VRSourceOptions, "desk", "lav", "webcam")
			state.VROutputOptions = routing.PlaybackOptions(vrConfig, vrSnapshot)
		}
		if cfg.Profiles != nil {
			profile := "Normal"
			if cfg.Policy != nil {
				if p := cfg.Policy(); p != nil {
					state.VRConfigured = true
					if p.Running {
						profile = "VR"
					}
				}
			}
			if state.Profile != profile {
				revision++
				state.Profile = profile
			}
		}
		if confirmRevision != revision {
			state.RestartConfirmation = false
		}
		state.Revision = revision
		state.StateError = cfg.StateError
		emit()
	}
	var meterTicks <-chan time.Time
	if deps.Meters {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		meterTicks = ticker.C
	}
	delay := time.Duration(0)
	lastInventory := time.Time{}
	for {
		if deps.Prepare != nil {
			cfg = deps.Prepare(cfg)
		}
		pollAt := time.Now().Add(delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-meterTicks:
			timer.Stop()
			sampleLevels()
			emit()
			delay = time.Until(pollAt)
			continue
		case action := <-actions:
			if ctx.Err() != nil {
				timer.Stop()
				return
			}
			timer.Stop()
			if action.Origin != "" && action.ID != 0 {
				if seen[action.Origin] >= action.ID {
					publish()
					continue
				}
				if len(seen) >= 16 && seen[action.Origin] == 0 {
					continue
				}
				seen[action.Origin] = action.ID
			}
			previousEditError := state.EditError
			if action.Origin != "" {
				state.SetNotice("", NoticeSuccess, time.Now())
			}
			if action.Origin != "" {
				state.EditError = ""
			}
			noticeRevision := state.noticeRevision
			state.actionFeedback(action, NoticePending, time.Now())
			switch action.Kind {
			case restartEngine:
				if action.Revision != revision || (action.Confirm && (!state.RestartConfirmation || confirmRevision != revision)) {
					state.SetNotice("Stale restart command; inspect status and try again", NoticeError, time.Now())
					break
				}
				if !state.Live {
					state.SetNotice("Preview: engine restart not sent", NoticeSuccess, time.Now())
					break
				}
				state.RestartConfirmation = false
				if ctl == nil || backend == nil {
					state.SetNotice("Audio unavailable", NoticeError, time.Now())
					break
				}
				recorder, err := backend.Recorder()
				if err != nil || recorder.State() != "Stopped" {
					if !action.Confirm {
						state.RestartConfirmation = true
						confirmRevision = revision
						state.SetNotice("Restart interrupts audio and may interrupt recording; select Confirm restart", NoticeError, time.Now())
						break
					}
				}
				err = recovery.restart(backend, state.Live, time.Now())
				if err != nil {
					state.SetNotice(err.Error(), NoticeError, time.Now())
				} else {
					revision++
					state.SetNotice("Audio engine restart submitted", NoticePending, time.Now())
				}
			case gain:
				var err error
				if recovery.pending || action.Revision != revision {
					err = fmt.Errorf("gain pending or target revision changed")
				} else if ctl == nil {
					err = fmt.Errorf("audio unavailable")
				} else {
					if action.ResetGain {
						err = ctl.ResetGain(ctx, action.Target, action.Identity, state.Live)
					} else {
						err = ctl.Gain(ctx, action.Target, action.Identity, action.Delta, state.Live)
					}
				}
				if err != nil {
					state.SetNotice(err.Error(), NoticeError, time.Now())
				}

			case startRecording, stopRecording, playSnippet:
				if recovery.pending {
					state.SetNotice("Audio recovery in progress; transport not submitted", NoticeError, time.Now())
					break
				}
				if action.Revision != revision {
					state.SetNotice("Stale recording command; inspect status and try again", NoticeError, time.Now())
					break
				}
				if ctl == nil {
					state.SetNotice("Recorder unavailable", NoticeError, time.Now())
					break
				}
				state.SetNotice("Recorder pending", NoticePending, time.Now())
				publish()
				var e error
				if action.Kind == playSnippet {
					e = ctl.PlaySnippet(ctx, state.Live)
				} else {
					e = ctl.Record(ctx, action.Kind == startRecording, state.Live)
				}
				if e != nil {
					state.SetNotice("Recorder: "+e.Error(), NoticeError, time.Now())
				} else {
					state.SetNotice("Recorder verified", NoticeSuccess, time.Now())
				}
				revision++
			case toggleLive:
				revision++
				setLive(!state.Live)
				reset()
			case reload:
				updated, e := deps.Load(path)
				if e == nil && updated.StateError != "" {
					e = fmt.Errorf("%s", updated.StateError)
				}
				if e == nil && backend != nil && backend.snapshot.Edition != 0 {
					e = updated.ValidateEdition(backend.snapshot.Edition)
				}
				if e != nil {
					state.SetNotice("Config reload failed: "+e.Error(), NoticeError, time.Now())
				} else {
					if e := revokeDefaults(); e != nil {
						state.SetNotice(e.Error(), NoticeError, time.Now())
						break
					}
					cfg = updated
					revision++
					state.SetNotice("Reloaded", NoticeSuccess, time.Now())
					reset()
				}
			case editRule, resetChoices:
				if action.ID != 0 && action.Origin == "" {
					state.EditAck = action.ID
					state.EditError = ""
				}
				if action.Revision != revision {
					state.EditError = "Stale rule command; try again"
					state.SetNotice("Stale rule command; try again", NoticeError, time.Now())
					break
				}
				next := cfg.VoiceIntent()
				if next == nil {
					state.EditError = "No voice profile configured"
					state.SetNotice("No voice profile configured", NoticeError, time.Now())
					break
				}
				var e error
				candidateConfig := cfg
				if action.Kind == resetChoices {
					base, err := config.LoadEffective(path)
					e = err
					if e == nil {
						base.Intent = nil
						next = base.VoiceIntent()
						candidateConfig = base
						if backend != nil && backend.snapshot.Edition != 0 {
							e = base.ValidateEdition(backend.snapshot.Edition)
						}
					}
				} else if cfg.StateError != "" {
					e = fmt.Errorf("reset or repair saved choices first")
				} else {
					e = EditBatch(next, action)
				}
				if e == nil {
					if next == nil {
						e = fmt.Errorf("reset configuration has no voice profile; reload it first")
					} else {
						e = next.Validate(candidateConfig)
					}
				}
				if e == nil {
					if action.Kind == editRule && ((EditsRow(action, "source") && next.Source != "off") || (EditsRow(action, "output") && next.PlaybackDevice != "")) {
						if backend == nil {
							e = fmt.Errorf("device observation unavailable")
						} else {
							var snapshot model.Snapshot
							snapshot, e = backend.Snapshot()
							if e == nil && EditsRow(action, "source") && !slices.Contains(routing.MicrophoneOptions(candidateConfig, snapshot), next.Source) {
								e = fmt.Errorf("microphone is no longer connected")
							}
							if e == nil && EditsRow(action, "output") && !slices.Contains(routing.PlaybackOptions(candidateConfig, snapshot), next.PlaybackDevice) {
								e = fmt.Errorf("playback device is no longer connected")
							}
						}
					}
				}
				if e == nil {
					var token string
					token, e = deps.Save(path, candidateConfig, next, candidateConfig.StateToken)
					if e == nil {
						var revokeErr error
						if old := cfg.VoiceIntent(); old != nil && old.ProtectDefaults && !next.ProtectDefaults {
							revokeErr = revokeDefaults()
						}
						cfg = candidateConfig
						cfg.Intent = next
						cfg.StateToken = token
						cfg.StateError = ""
						revision++
						reset()
						if state.Live {
							state.SetNotice("Saved · Pending", NoticePending, time.Now())
						} else {
							state.SetNotice("Saved · Preview", NoticeSuccess, time.Now())
						}
						if revokeErr != nil {
							state.SetNotice("Saved; "+revokeErr.Error(), NoticeError, time.Now())
						}
					}
				}
				if e != nil {
					state.EditError = e.Error()
					state.SetNotice("Rule change rejected: "+e.Error(), NoticeError, time.Now())
				}
			case refresh:
			}
			resultKind := NoticeSuccess
			if state.noticeRevision != noticeRevision && state.NoticeKind == NoticeError {
				resultKind = NoticeError
			}
			state.actionFeedback(action, resultKind, time.Now())
			if action.Origin != "" {
				next := map[string]Ack{}
				for k, v := range state.Acks {
					next[k] = v
				}
				actionError := state.EditError
				if state.NoticeKind == NoticeError && state.Notice != "" {
					actionError = state.Notice
				}
				next[action.Origin] = Ack{ID: action.ID, Error: actionError}
				state.EditError = previousEditError
				state.Acks = next
			}
		case <-timer.C:
		}
		if ctx.Err() != nil {
			return
		}
		if backend == nil {
			client, e := deps.Open(dll)
			if e != nil {
				state.Error = e.Error()
				state.Connected = false
				publish()
				delay = 5 * time.Second
				continue
			}
			backend = &observed{Client: client}
			reset()
			if e := (controller.RealClock{}).Wait(ctx, 50*time.Millisecond); e != nil {
				return
			}
		}
		// Stall detection runs whenever the live owner holds audio; Auto-recover only
		// gates automatic dispatch. Preview never registers the callback.
		monitorWanted := state.Live
		autoRecover := cfg.VoiceIntent() != nil && cfg.VoiceIntent().AutoRecover
		monitorErr := configureMonitor(monitorWanted)
		state.Error = ""
		if recovery.pending {
			_, err := backend.ParameterSnapshot()
			delay = 100 * time.Millisecond
			if err != nil {
				state.Error = err.Error()
			}
		} else {
			ctl.Config = cfg
			ctl.FastObservation = !lastInventory.IsZero() && time.Since(lastInventory) < time.Second
			if !ctl.FastObservation {
				lastInventory = time.Now()
			}
			delay = ctl.Step(ctx, state.Live)
			state.Error = ctl.Error
		}
		if ctx.Err() != nil {
			return
		}
		state.Snapshot = backend.snapshot
		healthSnapshot := state.Snapshot
		if backend.readError != nil {
			healthSnapshot = model.Snapshot{}
		}
		fault := false
		callbackMessage := ""
		if monitorWanted {
			fault, callbackMessage = recovery.callback.update(cfg, healthSnapshot, time.Now())
			if monitorErr != nil {
				fault = false
				callbackMessage = "Callback monitor unavailable: " + monitorErr.Error()
			}
		} else {
			recovery.callback = callbackHealth{}
		}
		wasPending := recovery.pending
		state.Health = recovery.observe(healthSnapshot, time.Now())
		state.Stalled = monitorWanted && monitorErr == nil && fault
		if monitorWanted && !wasPending {
			state.Health = callbackMessage
			if fault && !recovery.pending && !autoRecover {
				state.Health += " · restart required"
			}
			if fault && !recovery.pending && autoRecover {
				if err := recovery.automaticRestart(backend, cfg, state.Live, time.Now()); err != nil {
					state.Health += " · Auto: " + err.Error()
				} else {
					revision++
					state.Health = recovery.status
					state.SetNotice("Automatic audio engine restart submitted", NoticePending, time.Now())
				}
			}
		}
		if monitorErr != nil {
			state.Health = "Callback monitor unavailable: " + monitorErr.Error()
		}
		if monitorWanted {
			delay = min(delay, 500*time.Millisecond)
		}
		state.RecoveryOutcome = recovery.status
		state.RecoveryPending = recovery.pending
		state.Snapshot = backend.snapshot
		if state.Recorder.State() != backend.snapshot.Recorder.State() {
			revision++
		}
		state.Recorder = backend.snapshot.Recorder
		// Routing/write diagnostics do not mean the native observation was lost.
		state.Connected = backend.readError == nil
		if describer, ok := backend.Client.(interface{ RemoteInfo() model.RemoteInfo }); ok {
			state.Remote = describer.RemoteInfo()
		}
		if !state.Connected {
			revision++
			state.Recorder = &model.RecorderSnapshot{Error: backend.readError.Error()}
		}
		state.Plan = nil
		if state.Connected {
			p, e := routing.Build(cfg, state.Snapshot)
			if e != nil {
				state.Error = e.Error()
			} else {
				state.Plan = &p
			}
		}
		state.ResolveNotice(time.Now())
		publish()
	}
}

type Ack struct {
	ID    uint64
	Error string
}

// Exported aliases preserve one authoritative action numbering for UI and IPC.
const ToggleLiveKind = toggleLive
const ReloadKind = reload
const RefreshKind = refresh
const ResetChoices = resetChoices
const Edit = editRule
const Gain = gain
const Restart = restartEngine
const RecordStart = startRecording
const RecordStop = stopRecording
const SnippetPlay = playSnippet

func (o *observed) SetMixer(p string, v float32) error {
	b, ok := o.Client.(controller.MixerBackend)
	if !ok {
		return fmt.Errorf("mixer unavailable")
	}
	return b.SetMixer(p, v)
}
func (o *observed) RestartEngine() error {
	b, ok := o.Client.(interface{ RestartEngine() error })
	if !ok {
		return fmt.Errorf("engine restart unavailable")
	}
	return b.RestartEngine()
}
