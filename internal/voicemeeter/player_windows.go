//go:build windows && amd64

package voicemeeter

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ClipPlayer owns a native playback graph. All methods must run on the same
// locked OS thread. The DLL uses a Windows x64 C ABI; HRESULTs are signed int32.
type ClipPlayer struct {
	dll                     *windows.DLL
	handle                  uintptr
	play, stop, poll, close *windows.Proc
}

func clipResult(code uintptr) error {
	if int32(code) < 0 {
		return fmt.Errorf("soundboard native error 0x%08X", uint32(code))
	}
	return nil
}

// OpenClipPlayer loads only the explicitly supplied companion and renderer.
func OpenClipPlayer(path, renderer string) (*ClipPlayer, error) {
	dll, err := windows.LoadDLL(path)
	if err != nil {
		return nil, err
	}
	p := &ClipPlayer{dll: dll}
	var create *windows.Proc
	for name, target := range map[string]**windows.Proc{"SBCreate": &create, "SBPlay": &p.play, "SBStop": &p.stop, "SBPoll": &p.poll, "SBClose": &p.close} {
		*target, err = dll.FindProc(name)
		if err != nil {
			return nil, errors.Join(err, dll.Release())
		}
	}
	device, err := windows.UTF16PtrFromString(renderer)
	if err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	code, _, _ := create.Call(uintptr(unsafe.Pointer(device)), uintptr(unsafe.Pointer(&p.handle)))
	if err = clipResult(code); err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	return p, nil
}
func (p *ClipPlayer) Play(path string, muted bool) error {
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	var quiet uintptr
	if muted {
		quiet = 1
	}
	code, _, _ := p.play.Call(p.handle, uintptr(unsafe.Pointer(file)), quiet)
	return clipResult(code)
}
func (p *ClipPlayer) Stop() error { code, _, _ := p.stop.Call(p.handle); return clipResult(code) }
func (p *ClipPlayer) Poll() (bool, error) {
	code, _, _ := p.poll.Call(p.handle)
	return code == 0, clipResult(code)
}
func (p *ClipPlayer) Close() error {
	code, _, _ := p.close.Call(p.handle)
	return errors.Join(clipResult(code), p.dll.Release())
}
