package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/controller"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/ownership"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/voicemeeter"
)

type Client interface {
	controller.Backend
	Close() error
}
type Deps struct {
	Load    func(string) (config.Config, error)
	Open    func(string) (Client, error)
	Acquire func() (func(), error)
	Clock   controller.Clock
}

func DefaultDeps() Deps {
	return Deps{Open: func(path string) (Client, error) { return voicemeeter.Open(path) }, Acquire: ownership.Acquire, Clock: controller.RealClock{}}
}

const usage = `Snoofer audio maintenance

  snoofer --config ENVELOPE audio devices [--json] [--dll ABSOLUTE_PATH]
  snoofer --config ENVELOPE audio plan [--json] [--dll ABSOLUTE_PATH]
  snoofer --config ENVELOPE audio apply [--json] [--dll ABSOLUTE_PATH]
  snoofer --config ENVELOPE audio watch [--dry-run] [--json] [--dll ABSOLUTE_PATH]

Open interactive controls from the Snoofer tray. Watch is live by default;
--dry-run opts into preview. Maintenance runs audio without VR process policy.
`

func Run(ctx context.Context, args []string, out, errout io.Writer, deps Deps) (exit int) {
	// Refresh and DLL connection lifetime stay on one Windows thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(out, usage)
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(errout, "choose an audio maintenance command; open interactive controls from the Snoofer tray")
		return 2
	}
	command := args[0]
	if command != "devices" && command != "plan" && command != "apply" && command != "watch" {
		fmt.Fprintf(errout, "unknown command %q\n%s", command, usage)
		return 2
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errout)
	path := fs.String("dll", "", "absolute path to installed 64-bit Remote DLL")
	asJSON := fs.Bool("json", false, "JSON output (watch uses JSON Lines)")
	var configPath string
	var live bool
	var dryRun bool
	if command != "devices" {
		fs.StringVar(&configPath, "config", "", "audio state/configuration path")
	}
	if command == "watch" {
		fs.BoolVar(&live, "apply", true, "apply stable changes (default: true)")
		fs.BoolVar(&dryRun, "dry-run", false, "preview without applying mixer changes")
	}
	if e := fs.Parse(args[1:]); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errout, "unexpected positional arguments")
		return 2
	}
	var cfg config.Config
	if dryRun {
		live = false
	}
	if command != "devices" {
		if configPath == "" {
			fmt.Fprintln(errout, "--config is required")
			return 2
		}
		var e error
		load := deps.Load
		if load == nil {
			load = config.LoadEffective
		}
		cfg, e = load(configPath)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 2
		}
	}
	if cfg.StateError != "" {
		fmt.Fprintln(errout, cfg.StateError)
		return 2
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = runCtx
	output := &checkedWriter{writer: out, cancel: cancel}
	out = output
	defer func() {
		if output.err != nil {
			fmt.Fprintf(errout, "output: %v\n", output.err)
			exit = 1
		}
	}()
	if command == "watch" && cfg.VoiceIntent() != nil && cfg.VoiceIntent().AutoRecover {
		return watchRecovery(ctx, cfg, configPath, *path, live, *asJSON, out, errout, deps)
	}
	if command == "apply" || (command == "watch" && live) {
		release, e := deps.Acquire()
		if e != nil {
			fmt.Fprintln(errout, e)
			return 1
		}
		defer release()
	}
	client, e := deps.Open(*path)
	if e != nil {
		fmt.Fprintln(errout, e)
		return 1
	}
	defer func() {
		if e := client.Close(); e != nil {
			fmt.Fprintln(errout, e)
			if exit == 0 {
				exit = 1
			}
		}
	}()
	// Give the Remote API time to populate its first parameter snapshot after login.
	if e = deps.Clock.Wait(ctx, 50*time.Millisecond); e != nil {
		return 0
	}
	printJSON := func(v any) {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if e := enc.Encode(v); e != nil {
			output.err = e
			cancel()
		}
	}
	if command == "devices" {
		s, e := client.Snapshot()
		if e != nil {
			fmt.Fprintln(errout, e)
			return 1
		}
		s.Devices = model.InventoryDevices(s.Devices)
		if *asJSON {
			printJSON(s)
		} else {
			printDevices(out, s)
		}
		return 0
	}
	ctl := controller.Controller{Backend: client, Config: cfg, Clock: deps.Clock, Mixer: &controller.Mixer{Path: configPath + ".mutes.json"}}
	ctl.Emit = func(e controller.Event) {
		if *asJSON {
			_ = json.NewEncoder(out).Encode(e)
		} else {
			if e.Plan != nil {
				printPlan(out, *e.Plan)
			} else {
				fmt.Fprintf(out, "%s: %s\n", e.Kind, e.Message)
			}
		}
	}
	if command == "watch" {
		if !*asJSON {
			fmt.Fprintf(out, "watch: live=%t; Ctrl+C to stop\n", live)
		}
		e = ctl.Watch(ctx, live)
		if errors.Is(e, context.Canceled) {
			return 0
		}
		if e != nil {
			fmt.Fprintln(errout, e)
			return 1
		}
		return 0
	}
	p, e := ctl.Plan()
	if e != nil {
		fmt.Fprintln(errout, e)
		return 1
	}
	if *asJSON {
		printJSON(p)
	} else {
		printPlan(out, p)
	}
	if command == "apply" {
		if e = ctl.Apply(ctx, p); e != nil {
			fmt.Fprintln(errout, e)
			return 1
		}
	}
	return 0
}
func printDevices(w io.Writer, s model.Snapshot) {
	names := map[int]string{1: "Standard (unsupported for routing)", 2: "Banana", 3: "Potato"}
	name := names[s.Edition]
	if name == "" {
		name = "unknown (unsupported for routing)"
	}
	fmt.Fprintf(w, "Voicemeeter: %s (edition %d)\n", name, s.Edition)
	for _, d := range s.Devices {
		fmt.Fprintf(w, "%-6s %-5s %s\n       id: %s; eligible WDM enumeration: %t\n", d.Direction, d.Driver, d.Name, d.ID, d.Available)
	}
	for _, target := range model.Slots(s.Edition) {
		fmt.Fprintf(w, "%s = %q\n", target, s.Assignments[target])
	}
}
func printPlan(w io.Writer, p routing.Plan) {
	fmt.Fprintf(w, "Plan for edition %d\n", p.Edition)
	if p.Topology != nil {
		fmt.Fprintf(w, "ASIO active=%t; playback=%s\n", p.Topology.ASIOActive, p.Topology.PlaybackTarget)
		for _, op := range p.Topology.Operations {
			if op.Device != nil {
				fmt.Fprintf(w, "%s: %q -> %q (%s); change=%t\n", op.Target, op.BeforeName, op.Device.Name, op.Device.Driver, op.Change)
			} else {
				fmt.Fprintf(w, "%s: %g -> %d; change=%t\n", op.Parameter, op.BeforeValue, op.Value, op.Change)
			}
		}
		for _, reason := range p.Topology.Unresolved {
			fmt.Fprintln(w, "unresolved:", reason)
		}
		return
	}
	for _, d := range p.Decisions {
		desired := "(unresolved; leave unchanged)"
		if d.Desired != nil {
			desired = d.Desired.Name
		}
		fmt.Fprintf(w, "%s: %q -> %q; change=%t\n", d.Target, d.Current, desired, d.Change)
		for _, reason := range d.Reasons {
			fmt.Fprintf(w, "  %s\n", reason)
		}
	}
}

// watchRecovery shares the same owner and recovery policy as the tray/controls.
func watchRecovery(ctx context.Context, cfg config.Config, path, dll string, live, asJSON bool, out, errout io.Writer, deps Deps) int {
	ctx, cancel := context.WithCancel(ctx)
	states := make(chan control.State, 1)
	done := make(chan struct{})
	go control.Work(ctx, cfg, path, dll, live, control.Dependencies{
		Open:    func(path string) (control.Client, error) { return deps.Open(path) },
		Acquire: deps.Acquire, Load: config.LoadEffective,
	}, nil, states, done)
	defer func() { cancel(); <-done }()
	previous := ""
	for s := range states {
		if live && !s.Live {
			fmt.Fprintln(errout, s.Notice)
			return 1
		}
		event := controller.Event{Kind: "health", Message: s.Health, Plan: s.Plan}
		if s.Error != "" {
			event.Kind, event.Message = "error", s.Error
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			fmt.Fprintln(errout, err)
			return 1
		}
		if string(encoded) == previous {
			continue
		}
		previous = string(encoded)
		if asJSON {
			fmt.Fprintln(out, string(encoded))
		} else {
			fmt.Fprintln(out, event.Kind+": "+event.Message)
			if event.Plan != nil {
				printPlan(out, *event.Plan)
			}
		}
	}
	return 0
}
