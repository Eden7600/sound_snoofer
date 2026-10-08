//go:build windows

package presence

import (
	"context"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	wtsapi  = windows.NewLazySystemDLL("wtsapi32.dll")
	kernel  = windows.NewLazySystemDLL("kernel32.dll")
	getMod  = kernel.NewProc("GetModuleHandleW")
	regCls  = user32.NewProc("RegisterClassExW")
	create  = user32.NewProc("CreateWindowExW")
	destroy = user32.NewProc("DestroyWindow")
	defProc = user32.NewProc("DefWindowProcW")
	getMsg  = user32.NewProc("GetMessageW")
	dispMsg = user32.NewProc("DispatchMessageW")
	postMsg = user32.NewProc("PostMessageW")
	quitMsg = user32.NewProc("PostQuitMessage")
	regPow  = user32.NewProc("RegisterPowerSettingNotification")
	unPow   = user32.NewProc("UnregisterPowerSettingNotification")
	regWTS  = wtsapi.NewProc("WTSRegisterSessionNotification")
	unWTS   = wtsapi.NewProc("WTSUnRegisterSessionNotification")
	queryWT = wtsapi.NewProc("WTSQuerySessionInformationW")
	freeWTS = wtsapi.NewProc("WTSFreeMemory")
)

const (
	wmClose   = 0x0010
	wmDestroy = 0x0002
	// WTSSessionInfoEx; WTSINFOEXW's union is 8-byte aligned, so the
	// level-1 SessionFlags field sits at offset 16.
	wtsSessionInfoEx    = 25
	sessionFlagsOffset  = 16
	wtsCurrentSession   = ^uint32(0)
	notifyForThisSesion = 0
)

// GUID_CONSOLE_DISPLAY_STATE.
var consoleDisplayState = windows.GUID{Data1: 0x6fe69556, Data2: 0x704a, Data3: 0x47a0, Data4: [8]byte{0x8f, 0x24, 0xc2, 0x8d, 0x93, 0x6f, 0xda, 0x47}}

// One window procedure serves every watcher (callbacks are a finite
// resource); it routes messages by window handle.
var (
	procOnce sync.Once
	proc     uintptr
	watchers sync.Map // windows.Handle -> *watcher
	classErr error
	class    *uint16
)

type watcher struct {
	state  State
	states chan State
	notify uintptr // Power setting registration.
}

func (w *watcher) publish() {
	select {
	case <-w.states:
	default:
	}
	w.states <- w.state
}

func windowProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	value, ok := watchers.Load(windows.Handle(hwnd))
	if !ok {
		r, _, _ := defProc.Call(hwnd, msg, wparam, lparam)
		return r
	}
	w := value.(*watcher)
	switch msg {
	case wmClose:
		unWTS.Call(hwnd)
		if w.notify != 0 {
			unPow.Call(w.notify)
		}
		destroy.Call(hwnd)
		return 0
	case wmDestroy:
		quitMsg.Call(0)
		return 0
	}
	var display uint32
	if msg == wmPowerBroadcast && wparam == pbtPowerSettingChange && lparam != 0 {
		setting := (*windows.GUID)(native(lparam))
		if *setting != consoleDisplayState {
			return 1
		}
		display = *(*uint32)(unsafe.Add(native(lparam), powerSettingDataOffset))
	}
	if next, changed := apply(w.state, msg, wparam, display); changed {
		w.state = next
		w.publish()
	}
	if msg == wmPowerBroadcast {
		return 1
	}
	r, _, _ := defProc.Call(hwnd, msg, wparam, lparam)
	return r
}

// Watch observes the session lock and console display state until ctx is
// cancelled. The latest state is always available on the channel (older
// ones are dropped); done closes after the window is gone. Any failure
// reports an unknown state, which keeps the console usable.
func Watch(ctx context.Context) (<-chan State, <-chan struct{}) {
	states, done := make(chan State, 1), make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		w := &watcher{states: states}
		hwnd, err := open(w)
		if err != nil {
			w.publish() // Unknown.
			return
		}
		defer watchers.Delete(hwnd)
		w.state = State{Known: true, Locked: locked()}
		w.publish()
		// The power registration delivers the current display state at once.
		stop := context.AfterFunc(ctx, func() { postMsg.Call(uintptr(hwnd), wmClose, 0, 0) })
		defer stop()
		var msg [48]byte // MSG
		for {
			r, _, _ := getMsg.Call(uintptr(unsafe.Pointer(&msg[0])), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
			dispMsg.Call(uintptr(unsafe.Pointer(&msg[0])))
		}
	}()
	return states, done
}

// open creates the hidden window and registers both notifications.
func open(w *watcher) (windows.Handle, error) {
	procOnce.Do(func() {
		proc = windows.NewCallback(windowProc)
		class, _ = windows.UTF16PtrFromString("SnooferPresence")
		instance, _, _ := getMod.Call(0)
		// WNDCLASSEXW
		wc := struct {
			size, style            uint32
			proc                   uintptr
			clsExtra, wndExtra     int32
			instance, icon, cursor uintptr
			background             uintptr
			menuName, className    *uint16
			iconSm                 uintptr
		}{proc: proc, instance: instance, className: class}
		wc.size = uint32(unsafe.Sizeof(wc))
		if r, _, e := regCls.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
			classErr = e
		}
	})
	if classErr != nil {
		return 0, classErr
	}
	instance, _, _ := getMod.Call(0)
	// A hidden top-level window: it receives session and power notifications.
	r, _, e := create.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if r == 0 {
		return 0, e
	}
	hwnd := windows.Handle(r)
	watchers.Store(hwnd, w)
	if r, _, e := regWTS.Call(uintptr(hwnd), notifyForThisSesion); r == 0 {
		watchers.Delete(hwnd)
		destroy.Call(uintptr(hwnd))
		return 0, e
	}
	notify, _, e := regPow.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&consoleDisplayState)), 0)
	if notify == 0 {
		unWTS.Call(uintptr(hwnd))
		watchers.Delete(hwnd)
		destroy.Call(uintptr(hwnd))
		return 0, e
	}
	w.notify = notify
	return hwnd, nil
}

// locked reads the current session's lock state; unknown reads as unlocked.
func locked() bool {
	var buffer *byte
	var size uint32
	r, _, _ := queryWT.Call(0, uintptr(wtsCurrentSession), wtsSessionInfoEx, uintptr(unsafe.Pointer(&buffer)), uintptr(unsafe.Pointer(&size)))
	if r == 0 || buffer == nil {
		return false
	}
	defer freeWTS.Call(uintptr(unsafe.Pointer(buffer)))
	info := unsafe.Slice(buffer, size)
	// Level 1 (DWORD at 0), then SessionFlags.
	if size < sessionFlagsOffset+4 || *(*uint32)(unsafe.Pointer(&info[0])) != 1 {
		return false
	}
	return *(*int32)(unsafe.Pointer(&info[sessionFlagsOffset])) == sessionFlagLock
}

// native turns an address Windows passed as a message parameter into a
// pointer. The memory belongs to Windows and is valid for the message.
func native(address uintptr) unsafe.Pointer { return unsafe.Add(nil, address) }
