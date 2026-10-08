//go:build windows

package streamdeck

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"strings"
	"time"
	"unsafe"
)

var hid = windows.NewLazySystemDLL("hid.dll")
var cm = windows.NewLazySystemDLL("cfgmgr32.dll")

type device struct {
	handle        windows.Handle
	input, output int
	feature       int // Feature report length; 0 when the device has none.
	serial        string
}

func discover() ([]string, error) {
	var guid windows.GUID
	hid.NewProc("HidD_GetHidGuid").Call(uintptr(unsafe.Pointer(&guid)))
	var count uint32
	r, _, _ := cm.NewProc("CM_Get_Device_Interface_List_SizeW").Call(uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&guid)), 0, 0)
	if r != 0 {
		return nil, fmt.Errorf("HID enumeration failed: %d", r)
	}
	if count < 2 {
		return nil, nil
	}
	if count > 1024*1024 {
		return nil, fmt.Errorf("HID list too large")
	}
	data := make([]uint16, count)
	r, _, _ = cm.NewProc("CM_Get_Device_Interface_ListW").Call(uintptr(unsafe.Pointer(&guid)), 0, uintptr(unsafe.Pointer(&data[0])), uintptr(count), 0)
	if r != 0 {
		return nil, fmt.Errorf("HID list changed: %d", r)
	}
	paths := []string{}
	start := 0
	for n, v := range data {
		if v == 0 {
			if n == start {
				break
			}
			path := windows.UTF16ToString(data[start:n])
			lower := strings.ToLower(path)
			if strings.Contains(lower, "vid_0fd9&pid_00c6") {
				paths = append(paths, path)
			}
			start = n + 1
		}
	}
	return paths, nil
}
func openDevice(path string) (*device, error) {
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return nil, e
	}
	h, e := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
	if e != nil {
		return nil, e
	}
	d := &device{handle: h}
	var preparsed uintptr
	r, _, _ := hid.NewProc("HidD_GetPreparsedData").Call(uintptr(h), uintptr(unsafe.Pointer(&preparsed)))
	if r == 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("HID descriptor unavailable")
	}
	// HIDP_CAPS consists of 32 USHORT fields; byte lengths are fields 2/3/4.
	var caps [32]uint16
	r, _, _ = hid.NewProc("HidP_GetCaps").Call(preparsed, uintptr(unsafe.Pointer(&caps[0])))
	hid.NewProc("HidD_FreePreparsedData").Call(preparsed)
	if uint32(r) != 0x00110000 || caps[2] < 4 || caps[3] < 32 || caps[2] > 8192 || caps[3] > 8192 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("unsupported HID report descriptor")
	}
	d.input = int(caps[2])
	d.output = int(caps[3])
	d.feature = int(caps[4])
	var serial [128]uint16
	r, _, _ = hid.NewProc("HidD_GetSerialNumberString").Call(uintptr(h), uintptr(unsafe.Pointer(&serial[0])), uintptr(unsafe.Sizeof(serial)))
	if r != 0 {
		d.serial = windows.UTF16ToString(serial[:])
	}
	return d, nil
}
func (d *device) close() { windows.CancelIoEx(d.handle, nil); windows.CloseHandle(d.handle) }
func (d *device) io(ctx context.Context, b []byte, write bool) (int, error) {
	event, e := windows.CreateEvent(nil, 1, 0, nil)
	if e != nil {
		return 0, e
	}
	defer windows.CloseHandle(event)
	over := windows.Overlapped{HEvent: event}
	var n uint32
	if write {
		e = windows.WriteFile(d.handle, b, &n, &over)
	} else {
		e = windows.ReadFile(d.handle, b, &n, &over)
	}
	if e == nil {
		return int(n), nil
	}
	if !errors.Is(e, windows.ERROR_IO_PENDING) {
		return 0, e
	}
	deadline := time.Now().Add(time.Second)
	for {
		status, err := windows.WaitForSingleObject(event, 50)
		if err != nil || ctx.Err() != nil || time.Now().After(deadline) {
			windows.CancelIoEx(d.handle, &over)
			windows.GetOverlappedResult(d.handle, &over, &n, true)
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}
			if err != nil {
				return 0, err
			}
			return 0, context.DeadlineExceeded
		}
		if status == windows.WAIT_OBJECT_0 {
			e = windows.GetOverlappedResult(d.handle, &over, &n, false)
			return int(n), e
		}
	}
}

// setBrightness sends the Stream Deck v2 brightness feature report
// [0x03, 0x08, percent], padded to the device's feature report length.
func (d *device) setBrightness(percent int) error {
	if d.feature < 3 {
		return fmt.Errorf("Stream Deck has no feature report for brightness")
	}
	report := make([]byte, d.feature)
	report[0], report[1], report[2] = 0x03, 0x08, byte(min(100, max(0, percent)))
	r, _, e := hid.NewProc("HidD_SetFeature").Call(uintptr(d.handle), uintptr(unsafe.Pointer(&report[0])), uintptr(len(report)))
	if r == 0 {
		return fmt.Errorf("set Stream Deck brightness: %w", e)
	}
	return nil
}
