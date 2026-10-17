// Package desktopinput sends allowlisted shortcuts without reading the clipboard.
package desktopinput

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Action is one explicit clipboard-pad shortcut.
type Action string

const (
	Copy      Action = "copy"
	Cut       Action = "cut"
	Paste     Action = "paste"
	SelectAll Action = "select-all"
	Enter     Action = "enter"
	Tab       Action = "tab"
	Backspace Action = "backspace"
)

// Target captures a window and its owning process lifetime; Label is an executable basename.
type Target struct {
	Window          uintptr
	Process, Thread uint32
	Started         uint64
	Label           string
}

type windowInfo struct {
	Target
	Integrity uint32
	Visible   bool
}

type keyEvent struct {
	Key uint16
	Up  bool
}

// system is the small OS boundary used by the fake-only tests.
type system interface {
	environment() (process, integrity uint32, err error)
	foreground() uintptr
	window(uintptr) (windowInfo, error)
	activate(uintptr) bool
	down(uint16) bool
	inject([]keyEvent) (int, error)
	pause(context.Context, time.Duration) error
}

func sequence(action Action) ([]keyEvent, error) {
	var key uint16
	ctrl := true
	switch action {
	case Copy:
		key = 'C'
	case Cut:
		key = 'X'
	case Paste:
		key = 'V'
	case SelectAll:
		key = 'A'
	case Enter:
		key, ctrl = 0x0d, false
	case Tab:
		key, ctrl = 0x09, false
	case Backspace:
		key, ctrl = 0x08, false
	default:
		return nil, errors.New("Unknown clipboard shortcut")
	}
	if ctrl {
		return []keyEvent{{Key: 0xa2}, {Key: key}, {Key: key, Up: true}, {Key: 0xa2, Up: true}}, nil
	}
	return []keyEvent{{Key: key}, {Key: key, Up: true}}, nil
}

func inspect(api system, hwnd uintptr) (Target, error) {
	if hwnd == 0 {
		return Target{}, errors.New("Choose a desktop application")
	}
	self, integrity, err := api.environment()
	if err != nil {
		return Target{}, err
	}
	window, err := api.window(hwnd)
	if err != nil {
		return Target{}, fmt.Errorf("Target unavailable: %w", err)
	}
	if window.Window != hwnd || window.Process == 0 || window.Thread == 0 || window.Started == 0 || !window.Visible || window.Label == "" {
		return Target{}, errors.New("Target window is unavailable")
	}
	if window.Process == self {
		return Target{}, errors.New("Choose a desktop application")
	}
	switch strings.ToLower(window.Label) {
	case "snoofer.exe", "vrmonitor.exe", "vrdashboard.exe", "vrcompositor.exe", "vrserver.exe", "vrwebhelper.exe":
		return Target{}, errors.New("Choose a desktop application")
	}
	if window.Integrity > integrity {
		return Target{}, errors.New("Target requires higher privileges")
	}
	return window.Target, nil
}

func capture(api system) (Target, error) {
	target, err := inspect(api, api.foreground())
	if err != nil {
		return Target{}, err
	}
	if api.foreground() != target.Window {
		return Target{}, errors.New("Focused application changed; try again")
	}
	return target, nil
}

func released(api system, keys []keyEvent) error {
	for _, key := range []uint16{0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0x5b, 0x5c} {
		if api.down(key) {
			return errors.New("Release Ctrl, Alt, Shift and Windows keys")
		}
	}
	for _, key := range keys {
		if !key.Up && key.Key != 0xa2 && api.down(key.Key) {
			return errors.New("Release the shortcut key")
		}
	}
	return nil
}

func send(ctx context.Context, api system, target Target, action Action) error {
	keys, err := sequence(action)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	current, err := inspect(api, target.Window)
	if err != nil {
		return err
	}
	if current != target {
		return errors.New("Target application changed; choose it again")
	}
	if err = released(api, keys); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if before := api.foreground(); before != target.Window {
		if !api.activate(target.Window) {
			return errors.New("Windows could not focus the target; select it on the desktop")
		}
		for attempt := 0; ; attempt++ {
			if err = ctx.Err(); err != nil {
				return err
			}
			focused := api.foreground()
			if focused == target.Window {
				break
			}
			if attempt == 5 || (focused != 0 && focused != before) {
				return errors.New("Focused application changed; try again")
			}
			if err = api.pause(ctx, 10*time.Millisecond); err != nil {
				return err
			}
		}
	}
	current, err = inspect(api, target.Window)
	if err != nil {
		return err
	}
	if current != target {
		return errors.New("Target application changed; choose it again")
	}
	if err = released(api, keys); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	// shortcut: SendInput is not HWND-bound; use app-specific APIs if atomic delivery becomes necessary.
	if api.foreground() != target.Window {
		return errors.New("Focused application changed; try again")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	sent, injectErr := api.inject(keys)
	if sent == len(keys) && injectErr == nil {
		return nil
	}
	problem := fmt.Errorf("Shortcut not confirmed: %d of %d key events sent", sent, len(keys))
	if injectErr != nil {
		problem = fmt.Errorf("%w: %v", problem, injectErr)
	}
	cleanup := releases(keys[:max(0, min(sent, len(keys)))])
	if len(cleanup) != 0 {
		n, cleanupErr := api.inject(cleanup)
		if n != len(cleanup) || cleanupErr != nil {
			problem = fmt.Errorf("%w; key release not confirmed", problem)
		}
	}
	return problem
}

// releases balances only down events in the inserted prefix, never repeats a shortcut.
func releases(inserted []keyEvent) []keyEvent {
	held := map[uint16]bool{}
	for _, event := range inserted {
		held[event.Key] = !event.Up
	}
	var release []keyEvent
	for i := len(inserted) - 1; i >= 0; i-- {
		if key := inserted[i].Key; held[key] {
			release = append(release, keyEvent{Key: key, Up: true})
			held[key] = false
		}
	}
	return release
}
