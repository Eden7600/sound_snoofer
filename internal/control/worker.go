package control

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/controller"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/voicemeeter"
	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
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
	// Insert returns the wanted audio callback insert stages (or nil) and the
	// request's generation. The worker reads it each step, applies the stages
	// while live, and reports the outcome in State.Insert.
	Insert func() (*voicemeeter.InsertHook, uint64)
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
	// ObservedAt advances only on successful native snapshots.
	ObservedAt          time.Time
	ChoiceLabels        map[string]string
	DefaultKind         windowsaudio.StatusKind
	DefaultsDetail      windowsaudio.Result // Latest full defaults observation, for diagnostics.
	Remote              model.RemoteInfo    // Native Remote API identity, for diagnostics.
	Stalled             bool                // A qualified callback stall: the engine is not processing audio.
	RecoveryOutcome     string
	RecoveryPending     bool
	Acks                map[string]Ack
	Health              string
	Defaults            string
	RestartConfirmation bool

	// Insert is the insert hook the callback is registered with (nil when none)
	// after applying request InsertGeneration. A failed change leaves both as
	// they were, so a removed hook is released only once Insert is nil at the
	// removal's generation or later.
	Insert           *voicemeeter.InsertHook
	InsertGeneration uint64

	// SilentMics are microphone options latched silent by activity metering.
	SilentMics    []string
	MicOptions    []string
	OutputOptions []string
	EditAck       uint64
	EditError     string
	Recorder      *model.RecorderSnapshot
	TapePaused    bool // Snoofer's stop paused playback; see Controller.TapePaused.
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

	// Generations are separate only for the two global mute preferences.
	MicMuteGeneration, PlaybackMuteGeneration uint64
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
	tape // Target names the command: play, stop, rew or ff.
)

type Action struct {
	// Complete reports the worker boundary; native observation remains in State.
	Complete snoofer.Completion `json:"-"`

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

	MuteGeneration uint64
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

// restartCallback restarts the live monitor after the engine ends or changes its stream.
func restartCallback(client Client) error {
	if c, ok := client.(interface{ RestartCallback() error }); ok {
		return c.RestartCallback()
	}
	return fmt.Errorf("callback monitoring unsupported by this backend")
}

func setCallback(client Client, enable bool, insert *voicemeeter.InsertHook) error {
	if c, ok := client.(interface {
		SetCallback(bool, *voicemeeter.InsertHook) error
	}); ok {
		return c.SetCallback(enable, insert)
	}
	if enable {
		return fmt.Errorf("callback monitoring unsupported by this backend")
	}
	return nil
}

// The actor exclusively owns DLL calls, controller state and writer ownership.
// State messages contain fresh snapshots; no shared mutable maps reach the UI.
func Work(ctx context.Context, cfg config.Config, path, dll string, live bool, deps Dependencies, actions <-chan Action, states chan State, done chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)
	defer close(states)
	defer func() {
		for {
			select {
			case action, ok := <-actions:
				if !ok {
					return
				}
				action.finish(snoofer.Outcome{Status: snoofer.Unknown, Detail: "Audio stopped before completion"})
			default:
				return
			}
		}
	}()
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
	// planning is cfg with device identities resolved to the names devices
	// have now; edits and saves always use cfg itself.
	planning := func() config.Config {
		return cfg.Resolve(DeviceNames(state.DefaultsDetail.Endpoints, state.Snapshot))
	}
	seen := map[string]uint64{}
	if deps.Save == nil {
		deps.Save = config.SaveIntent
	}
	if deps.Load == nil {
		deps.Load = config.LoadEffective
	}
	revision := uint64(1)
	mutes := newMuteGenerations(cfg.VoiceIntent())
	var release func()
	var ctl *controller.Controller
	var backend *observed
	// configureCallback registers the audio callback for monitoring and, while
	// enabled, any wanted insert stages.
	configureCallback := func(enable bool) error {
		var insert *voicemeeter.InsertHook
		var generation uint64
		if deps.Insert != nil {
			insert, generation = deps.Insert()
		}
		if !enable || backend == nil {
			insert = nil
		}
		// Without a backend nothing was ever registered.
		if backend != nil {
			if err := setCallback(backend.Client, enable, insert); err != nil {
				return err
			}
		}
		state.Insert = nil
		if insert != nil {
			copied := *insert
			state.Insert = &copied
		}
		state.InsertGeneration = generation
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
	setLive := func(enable bool) error {
		if enable == state.Live {
			return nil
		}
		if enable {
			if cfg.StateError != "" {
				state.SetNotice(cfg.StateError, NoticeError, time.Now())
				return errors.New(cfg.StateError)
			}
			r, e := deps.Acquire()
			if e != nil {
				state.SetNotice(e.Error(), NoticeError, time.Now())
				return e
			}
			release = r
		} else if release != nil {
			if err := configureCallback(false); err != nil {
				state.SetNotice(err.Error(), NoticeError, time.Now())
				return err
			}
			if err := revokeDefaults(); err != nil {
				state.SetNotice(err.Error(), NoticeError, time.Now())
				return err
			}
			release()
			release = nil
		}
		state.Live = enable
		state.SetNotice("", NoticeSuccess, time.Now())
		return nil
	}
	_ = setLive(live) // Initial failures remain visible in State.
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
	var stream callbackStream
	mixer := &controller.Mixer{Path: path + ".mutes.json"}
	// saveTapeOwnership saves tape-routing ownership before Play first routes the
	// tape to the Playback bus (save before apply). The controller calls it only
	// when Play starts tape playback, so pausing a recording never depends on it.
	saveTapeOwnership := func() error {
		i := cfg.VoiceIntent()
		if i == nil || i.Recording == nil || i.Recording.TapeRoutingManaged {
			return nil
		}
		if cfg.StateError != "" {
			return fmt.Errorf("reset or repair saved choices first")
		}
		i.Recording.TapeRoutingManaged = true
		token, e := deps.Save(path, cfg, i, cfg.StateToken)
		if e != nil {
			return e
		}
		cfg.Intent, cfg.StateToken = i, token
		ctl.Config = planning()
		return nil
	}
	// runTape runs one tape command.
	runTape := func(target string) error {
		command, ok := tapeCommands[target]
		if !ok {
			return fmt.Errorf("unknown tape command %q", target)
		}
		return ctl.Tape(ctx, command, state.Live)
	}
	var confirmRevision uint64
	reset := func() {
		if backend == nil {
			return
		}
		prepared := ctl != nil && ctl.RecorderPrepared
		listening := ctl != nil && ctl.TapeListening
		paused := ctl != nil && ctl.TapePaused
		ctl = &controller.Controller{Mixer: mixer, RecorderPrepared: prepared, TapeListening: listening, TapePaused: paused, BeforeTapePlay: saveTapeOwnership, Backend: backend, Config: cfg, Clock: controller.RealClock{}, Emit: func(e controller.Event) {
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
		state.ChoiceLabels = map[string]string{}
		if cfg.Studio != nil {
			for _, m := range cfg.Studio.Mics() {
				state.ChoiceLabels[m.ID] = m.Name
			}
		}
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
			state.MicOptions = routing.MicrophoneOptions(planning(), state.Snapshot)
			state.OutputOptions = routing.PlaybackOptions(planning(), state.Snapshot)
		}
		state.Intent = cfg.VoiceIntent()
		mutes.update(state.Intent, false)
		state.MicMuteGeneration, state.PlaybackMuteGeneration = mutes.mic, mutes.playback
		state.ActiveIntent = routing.ProfileConfig(planning(), state.Snapshot).VoiceIntent()
		state.VRSourceOptions, state.VROutputOptions = vrOptions(cfg, state.Snapshot)
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
	// Activity metering samples wired microphones on this goroutine, which
	// owns native reads.
	var latch activityLatch
	activityTicker := time.NewTicker(100 * time.Millisecond)
	defer activityTicker.Stop()
	sampleActivity := func() bool {
		var activity *config.Activity
		if cfg.Profiles != nil {
			activity = cfg.Profiles.Activity
		}
		var wired []string
		if state.Plan != nil && state.Plan.Topology != nil {
			wired = meteredMics(cfg, state.Plan.Topology.WiredMics)
		}
		var reader inputLevelReader
		if backend != nil {
			reader, _ = backend.Client.(inputLevelReader)
		}
		if activity == nil || reader == nil || !state.Connected || state.RecoveryPending || len(wired) == 0 {
			changed := len(latch.silent()) > 0
			latch.reset()
			return changed
		}
		strips := []int{}
		for _, id := range wired {
			if strip, ok := micStrip(cfg, id); ok {
				strips = append(strips, strip)
			}
		}
		levels := reader.InputLevels(strips)
		byID := map[string]float32{}
		for _, id := range wired {
			strip, mapped := micStrip(cfg, id)
			if level, ok := levels[strip]; mapped && ok {
				byID[id] = level
			}
		}
		return latch.observe(activity, byID, time.Now())
	}
	var meterTicks <-chan time.Time
	if deps.Meters {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		meterTicks = ticker.C
	}
	delay := time.Duration(0)
	lastInventory := time.Time{}
	for {
		if deps.Prepare != nil {
			cfg = deps.Prepare(cfg)
		}
		mutes.update(cfg.VoiceIntent(), false)
		pollAt := time.Now().Add(delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-activityTicker.C:
			timer.Stop()
			// A changed silent set re-plans now; otherwise keep the schedule.
			delay = time.Until(pollAt)
			if sampleActivity() {
				delay = 0
			}
			continue
		case <-meterTicks:
			timer.Stop()
			sampleLevels()
			emit()
			delay = time.Until(pollAt)
			continue
		case action := <-actions:
			if ctx.Err() != nil {
				action.finish(snoofer.Outcome{Status: snoofer.Unknown, Detail: "Audio stopped before completion"})
				timer.Stop()
				return
			}
			timer.Stop()
			if action.Origin != "" && action.ID != 0 {
				if seen[action.Origin] >= action.ID {
					action.finish(snoofer.Outcome{Status: snoofer.Rejected, Detail: "Duplicate audio command"})
					publish()
					continue
				}
				if len(seen) >= 16 && seen[action.Origin] == 0 {
					action.finish(snoofer.Outcome{Status: snoofer.Rejected, Detail: "Audio origin capacity full"})
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
			result := snoofer.Outcome{Status: snoofer.Processed, Detail: "Processed"}
			reject := func(detail string) {
				result = snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}
				state.SetNotice(detail, NoticeError, time.Now())
			}
			fail := func(err error) {
				result = failedOutcome(err)
				state.SetNotice(err.Error(), NoticeError, time.Now())
			}
			noticeRevision := state.noticeRevision
			state.actionFeedback(action, NoticePending, time.Now())
			switch action.Kind {
			case restartEngine:
				if action.Revision != revision || (action.Confirm && (!state.RestartConfirmation || confirmRevision != revision)) {
					reject("Stale restart command; inspect status and try again")
					break
				}
				if !state.Live {
					result = snoofer.Outcome{Status: snoofer.Rejected, Detail: "Preview: engine restart not sent"}
					state.SetNotice(result.Detail, NoticeSuccess, time.Now())
					break
				}
				state.RestartConfirmation = false
				if ctl == nil || backend == nil {
					reject("Audio unavailable")
					break
				}
				recorder, err := backend.Recorder()
				if err != nil || recorder.State() != "Stopped" {
					if !action.Confirm {
						state.RestartConfirmation = true
						confirmRevision = revision
						reject("Restart interrupts audio and may interrupt recording; select Confirm restart")
						break
					}
				}
				err = recovery.restart(backend, state.Live, time.Now())
				if err != nil {
					fail(err)
				} else {
					revision++
					result = snoofer.Outcome{Status: snoofer.Submitted, Detail: "Audio engine restart submitted"}
					state.SetNotice(result.Detail, NoticePending, time.Now())
				}
			case gain:
				if recovery.pending || action.Revision != revision {
					reject("gain pending or target revision changed")
					break
				}
				if ctl == nil {
					reject("Audio unavailable")
					break
				}
				var err error
				if action.ResetGain {
					err = ctl.ResetGain(ctx, action.Target, action.Identity, state.Live)
				} else {
					err = ctl.Gain(ctx, action.Target, action.Identity, action.Delta, state.Live)
				}
				if err != nil {
					fail(err)
				}

			case startRecording, stopRecording, playSnippet, tape:
				if recovery.pending {
					reject("Audio recovery in progress; transport not submitted")
					break
				}
				if action.Revision != revision {
					reject("Stale recording command; inspect status and try again")
					break
				}
				if ctl == nil {
					reject("Recorder unavailable")
					break
				}
				state.SetNotice("Recorder pending", NoticePending, time.Now())
				publish()
				var e error
				if action.Kind == tape {
					e = runTape(action.Target)
				} else if action.Kind == playSnippet {
					e = ctl.PlaySnippet(ctx, state.Live)
				} else {
					e = ctl.Record(ctx, action.Kind == startRecording, state.Live)
				}
				if e != nil {
					fail(fmt.Errorf("Recorder: %w", e))
				} else {
					state.SetNotice("Recorder verified", NoticeSuccess, time.Now())
				}
				state.TapePaused = ctl.TapePaused
				revision++
			case toggleLive:
				revision++
				if err := setLive(!state.Live); err != nil {
					fail(err)
				}
				reset()
			case reload:
				mutes.update(cfg.VoiceIntent(), true)
				updated, e := deps.Load(path)
				if e == nil && updated.StateError != "" {
					e = fmt.Errorf("%s", updated.StateError)
				}
				if e == nil && backend != nil && backend.snapshot.Edition != 0 {
					e = updated.ValidateEdition(backend.snapshot.Edition)
				}
				if e != nil {
					fail(fmt.Errorf("Config reload failed: %w", e))
				} else {
					if e := revokeDefaults(); e != nil {
						fail(e)
						break
					}
					cfg = updated
					revision++
					result.Detail = "Reloaded"
					state.SetNotice("Reloaded", NoticeSuccess, time.Now())
					reset()
				}
			case editRule, resetChoices:
				if action.ID != 0 && action.Origin == "" {
					state.EditAck = action.ID
					state.EditError = ""
				}
				if (action.MuteGeneration == 0 && action.Revision != revision) || (action.MuteGeneration != 0 && !mutes.matches(action)) {
					state.EditError = "Stale rule command; try again"
					reject("Stale rule command; try again")
					break
				}
				if action.Kind == resetChoices {
					mutes.update(cfg.VoiceIntent(), true)
				}
				next := cfg.VoiceIntent()
				if next == nil {
					state.EditError = "No voice profile configured"
					reject("No voice profile configured")
					break
				}
				var e error
				candidateConfig := cfg
				if action.Kind == resetChoices {
					// The running configuration's source, never path read as a file:
					// plugin hosts pass a state path, not a configuration.
					base, err := deps.Load(path)
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
				if e == nil && action.Kind == editRule {
					// The plugin sends profile-prefixed rows; bare rows remain for older callers.
					source := (EditsRow(action, "source") || EditsRow(action, "normal-source")) && next.Source != "off"
					output := (EditsRow(action, "output") || EditsRow(action, "normal-output")) && next.PlaybackDevice != ""
					vrSource := EditsRow(action, "vr-profile-source") && next.VRProfile != nil && next.VRProfile.Source != "off"
					vrOutput := EditsRow(action, "vr-profile-output") && next.VRProfile != nil && next.VRProfile.Playback != ""
					if source || output || vrSource || vrOutput {
						if backend == nil {
							e = fmt.Errorf("device observation unavailable")
						} else {
							var snapshot model.Snapshot
							snapshot, e = backend.Snapshot()
							var vrSources, vrOutputs []string
							if vrSource || vrOutput {
								vrSources, vrOutputs = vrOptions(candidateConfig, snapshot)
							}
							switch {
							case e != nil:
							case source && !slices.Contains(routing.MicrophoneOptions(candidateConfig, snapshot), next.Source),
								vrSource && !slices.Contains(vrSources, next.VRProfile.Source):
								e = fmt.Errorf("microphone is no longer connected")
							case output && !slices.Contains(routing.PlaybackOptions(candidateConfig, snapshot), next.PlaybackDevice),
								vrOutput && !slices.Contains(vrOutputs, next.VRProfile.Playback):
								e = fmt.Errorf("playback device is no longer connected")
							}
						}
					}
				}
				if e == nil {
					var token string
					token, e = deps.Save(path, candidateConfig, next, candidateConfig.StateToken)
					if e != nil {
						result = failedOutcome(e)
					}
					if e == nil {
						var revokeErr error
						if old := cfg.VoiceIntent(); old != nil && old.ProtectDefaults && !next.ProtectDefaults {
							revokeErr = revokeDefaults()
						}
						cfg = candidateConfig
						cfg.Intent = next
						cfg.StateToken = token
						cfg.StateError = ""
						result.Detail = "Saved"
						revision++
						reset()
						if state.Live {
							state.SetNotice("Saved · Pending", NoticePending, time.Now())
						} else {
							state.SetNotice("Saved · Preview", NoticeSuccess, time.Now())
						}
						if revokeErr != nil {
							fail(fmt.Errorf("Saved; %w", revokeErr))
						}
					}
				}
				if e != nil {
					state.EditError = e.Error()
					if result.Status == snoofer.Processed {
						result = snoofer.Outcome{Status: snoofer.Rejected, Detail: e.Error()}
					}
					state.SetNotice("Rule change rejected: "+e.Error(), NoticeError, time.Now())
				}
			case refresh:
			}
			action.finish(result)
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
		// Every iteration, so a reloaded configuration takes effect.
		ConfigureClient(backend.Client, cfg)
		// Stall detection runs whenever the live owner holds audio; Auto-recover only
		// gates automatic dispatch. Preview never registers the callback.
		monitorWanted := state.Live
		autoRecover := cfg.VoiceIntent() != nil && cfg.VoiceIntent().AutoRecover
		monitorErr := configureCallback(monitorWanted)
		state.Error = ""
		if recovery.pending {
			_, err := backend.ParameterSnapshot()
			delay = 100 * time.Millisecond
			if err != nil {
				state.Error = err.Error()
			}
		} else {
			ctl.Config = planning()
			ctl.SilentMics = latch.silent()
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
		if monitorWanted && monitorErr == nil && backend.readError == nil && stream.restartDue(backend.snapshot.Callback, time.Now()) {
			monitorErr = restartCallback(backend.Client)
		}
		state.Snapshot = backend.snapshot
		healthSnapshot := state.Snapshot
		if backend.readError != nil {
			healthSnapshot = model.Snapshot{}
		}
		fault := false
		callbackMessage := ""
		if monitorWanted {
			fault, callbackMessage = recovery.callback.update(planning(), healthSnapshot, time.Now())
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
				if err := recovery.automaticRestart(backend, planning(), state.Live, time.Now()); err != nil {
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
		if ctl != nil && state.Recorder.State() != "Stopped" {
			ctl.TapePaused = false // Observed elsewhere: no longer paused by Snoofer.
		}
		state.TapePaused = ctl != nil && ctl.TapePaused
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
			planCfg := planning()
			planCfg.TapeListening = ctl != nil && ctl.TapeListening
			planCfg.SilentMics = latch.silent()
			state.SilentMics = planCfg.SilentMics
			p, e := routing.Build(planCfg, state.Snapshot)
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
const Tape = tape

// tapeCommands maps a tape action's Target to its controller command.
var tapeCommands = map[string]controller.TapeCommand{"play": controller.TapePlayPause, "stop": controller.TapeStop, "rew": controller.TapeRewind, "ff": controller.TapeForward}

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

// vrOptions lists the VR profile's selectable sources and playback devices, as
// they would be with SteamVR running and the policy's VR playback candidates.
func vrOptions(cfg config.Config, snapshot model.Snapshot) (sources, outputs []string) {
	sources = []string{"auto", "off"}
	if cfg.VR == nil {
		return sources, nil
	}
	for _, h := range cfg.VR.Headsets {
		sources = append(sources, "vr:"+h.ID)
	}
	snapshot.SteamVR = &model.ProcessStatus{Known: true, Running: true}
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
	sources = append(sources, cfg.Studio.MicrophoneIDs()...)
	return sources, routing.PlaybackOptions(vrConfig, snapshot)
}

func (a Action) finish(result snoofer.Outcome) {
	if a.Complete != nil {
		a.Complete(result)
	}
}

func failedOutcome(err error) snoofer.Outcome {
	status := snoofer.Failed
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		status = snoofer.Unknown
	}
	return snoofer.Outcome{Status: status, Detail: err.Error()}
}
