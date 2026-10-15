//go:build windows

package app

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"

	"sound-snoofer/internal/browserbridge"
	"sound-snoofer/snoofer"
)

//go:embed tray.ico
var trayIcon []byte

// Run composes the supplied plugins and owns the desktop lifecycle.
func Run(ctx context.Context, args []string, plugins ...snoofer.Plugin) error {
	if len(args) == 1 && args[0] == "__controls" {
		return RunDesktop(ctx)
	}
	fs := flag.NewFlagSet("Snoofer", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("config", "", "Snoofer configuration")
	dry := fs.Bool("dry-run", false, "preview only")
	check := fs.Bool("check", false, "validate configuration envelope without starting plugins")
	if len(args) > 0 && args[0] == "tray" {
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *path == "" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		*path = filepath.Join(filepath.Dir(exe), "snoofer.json")
	}
	absolute, err := filepath.Abs(*path)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(absolute); os.IsNotExist(statErr) {
		if _, saveErr := snoofer.Save(absolute, snoofer.Defaults(plugins), "missing"); saveErr != nil {
			return saveErr
		}
	}
	cfg, _, err := snoofer.Load(absolute)
	if err != nil {
		return err
	}
	if fs.NArg() > 0 {
		id := fs.Arg(0)
		for _, p := range plugins {
			if p.ID == id && p.Command != nil {
				if !cfg.Plugins[id].Enabled {
					return fmt.Errorf("plugin %s is disabled", id)
				}
				if p.Validate != nil {
					if err := p.Validate(cfg.Plugins[id].Settings); err != nil {
						return err
					}
				}
				if err := Console(); err != nil {
					return err
				}
				return p.Command(ctx, snoofer.Services{Path: absolute, Live: !*dry}, cfg.Plugins[id].Settings, fs.Args()[1:])
			}
		}
		return fmt.Errorf("unknown plugin command %s", id)
	}
	if *check {
		return snoofer.ValidateEnabled(cfg, plugins...)
	}
	if err = detachConsole(); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	handle, existing, err := instance(absolute, !*dry)
	if err != nil {
		return err
	}
	if handle != 0 {
		defer func() {
			if handle != 0 {
				windows.CloseHandle(handle)
			}
		}()
	}
	if existing {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	registry := snoofer.NewControls()
	browser := browserbridge.New(runCtx, cfg.Plugins["nowplaying"].Settings)
	host := snoofer.New(cfg, snoofer.Services{Controls: registry, Browser: browser, Path: absolute, Live: !*dry}, plugins...)
	host.Start(runCtx)
	finished := make(chan error, 1)
	restart := false
	thread := windows.GetCurrentThreadId()
	ready := make(chan struct{})
	startupError := make(chan error, 1)
	watchDone := make(chan struct{})
	defer close(watchDone)
	go func() {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-ready:
		case <-watchDone:
		case <-timer.C:
			startupError <- errors.New("Windows tray initialization failed; check desktop availability")
			user.NewProc("PostThreadMessageW").Call(uintptr(thread), 0x0012, 0, 0)
		}
	}()
	systray.Run(func() {
		close(ready)
		systray.SetIcon(trayIcon)
		if err := waitForTrayIcon(runCtx, thread); err != nil {
			finished <- err
			systray.Quit()
			return
		}
		systray.SetTooltip("Snoofer")
		status := systray.AddMenuItem("Snoofer running", "")
		status.Disable()
		open := systray.AddMenuItem("Open controls", "Configure Snoofer")
		quit := systray.AddMenuItem("Quit Snoofer", "Release Snoofer resources")
		defer systray.Quit()
		actions := make(chan UIAction, 16)
		var child *controls
		defer func() {
			if child != nil {
				child.stop()
			}
		}()
		tick := time.NewTicker(200 * time.Millisecond)
		defer tick.Stop()
		latest := ViewState{}
		notice := ""
		lifecycle := lifecycle{path: absolute, host: host}
		openUI := func() error {
			if child != nil {
				select {
				case <-child.done:
					child = nil
				default:
					child.focus(latest)
					return nil
				}
			}
			var e error
			child, e = startControls(runCtx, latest, actions)
			if e != nil {
				notice = e.Error()
				ShowError(e)
			}
			return e
		}
		startup := ""
		if exe, e := os.Executable(); e == nil {
			startup = startupCommand(exe, absolute)
		}
		// publishCore re-reads login startup so the toggle shows what Windows
		// holds. Its handler only enqueues the opposite of the published value.
		publishCore := func() {
			observed, readErr := loginStartup.read()
			if startup == "" {
				readErr = errors.New("executable path unavailable")
			}
			startupToggle := startupControl(observed, readErr, startup, !*dry)
			enable := startupToggle.Value != "On"
			openControls := snoofer.Control{ID: "core.open-controls", SurfaceOnly: true, Label: "Controls", Group: "System", Kind: "command", Available: true, Icon: "open-controls", Operations: []string{"press"}}
			_ = registry.Publish("core", []snoofer.Control{openControls, startupToggle}, coreHandler(runCtx, actions, observed, enable))
		}
		publish := func() {
			publishCore()
			current, revision, e := snoofer.Load(absolute)
			if e != nil {
				notice = e.Error()
				current = cfg
			}
			latest = ViewState{Controls: registry.Snapshot(), Plugins: host.Status(), PluginStates: host.Lifecycle(), PluginLabels: map[string]string{"meetings": "Meetings"}, Enabled: map[string]bool{}, Notice: notice, ConfigRevision: revision}
			for _, p := range plugins {
				latest.PluginLabels[p.ID] = p.Label
				if _, ok := latest.Plugins[p.ID]; !ok {
					latest.Plugins[p.ID] = "Disabled"
				}
			}
			for id, p := range current.Plugins {
				latest.Enabled[id] = p.Enabled
			}
			if proposal := lifecycle.pending(); proposal != nil {
				latest.Confirmation, latest.ProposalID = proposal.text, proposal.id
			}
			if child != nil {
				if e := child.publish(latest); e != nil {
					notice = e.Error()
				}
			}
			attention := false
			for _, value := range latest.Plugins {
				if value != "Running" && value != "Disabled" {
					attention = true
				}
			}
			if attention {
				status.SetTitle("Plugin needs attention — open controls")
			} else {
				status.SetTitle("Snoofer running")
			}
		}
		publish()
		for {
			select {
			case <-runCtx.Done():
				finished <- nil
				return
			case <-quit.ClickedCh:
				cancel()
			case <-open.ClickedCh:
				openUI()
			case <-tick.C:
				signaled, e := windows.WaitForSingleObject(handle, 0)
				if e != nil {
					finished <- e
					return
				}
				if signaled == windows.WAIT_OBJECT_0 {
					openUI()
				}
				publish()
			case a := <-actions:
				if a.session != nil && a.session.ctx.Err() != nil {
					break // A departed child cannot leave unstarted work for a new one.
				}
				if a.Request != nil {
					if a.complete != nil {
						_ = registry.DispatchWithResult(runCtx, *a.Request, a.complete)
					} else if e := registry.Dispatch(runCtx, *a.Request); e != nil {
						notice = e.Error()
					}
					break
				}
				var result snoofer.Outcome
				var restartRequested bool
				switch a.Kind {
				case "open":
					result = snoofer.Outcome{Status: snoofer.Processed, Detail: "Controls opened"}
					if err := openUI(); err != nil {
						result = lifecycleFailure(err)
					}
				case "startup":
					result = applyStartup(runCtx, a, !*dry, startup, loginStartup.read, loginStartup.apply)
				default:
					result, restartRequested = lifecycle.handle(a)
				}
				a.reply(result)
				if a.complete == nil && (result.Status == snoofer.Rejected || result.Status == snoofer.Failed || result.Status == snoofer.Unknown) {
					notice = result.Detail
				}
				if restartRequested {
					restart = true
					cancel()
				}
				publish()
			}
		}
	}, cancel)
	var serveErr error
	select {
	case serveErr = <-startupError:
	case serveErr = <-finished:
	case <-time.After(time.Second):
		serveErr = errors.New("tray stopped unexpectedly")
	}
	stopErr := stopHost(host, browser)
	if err = errors.Join(serveErr, stopErr); err != nil {
		return err
	}
	// Release the instance event before the replacement process starts.
	windows.CloseHandle(handle)
	handle = 0
	if restart {
		exe, e := os.Executable()
		if e != nil {
			return e
		}
		command := exec.Command(exe, args...)
		if err := command.Start(); err != nil {
			return err
		}
		return command.Process.Release()
	}
	return nil
}
func waitForTrayIcon(ctx context.Context, thread uint32) error {
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	retry := time.NewTicker(100 * time.Millisecond)
	defer retry.Stop()
	for {
		err := verifyTrayIcon(thread)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return err
		case <-retry.C:
		}
	}
}
