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

	"sound-snoofer/snoofer"
)

//go:embed tray.ico
var trayIcon []byte

// Run composes the supplied plugins and owns the desktop lifecycle.
func Run(ctx context.Context, args []string, plugins ...snoofer.Plugin) error {
	if len(args) == 1 && args[0] == "__controls" {
		return RunControls(ctx)
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
	host := snoofer.New(cfg, snoofer.Services{Controls: registry, Path: absolute, Live: !*dry}, plugins...)
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
		var proposed *snoofer.Config
		expected := ""
		openUI := func() {
			if child != nil {
				select {
				case <-child.done:
					child = nil
				default:
					child.focus(latest)
					return
				}
			}
			var e error
			child, e = startControls(runCtx, latest, actions)
			if e != nil {
				notice = e.Error()
				ShowError(e)
			}
		}
		_ = registry.Publish("core", []snoofer.Control{{ID: "core.open-controls", SurfaceOnly: true, Label: "Controls", Group: "System", Kind: "command", Available: true, Icon: "open-controls", Operations: []string{"press"}}}, func(ctx context.Context, r snoofer.Request) error {
			select {
			case actions <- UIAction{Kind: "open"}:
				return nil
			default:
				return fmt.Errorf("controls busy")
			}
		})
		publish := func() {
			current, _, e := snoofer.Load(absolute)
			if e != nil {
				notice = e.Error()
				current = cfg
			}
			latest = ViewState{Controls: registry.Snapshot(), Plugins: host.Status(), Enabled: map[string]bool{}, Notice: notice}
			for _, p := range plugins {
				if _, ok := latest.Plugins[p.ID]; !ok {
					latest.Plugins[p.ID] = "Disabled"
				}
			}
			for id, p := range current.Plugins {
				latest.Enabled[id] = p.Enabled
			}
			if proposed != nil {
				latest.Confirmation = confirmation(current, *proposed)
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
				if a.Request != nil {
					if e := registry.Dispatch(runCtx, *a.Request); e != nil {
						notice = e.Error()
					}
					break
				}
				switch a.Kind {
				case "open":
					openUI()
				case "retry":
					host.Retry()
				case "selection":
					current, rev, e := snoofer.Load(absolute)
					if e != nil {
						notice = e.Error()
						break
					}
					next, e := host.Selection(a.Plugin, a.Enable)
					if e != nil {
						notice = e.Error()
						break
					}
					// Keep live plugin settings; only the enablement closure comes from Host.
					for id, p := range next.Plugins {
						v := current.Plugins[id]
						if len(v.Settings) == 0 {
							v.Settings = p.Settings
						}
						v.Enabled = p.Enabled
						current.Plugins[id] = v
					}
					proposed = &current
					expected = rev
				case "cancel":
					proposed = nil
				case "confirm":
					if proposed != nil {
						if _, e := snoofer.Save(absolute, *proposed, expected); e != nil {
							notice = e.Error()
							proposed = nil
							break
						}
						restart = true
						cancel()
					}
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
	stopErr := stopHost(host)
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
