//go:build windows

package ownership

import (
	"fmt"
	"golang.org/x/sys/windows"
)

// A named kernel object lives while this handle is held. No thread-affine
// mutex ownership is needed, and a process crash releases the handle.
func Acquire() (func(), error) { return acquire(`Local\VoiceSnooter.Writer.v1`) }
func acquire(name string) (func(), error) {
	p, e := windows.UTF16PtrFromString(name)
	if e != nil {
		return nil, e
	}
	h, e := windows.CreateMutex(nil, false, p)
	if e == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			windows.CloseHandle(h)
		}
		return nil, fmt.Errorf("another Voice Snooter writer is active in this session")
	}
	if e != nil {
		return nil, e
	}
	return func() { windows.CloseHandle(h) }, nil
}
