//go:build windows

package app

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// startupEntry is a per-user Run registration. Windows' startup-apps setting
// (Task Manager or Settings) disables a Run entry by writing a binary value of
// the same name under approvedPath, leaving the Run value in place.
type startupEntry struct {
	runPath, approvedPath, name string
}

var loginStartup = startupEntry{
	runPath:      `Software\Microsoft\Windows\CurrentVersion\Run`,
	approvedPath: `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`,
	name:         "Snoofer",
}

// read returns the observed registration. Missing keys and values mean
// nothing is registered, or nothing is disabled.
func (e startupEntry) read() (startupState, error) {
	command, err := e.readRun()
	if err != nil || command == "" {
		return startupState{}, err
	}
	disabled, err := e.readDisabled()
	if err != nil {
		return startupState{}, err
	}
	return startupState{Command: command, Disabled: disabled}, nil
}

func (e startupEntry) readRun() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, e.runPath, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read login startup: %w", err)
	}
	defer key.Close()
	command, _, err := key.GetStringValue(e.name)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read login startup: %w", err)
	}
	return command, nil
}

func (e startupEntry) readDisabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, e.approvedPath, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Windows startup apps: %w", err)
	}
	defer key.Close()
	data, _, err := key.GetBinaryValue(e.name)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Windows startup apps: %w", err)
	}
	// An odd first byte (0x03, 0x07) is disabled; an even one is enabled.
	return len(data) > 0 && data[0]&1 == 1, nil
}

// apply registers command, or removes the registration, then verifies the
// result by reading it back. Turning it on also clears a Windows disable,
// because the user has explicitly asked for it to run.
func (e startupEntry) apply(enable bool, command string) error {
	var err error
	if enable {
		err = e.register(command)
	} else {
		err = e.deleteValue(e.runPath)
	}
	if err == nil {
		err = e.deleteValue(e.approvedPath)
	}
	if err != nil {
		return fmt.Errorf("change login startup: %w", err)
	}
	observed, err := e.read()
	if err != nil {
		return err
	}
	on := strings.EqualFold(observed.Command, command) && !observed.Disabled
	if on != enable {
		return errors.New("login startup did not change")
	}
	return nil
}

func (e startupEntry) register(command string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, e.runPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(e.name, command)
}

// deleteValue removes this entry's value under path; an absent key or value
// is already the requested state.
func (e startupEntry) deleteValue(path string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(e.name)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}
