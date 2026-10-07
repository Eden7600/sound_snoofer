//go:build windows

package camera

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Library is the loaded companion DLL. It may be shared by successive devices.
type Library struct {
	dll                             *windows.DLL
	open, length, get, set, release *windows.Proc
}

// Device is one open camera. All of its calls, and Close, must run on the OS
// thread that opened it, which the caller locks: the DLL owns a COM apartment
// on that thread.
type Device struct {
	lib    *Library
	handle uintptr
}

const (
	sFalse     = 1
	dupName    = 0x80070034 // HRESULT_FROM_WIN32(ERROR_DUP_NAME)
	notFound   = 0x80070490 // HRESULT_FROM_WIN32(ERROR_NOT_FOUND)
	maxControl = 4096
)

func result(name string, code uintptr) error {
	switch uint32(code) {
	case dupName:
		return ErrAmbiguous
	case notFound:
		return ErrNoControl
	}
	if int32(code) < 0 {
		return fmt.Errorf("%s returned 0x%08X", name, uint32(code))
	}
	return nil
}

// Load loads the companion DLL at path.
func Load(path string) (*Library, error) {
	dll, err := windows.LoadDLL(path)
	if err != nil {
		return nil, err
	}
	l := &Library{dll: dll}
	procs := map[string]**windows.Proc{"CamOpen": &l.open, "CamLength": &l.length, "CamGet": &l.get, "CamSet": &l.set, "CamClose": &l.release}
	for name, target := range procs {
		if *target, err = dll.FindProc(name); err != nil {
			return nil, errors.Join(err, dll.Release())
		}
	}
	return l, nil
}

// Release unloads the DLL. Every device must be closed first.
func (l *Library) Release() error {
	return l.dll.Release()
}

// Open opens the single camera whose lowercase device path contains match.
func (l *Library) Open(match string) (*Device, error) {
	text, err := windows.UTF16PtrFromString(match)
	if err != nil {
		return nil, err
	}
	d := &Device{lib: l}
	code, _, _ := l.open.Call(uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(&d.handle)))
	if code == sFalse {
		return nil, ErrAbsent
	}
	if err := result("CamOpen", code); err != nil {
		return nil, err
	}
	return d, nil
}

// Length reports a control's length in bytes.
func (d *Device) Length(set GUID, selector uint32) (int, error) {
	var size uint32
	code, _, _ := d.lib.length.Call(d.handle, uintptr(unsafe.Pointer(&set)), uintptr(selector), uintptr(unsafe.Pointer(&size)))
	return int(size), result("CamLength", code)
}

// Get reads a control at the length the device reports.
func (d *Device) Get(set GUID, selector uint32) ([]byte, error) {
	size, err := d.Length(set, selector)
	if err != nil {
		return nil, err
	}
	if size <= 0 || size > maxControl {
		return nil, fmt.Errorf("camera control length %d", size)
	}
	data := make([]byte, size)
	var returned uint32
	code, _, _ := d.lib.get.Call(d.handle, uintptr(unsafe.Pointer(&set)), uintptr(selector), uintptr(unsafe.Pointer(&data[0])), uintptr(size), uintptr(unsafe.Pointer(&returned)))
	if err := result("CamGet", code); err != nil {
		return nil, err
	}
	return data[:min(int(returned), size)], nil
}

// Set writes a control; the DLL pads or truncates to the device's length.
func (d *Device) Set(set GUID, selector uint32, data []byte) error {
	var first *byte
	if len(data) > 0 {
		first = &data[0]
	}
	code, _, _ := d.lib.set.Call(d.handle, uintptr(unsafe.Pointer(&set)), uintptr(selector), uintptr(unsafe.Pointer(first)), uintptr(len(data)))
	return result("CamSet", code)
}

// Close releases the device.
func (d *Device) Close() error {
	code, _, _ := d.lib.release.Call(d.handle)
	d.handle = 0
	return result("CamClose", code)
}
