package tui

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/controller"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

type Client interface {
	controller.Backend
	Close() error
}
type Dependencies struct {
	Save    func(string, config.Config, *config.Intent, string) (string, error)
	Open    func(string) (Client, error)
	Acquire func() (func(), error)
	Load    func(string) (config.Config, error)
}
type State struct {
	Recorder       *model.RecorderSnapshot
	RecorderNotice string
	Intent         *config.Intent
	Revision       uint64
	StateError     string
	Live           bool
	Connected      bool
	Error          string
	Notice         string
	NoticeKind     noticeKind
	NoticeUntil    time.Time
	Updated        time.Time
	Snapshot       model.Snapshot
	Plan           *routing.Plan
	Events         []string
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
)

type Action struct {
	Kind       ActionKind
	Row, Value string
	Revision   uint64
}

var ToggleLive = Action{Kind: toggleLive}
var Reload = Action{Kind: reload}
var Refresh = Action{Kind: refresh}

type observed struct {
	Client
	snapshot model.Snapshot
}

func (o *observed) Snapshot() (model.Snapshot, error) {
	s, e := o.Client.Snapshot()
	if e == nil {
		o.snapshot = s
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
func work(ctx context.Context, cfg config.Config, path, dll string, live bool, deps Dependencies, actions <-chan Action, states chan State, done chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)
	defer close(states)
	state := State{}
	if deps.Save == nil {
		deps.Save = config.SaveIntent
	}
	revision := uint64(1)
	history := []string{}
	log := func(message string) {
		if len(history) > 0 && history[len(history)-1][9:] == message {
			return
		}
		history = append(history, time.Now().Format("15:04:05")+" "+message)
		if len(history) > 50 {
			history = history[len(history)-50:]
		}
	}
	var release func()
	defer func() {
		if release != nil {
			release()
		}
	}()
	setLive := func(enable bool) {
		if enable == state.Live {
			return
		}
		if enable {
			if cfg.StateError != "" {
				state.setNotice(cfg.StateError, noticeError, time.Now())
				return
			}
			r, e := deps.Acquire()
			if e != nil {
				state.setNotice(e.Error(), noticeError, time.Now())
				log("Live mode refused: " + e.Error())
				return
			}
			release = r
		} else if release != nil {
			release()
			release = nil
		}
		state.Live = enable
		state.setNotice("", noticeSuccess, time.Now())
		log(fmt.Sprintf("Live enforcement: %t", enable))
	}
	setLive(live)
	var backend *observed
	defer func() {
		if backend != nil {
			backend.Close()
		}
	}()
	var ctl *controller.Controller
	reset := func() {
		if backend == nil {
			return
		}
		prepared := ctl != nil && ctl.RecorderPrepared
		ctl = &controller.Controller{RecorderPrepared: prepared, Backend: backend, Config: cfg, Clock: controller.RealClock{}, Emit: func(e controller.Event) {
			if e.Kind == "error" {
				state.Error = e.Message
			}
			if e.Kind != "plan" {
				log(e.Kind + ": " + e.Message)
			}
		}}
	}
	publish := func() {
		state.Intent = cfg.VoiceIntent()
		state.Revision = revision
		state.StateError = cfg.StateError
		state.Updated = time.Now()
		state.Events = append([]string{}, history...)
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
	delay := time.Duration(0)
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case action := <-actions:
			timer.Stop()
			switch action.Kind {
			case startRecording, stopRecording:
				if action.Revision != revision {
					state.RecorderNotice = "Stale recording command; inspect status and try again"
					state.setNotice(state.RecorderNotice, noticeError, time.Now())
					log(state.RecorderNotice)
					break
				}
				if ctl == nil {
					state.RecorderNotice = "Recorder unavailable"
					state.setNotice(state.RecorderNotice, noticeError, time.Now())
					log(state.RecorderNotice)
					break
				}
				state.RecorderNotice = "Pending recorder command"
				state.setNotice("Recorder pending", noticeError, time.Now())
				publish()
				e := ctl.Record(ctx, action.Kind == startRecording, state.Live)
				if e != nil {
					state.RecorderNotice = e.Error()
					state.setNotice("Recorder: "+e.Error(), noticeError, time.Now())
				} else {
					state.RecorderNotice = "Recorder command verified (file contents not verified)"
					state.setNotice("Recorder verified", noticeSuccess, time.Now())
				}
				log(state.RecorderNotice)
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
					state.setNotice("Config reload failed: "+e.Error(), noticeError, time.Now())
					log(state.Notice)
				} else {
					cfg = updated
					revision++
					state.setNotice("Reloaded", noticeSuccess, time.Now())
					log(state.Notice)
					reset()
				}
			case editRule, resetChoices:
				if action.Revision != revision {
					state.setNotice("Stale rule command; try again", noticeError, time.Now())
					log(state.Notice)
					break
				}
				next := cfg.VoiceIntent()
				if next == nil {
					state.setNotice("No voice profile configured", noticeError, time.Now())
					log(state.Notice)
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
					e = editIntent(next, action)
				}
				if e == nil {
					if next == nil {
						e = fmt.Errorf("reset configuration has no voice profile; reload it first")
					} else {
						e = next.Validate(candidateConfig)
					}
				}
				if e == nil {
					var token string
					token, e = deps.Save(path, candidateConfig, next, candidateConfig.StateToken)
					if e == nil {
						cfg = candidateConfig
						cfg.Intent = next
						cfg.StateToken = token
						cfg.StateError = ""
						revision++
						reset()
						if state.Live {
							state.setNotice("Saved · Pending", noticePending, time.Now())
						} else {
							state.setNotice("Saved · Preview", noticeSuccess, time.Now())
						}
						log(state.Notice)
					}
				}
				if e != nil {
					state.setNotice("Rule change rejected: "+e.Error(), noticeError, time.Now())
					log(state.Notice)
				}
			case refresh:
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
				log("Connection: " + e.Error())
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
		state.Error = ""
		delay = ctl.Step(ctx, state.Live)
		if ctx.Err() != nil {
			return
		}
		state.Snapshot = backend.snapshot
		if state.Recorder.State() != backend.snapshot.Recorder.State() {
			revision++
		}
		state.Recorder = backend.snapshot.Recorder
		// A successful read may still yield a routing conflict. Do not label the
		// plan healthy on stale data after an error.
		state.Connected = state.Error == ""
		if !state.Connected {
			revision++
			state.Recorder = &model.RecorderSnapshot{Error: state.Error}
		}
		state.Plan = nil
		if state.Connected {
			p, e := routing.Build(cfg, state.Snapshot)
			if e != nil {
				state.Error = e.Error()
				state.Connected = false
				log("Routing: " + e.Error())
			} else {
				state.Plan = &p
			}
		}
		state.resolveNotice(time.Now())
		publish()
	}
}
