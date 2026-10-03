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
	"voice-snooter/internal/config"
	"voice-snooter/internal/controller"
	"voice-snooter/internal/model"
	"voice-snooter/internal/ownership"
	"voice-snooter/internal/routing"
	"voice-snooter/internal/tui"
	"voice-snooter/internal/voicemeeter"
)

type Client interface {
	controller.Backend
	Close() error
}
type Deps struct {
	Open    func(string) (Client, error)
	Acquire func() (func(), error)
	Clock   controller.Clock
}

func DefaultDeps() Deps {
	return Deps{Open: func(path string) (Client, error) { return voicemeeter.Open(path) }, Acquire: ownership.Acquire, Clock: controller.RealClock{}}
}

const usage = `Voice Snooter - regex-driven Voicemeeter device routing

  voice-snooter tui --config FILE [--apply] [--dll ABSOLUTE_PATH]
  voice-snooter devices [--json] [--dll ABSOLUTE_PATH]
  voice-snooter plan --config FILE [--json] [--dll ABSOLUTE_PATH]
  voice-snooter apply --config FILE [--json] [--dll ABSOLUTE_PATH]
  voice-snooter watch --config FILE [--apply] [--json] [--dll ABSOLUTE_PATH]

Watch is dry-run unless --apply is supplied. Fixed routes manage WDM devices;
studio rules manage ASIO input patches, playback outputs and strip sends.
`

func Run(ctx context.Context, args []string, out, errout io.Writer, deps Deps) (exit int) {
	// Refresh and DLL connection lifetime stay on one Windows thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(out, usage)
		return 0
	}
	command := args[0]
	if command != "devices" && command != "plan" && command != "apply" && command != "watch" && command != "tui" {
		fmt.Fprintf(errout, "unknown command %q\n%s", command, usage)
		return 2
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errout)
	path := fs.String("dll", "", "absolute path to installed 64-bit Remote DLL")
	asJSON := fs.Bool("json", false, "JSON output (watch uses JSON Lines)")
	var configPath string
	var live bool
	if command != "devices" {
		fs.StringVar(&configPath, "config", "", "configuration JSON path (required)")
	}
	if command == "watch" || command == "tui" {
		fs.BoolVar(&live, "apply", false, "apply stable changes (default: dry-run)")
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
	if command != "devices" {
		if configPath == "" {
			fmt.Fprintln(errout, "--config is required")
			return 2
		}
		var e error
		cfg, e = config.LoadEffective(configPath)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 2
		}
	}
	if command != "tui" && cfg.StateError != "" {
		fmt.Fprintln(errout, cfg.StateError)
		return 2
	}
	if command == "tui" {
		if *asJSON {
			fmt.Fprintln(errout, "tui does not support --json; use watch --json")
			return 2
		}
		e := tui.Run(ctx, cfg, configPath, *path, live, out, tui.Dependencies{Open: func(path string) (tui.Client, error) { return deps.Open(path) }, Acquire: deps.Acquire, Load: config.LoadEffective})
		if e != nil {
			fmt.Fprintln(errout, e)
			return 1
		}
		return 0
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
	printJSON := func(v any) { enc := json.NewEncoder(out); enc.SetIndent("", "  "); _ = enc.Encode(v) }
	if command == "devices" {
		s, e := client.Snapshot()
		if e != nil {
			fmt.Fprintln(errout, e)
			return 1
		}
		if *asJSON {
			printJSON(s)
		} else {
			printDevices(out, s)
		}
		return 0
	}
	ctl := controller.Controller{Backend: client, Config: cfg, Clock: deps.Clock}
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
