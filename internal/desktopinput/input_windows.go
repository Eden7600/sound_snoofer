//go:build windows && amd64

package desktopinput

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Capture observes an eligible foreground window without changing focus.
func Capture() (Target, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return capture(native{})
}

// Send submits one shortcut. Success means Windows accepted the keys, not that the app applied them.
func Send(ctx context.Context, target Target, action Action) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return send(ctx, native{}, target, action)
}

type native struct{}

// Lazy procedures are inert until a public operation is explicitly called.
var user32 = windows.NewLazySystemDLL("user32.dll")
var (
	getForegroundWindow      = user32.NewProc("GetForegroundWindow")
	getWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	isWindowVisible          = user32.NewProc("IsWindowVisible")
	isWindowEnabled          = user32.NewProc("IsWindowEnabled")
	setForegroundWindow      = user32.NewProc("SetForegroundWindow")
	getAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	getThreadDesktop         = user32.NewProc("GetThreadDesktop")
	getUserObjectInformation = user32.NewProc("GetUserObjectInformationW")
	sendInput                = user32.NewProc("SendInput")
)

func (native) environment() (uint32, uint32, error) {
	desktop, _, err := getThreadDesktop.Call(uintptr(windows.GetCurrentThreadId()))
	if desktop == 0 {
		return 0, 0, fmt.Errorf("Desktop unavailable: %v", err)
	}
	var receiving, needed uint32
	ok, _, err := getUserObjectInformation.Call(desktop, 6, uintptr(unsafe.Pointer(&receiving)), unsafe.Sizeof(receiving), uintptr(unsafe.Pointer(&needed)))
	if ok == 0 || receiving == 0 {
		return 0, 0, errors.New("Unlock the desktop before sending a shortcut")
	}
	var name [256]uint16
	ok, _, err = getUserObjectInformation.Call(desktop, 2, uintptr(unsafe.Pointer(&name[0])), unsafe.Sizeof(name), uintptr(unsafe.Pointer(&needed)))
	if ok == 0 || !strings.EqualFold(windows.UTF16ToString(name[:]), "Default") {
		return 0, 0, errors.New("Desktop shortcuts are unavailable on this desktop")
	}
	level, err := processIntegrity(windows.CurrentProcess())
	return windows.GetCurrentProcessId(), level, err
}

func (native) foreground() uintptr {
	hwnd, _, _ := getForegroundWindow.Call()
	return hwnd
}

func (native) window(hwnd uintptr) (info windowInfo, err error) {
	var pid uint32
	tid, _, callErr := getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if tid == 0 || pid == 0 {
		return info, fmt.Errorf("Window identity unavailable: %v", callErr)
	}
	visible, _, _ := isWindowVisible.Call(hwnd)
	enabled, _, _ := isWindowEnabled.Call(hwnd)
	if visible == 0 || enabled == 0 {
		return info, errors.New("Window is hidden or disabled")
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return info, err
	}
	defer func() { err = errors.Join(err, windows.CloseHandle(process)) }()
	var started, exited, kernel, user windows.Filetime
	if err = windows.GetProcessTimes(process, &started, &exited, &kernel, &user); err != nil {
		return info, err
	}
	var exitCode uint32
	if err = windows.GetExitCodeProcess(process, &exitCode); err != nil {
		return info, err
	}
	if exitCode != 259 { // STILL_ACTIVE; GetProcessTimes exit time is undefined while running.
		return info, errors.New("Target process exited")
	}
	path := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(path))
	if err = windows.QueryFullProcessImageName(process, 0, &path[0], &size); err != nil {
		return info, err
	}
	level, err := processIntegrity(process)
	if err != nil {
		return info, err
	}
	var checkedPID uint32
	checkedThread, _, _ := getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&checkedPID)))
	if checkedPID != pid || checkedThread != tid {
		return info, errors.New("Window identity changed")
	}
	return windowInfo{Target: Target{Window: hwnd, Process: pid, Thread: uint32(tid), Started: uint64(started.HighDateTime)<<32 | uint64(started.LowDateTime), Label: filepath.Base(windows.UTF16ToString(path[:size]))}, Integrity: level, Visible: true}, nil
}

// Windows token queries require aligned output storage.
type integrityBuffer [32]uint64

func processIntegrity(process windows.Handle) (level uint32, err error) {
	var token windows.Token
	if err = windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return 0, fmt.Errorf("Cannot check target privileges: %w", err)
	}
	defer func() { err = errors.Join(err, token.Close()) }()
	var storage integrityBuffer
	data := unsafe.Slice((*byte)(unsafe.Pointer(&storage[0])), int(unsafe.Sizeof(storage)))
	var size uint32
	if err = windows.GetTokenInformation(token, windows.TokenIntegrityLevel, &data[0], uint32(len(data)), &size); err != nil {
		return 0, fmt.Errorf("Cannot check target privileges: %w", err)
	}
	if size < uint32(unsafe.Sizeof(windows.Tokenmandatorylabel{})) || size > uint32(len(data)) {
		return 0, errors.New("Invalid integrity information")
	}
	label := (*windows.Tokenmandatorylabel)(unsafe.Pointer(&data[0]))
	// The returned SID belongs to this bounded token-information buffer.
	base, sid := uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(label.Label.Sid))
	if sid < base || sid-base > uintptr(size)-12 || label.Label.Attributes&windows.SE_GROUP_INTEGRITY == 0 {
		return 0, errors.New("Invalid integrity information")
	}
	if !label.Label.Sid.IsValid() || label.Label.Sid.SubAuthorityCount() != 1 || label.Label.Sid.IdentifierAuthority() != windows.SECURITY_MANDATORY_LABEL_AUTHORITY {
		return 0, errors.New("Invalid integrity label")
	}
	return label.Label.Sid.SubAuthority(0), nil
}

func (native) activate(hwnd uintptr) bool {
	result, _, _ := setForegroundWindow.Call(hwnd)
	return result != 0
}
func (native) down(key uint16) bool {
	state, _, _ := getAsyncKeyState.Call(uintptr(key))
	return state&0x8000 != 0
}

// INPUT is 40 bytes on Windows amd64; KEYBDINPUT occupies 24 bytes of its 32-byte union.
type keyboardInput struct {
	Key, Scan   uint16
	Flags, Time uint32
	Extra       uintptr
}
type input struct {
	Kind      uint32
	Keyboard  keyboardInput
	UnionTail [8]byte
}

func (native) inject(keys []keyEvent) (int, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	data := make([]input, len(keys))
	for i, key := range keys {
		data[i].Kind, data[i].Keyboard.Key = 1, key.Key
		if key.Up {
			data[i].Keyboard.Flags = 2
		}
	}
	count, _, err := sendInput.Call(uintptr(len(data)), uintptr(unsafe.Pointer(&data[0])), unsafe.Sizeof(data[0]))
	if int(count) == len(data) {
		return int(count), nil
	}
	if err == nil || err == windows.ERROR_SUCCESS {
		err = errors.New("Windows blocked keyboard input")
	}
	return int(count), err
}

func (native) pause(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
