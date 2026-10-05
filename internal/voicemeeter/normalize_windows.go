//go:build windows && amd64

package voicemeeter

import (
	"context"
	"errors"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// NormalizeClip decodes to a peak-normalized WAV without opening an audio device.
// It blocks its caller; callers must use a cancellable background worker.
func NormalizeClip(ctx context.Context, dllPath, source, destination string) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	dll, err := windows.LoadDLL(dllPath)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, dll.Release()) }()
	normalize, err := dll.FindProc("SBNormalize")
	if err != nil {
		return err
	}
	src, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	cancel, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, windows.CloseHandle(cancel)) }()
	finished := make(chan struct{})
	joined := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			joined <- windows.SetEvent(cancel)
		case <-finished:
			joined <- nil
		}
	}()
	runtime.LockOSThread()
	code, _, _ := normalize.Call(uintptr(unsafe.Pointer(src)), uintptr(unsafe.Pointer(dst)), uintptr(cancel))
	runtime.UnlockOSThread()
	close(finished)
	signalErr := <-joined
	return errors.Join(ctx.Err(), clipResult(code), signalErr)
}
