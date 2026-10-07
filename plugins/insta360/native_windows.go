//go:build windows

package insta360

import (
	"fmt"
	"os"
	"path/filepath"

	"sound-snoofer/internal/camera"
)

// nativeOpener loads snoofer-camera.dll beside the executable on first use
// and keeps it until release. It is used only by the worker's thread.
type nativeOpener struct {
	lib *camera.Library
}

func (o *nativeOpener) open() (device, error) {
	if o.lib == nil {
		exe, err := os.Executable()
		if err != nil {
			return nil, err
		}
		lib, err := camera.Load(filepath.Join(filepath.Dir(exe), "snoofer-camera.dll"))
		if err != nil {
			return nil, fmt.Errorf("camera companion missing or unloadable beside executable: %w", err)
		}
		o.lib = lib
	}
	d, err := o.lib.Open(devicePath)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (o *nativeOpener) release() error {
	if o.lib == nil {
		return nil
	}
	err := o.lib.Release()
	o.lib = nil
	return err
}
