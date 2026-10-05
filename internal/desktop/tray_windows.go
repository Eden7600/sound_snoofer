//go:build windows

package desktop

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/ownership"
	"sound-snoofer/internal/streamdeck"
	"sound-snoofer/internal/tui"
	"sound-snoofer/internal/voicemeeter"
)

//go:embed tray.ico
var trayIcon []byte

// RunTray owns the audio worker and optional controls process until explicit Quit.
func RunTray(ctx context.Context, args []string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if len(args) > 0 && args[0] == "tray" {
		args = args[1:]
	}
	fs := flag.NewFlagSet("tray", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("config", "", "configuration file")
	dll := fs.String("dll", "", "Remote DLL path")
	dry := fs.Bool("dry-run", false, "preview only")
	live := fs.Bool("apply", true, "apply routing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected tray arguments")
	}
	if *path == "" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		*path = filepath.Join(filepath.Dir(exe), "config.json")
		if err := config.EnsureDefault(*path); err != nil {
			return err
		}
	}
	absolute, err := filepath.Abs(*path)
	if err != nil {
		return err
	}
	handle, existing, err := instance(absolute, *live && !*dry)
	if err != nil {
		return fmt.Errorf("tray instance: %w", err)
	}
	defer windows.CloseHandle(handle)
	if existing {
		return nil
	}
	cfg, err := config.LoadEffective(absolute)
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	finished := make(chan error, 1)
	ready := make(chan struct{})
	startupError := make(chan error, 1)
	watchDone := make(chan struct{})
	defer close(watchDone)
	thread := windows.GetCurrentThreadId()
	go func() {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-ready:
			return
		case <-watchDone:
			return
		case <-timer.C:
			startupError <- errors.New("Windows tray initialization failed; check desktop availability")
			user.NewProc("PostThreadMessageW").Call(uintptr(thread), 0x0012, 0, 0) // WM_QUIT
		}
	}()
	systray.Run(func() {
		close(ready)
		systray.SetIcon(trayIcon)
		systray.SetTooltip("Sound Snoofer · Starting")
		if err := waitForTrayIcon(runCtx, thread); err != nil {
			finished <- err
			systray.Quit()
			return
		}
		status := systray.AddMenuItem("Starting…", "")
		status.Disable()
		systray.AddSeparator()
		open := systray.AddMenuItem("Open controls", "Open the Sound Snoofer TUI")
		quit := systray.AddMenuItem("Quit Sound Snoofer", "Stop automatic audio management")
		finished <- serveTray(runCtx, cancel, cfg, absolute, *dll, *live && !*dry, handle, status, open, quit)
		systray.Quit()
	}, cancel)
	select {
	case err := <-finished:
		return err
	case err := <-startupError:
		return err
	case <-time.After(7 * time.Second):
		return errors.New("tray shutdown timed out")
	}
}

func serveTray(ctx context.Context, cancel context.CancelFunc, cfg config.Config, path, dll string, live bool, handle windows.Handle, status, open, quit *systray.MenuItem) error {
	defer cancel()
	actions, states, workerDone := tui.StartWorker(ctx, cfg, path, dll, live, tui.Dependencies{
		Open:    func(path string) (tui.Client, error) { return voicemeeter.Open(path) },
		Acquire: ownership.Acquire, Load: config.LoadEffective,
	})
	restart := systray.AddMenuItem("Restart audio engine", "Restart audio; controls request confirmation if recording")
	deckStates, deckEvents := streamdeck.Start(ctx, cfg.StreamDeck)
	deckQueue := streamdeck.Queue{}
	latest := tui.State{Live: live, Intent: cfg.VoiceIntent(), StateError: cfg.StateError}
	var child *controls
	var childDone <-chan struct{}
	defer func() {
		if child != nil {
			child.stop()
		}
	}()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	openControls := func() {
		if child != nil {
			select {
			case <-child.done:
				child = nil
				childDone = nil
			default:
				child.focus(latest)
				return
			}
		}
		var err error
		child, err = startControls(ctx, latest, actions)
		if err != nil {
			status.SetTitle("Controls failed — try again")
			ShowError(err)
			return
		}
		childDone = child.done
	}
	lastStatus := ""
	controlsError := ""
	for {
		select {
		case <-ctx.Done():
			select {
			case <-workerDone:
				return nil
			case <-time.After(3 * time.Second):
				return errors.New("audio worker shutdown timed out")
			}
		case <-quit.ClickedCh:
			cancel()
		case <-restart.ClickedCh:
			select {
			case actions <- tui.Action{Kind: control.Restart, Revision: latest.Revision}:
			default:
				status.SetTitle("Audio command queue full")
			}
		case event, ok := <-deckEvents:
			if !ok {
				deckEvents = nil
				continue
			}
			if event.Error != "" {
				status.SetTitle("Stream Deck: " + event.Error)
				continue
			}
			_, command := streamdeck.Action(event, latest)
			switch command {
			case "audio":
				if err := deckQueue.Push(event, latest); err != nil {
					status.SetTitle(err.Error())
				}
				if next, ok := deckQueue.Next(latest); ok {
					select {
					case actions <- next:
					default:
						deckQueue.Rejected()
						status.SetTitle("Stream Deck command queue full")
					}
				}
			case "open-controls":
				openControls()
			case "media-next", "media-prev", "media-play", "media-stop":
				if latest.Live {
					if err := streamdeck.Media(command); err != nil {
						status.SetTitle(err.Error())
					}
				}
			}
		case <-open.ClickedCh:
			controlsError = ""
			openControls()
		case <-childDone:
			if child.exitErr != nil {
				controlsError = "Controls exited unexpectedly — reopen controls"
				status.SetTitle(controlsError)
			}
			child = nil
			childDone = nil
		case <-ticker.C:
			if !latest.ObservedAt.IsZero() && time.Since(latest.ObservedAt) > 5*time.Second {
				status.SetTitle("Audio worker stalled — open controls")
			}
			signaled, err := windows.WaitForSingleObject(handle, 0)
			if err != nil {
				return fmt.Errorf("tray instance signal: %w", err)
			}
			if signaled == windows.WAIT_OBJECT_0 {
				controlsError = ""
				openControls()
			}
		case state, ok := <-states:
			if !ok {
				return nil
			}
			latest = state
			if next, ok := deckQueue.Next(latest); ok {
				select {
				case actions <- next:
				default:
					deckQueue.Rejected()
					status.SetTitle("Stream Deck command queue full")
				}
			}
			if state.RestartConfirmation {
				openControls()
			}
			select {
			case <-deckStates:
			default:
			}
			select {
			case deckStates <- state:
			default:
			}
			label := statusText(state)
			if controlsError != "" {
				label = controlsError
			}
			if label != lastStatus {
				status.SetTitle(label)
				systray.SetTooltip("Sound Snoofer · " + label)
				lastStatus = label
			}
			if child != nil {
				if err := child.publish(state); err != nil {
					return err
				}
			}
		}
	}
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
