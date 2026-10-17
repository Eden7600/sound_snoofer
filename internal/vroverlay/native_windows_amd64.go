//go:build windows && amd64

package vroverlay

import (
	"errors"
	"fmt"
	"image"
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Layout pinned to Valve's IVROverlay_028 C function table, not its C++ vtable:
// https://github.com/ValveSoftware/openvr/blob/master/headers/openvr_capi.h
// The unused slots must stay present when the requested interface is unchanged.
type overlayTable struct {
	_         [3]uintptr
	destroy   uintptr // 3
	_         [18]uintptr
	width     uintptr // 22
	_         [22]uintptr
	visible   uintptr // 45
	_         [2]uintptr
	poll      uintptr // 48
	_         uintptr
	input     uintptr // 50
	_         uintptr
	scale     uintptr // 52
	_         [9]uintptr
	raw       uintptr // 62
	_         [4]uintptr
	create    uintptr // 67
	dashboard uintptr // 68
	active    uintptr // 69
}

func (t overlayTable) valid() bool {
	return t.destroy != 0 && t.width != 0 && t.visible != 0 && t.poll != 0 &&
		t.input != 0 && t.scale != 0 && t.raw != 0 && t.create != 0 && t.dashboard != 0 && t.active != 0
}

type nativeDriver struct {
	dll      *windows.DLL
	shutdown *windows.Proc
	table    overlayTable
}

// Open loads the installed OpenVR loader and creates a dashboard tab without
// summoning the dashboard. Call only in the isolated overlay worker, on a locked
// OS thread. Overlay initialization can start SteamVR if it exits after the
// supervisor's process check; no background/no-launch guarantee is implied.
func Open(width, height int, thumbnail *image.RGBA) (*Session, error) {
	if err := validateImages(width, height, thumbnail); err != nil {
		return nil, err
	}
	path, err := installedLoaderPath()
	if err != nil {
		return nil, err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("identify VR clipboard user: %w", err)
	}
	key := "snoofer.vrclipboard." + user.User.Sid.String()
	handle, err := windows.LoadLibraryEx(path, 0, windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
	if err != nil {
		return nil, fmt.Errorf("load installed OpenVR loader: %w", err)
	}
	dll := &windows.DLL{Name: path, Handle: handle}
	init, err := dll.FindProc("VR_InitInternal2")
	if err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	getInterface, err := dll.FindProc("VR_GetGenericInterface")
	if err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	shutdown, err := dll.FindProc("VR_ShutdownInternal")
	if err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	var initError int32
	init.Call(uintptr(unsafe.Pointer(&initError)), 2, 0) // VRApplication_Overlay.
	if initError != 0 {
		return nil, errors.Join(fmt.Errorf("initialize OpenVR: error %d", initError), dll.Release())
	}
	d := &nativeDriver{dll: dll, shutdown: shutdown}
	version, _ := windows.BytePtrFromString(interfaceVersion)
	pointer, _, _ := getInterface.Call(uintptr(unsafe.Pointer(version)), uintptr(unsafe.Pointer(&initError)))
	if initError != 0 || pointer == 0 {
		return nil, errors.Join(fmt.Errorf("OpenVR requires IVROverlay_028: error %d", initError), d.close())
	}
	// The return value is a foreign address, not a Go pointer converted earlier.
	// Reinterpret its pointer-sized representation and copy while the DLL lives.
	table := *(**overlayTable)(unsafe.Pointer(&pointer))
	d.table = *table
	if !d.table.valid() {
		return nil, errors.Join(errors.New("OpenVR returned an incomplete IVROverlay_028 table"), d.close())
	}
	return newSession(d, key, width, height, thumbnail)
}

func overlayResult(operation string, code uintptr) error {
	if uint32(code) == 0 {
		return nil
	}
	return fmt.Errorf("OpenVR %s: overlay error %d", operation, uint32(code))
}

func (d *nativeDriver) create(key string) (uint64, uint64, error) {
	name, err := windows.BytePtrFromString(key)
	if err != nil {
		return 0, 0, err
	}
	label, _ := windows.BytePtrFromString("Snoofer Clipboard")
	var main, thumbnail uint64
	code, _, _ := syscall.SyscallN(d.table.create, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(label)), uintptr(unsafe.Pointer(&main)), uintptr(unsafe.Pointer(&thumbnail)))
	return main, thumbnail, overlayResult("create dashboard", code)
}

func (d *nativeDriver) configure(main uint64, width, height int) error {
	code, _, _ := syscall.SyscallN(d.table.width, uintptr(main), uintptr(math.Float32bits(1.2)))
	if err := overlayResult("set panel width", code); err != nil {
		return err
	}
	code, _, _ = syscall.SyscallN(d.table.input, uintptr(main), 1) // VROverlayInputMethod_Mouse.
	if err := overlayResult("enable pointer input", code); err != nil {
		return err
	}
	scale := [2]float32{float32(width), float32(height)}
	code, _, _ = syscall.SyscallN(d.table.scale, uintptr(main), uintptr(unsafe.Pointer(&scale)))
	return overlayResult("set pointer coordinates", code)
}

func (d *nativeDriver) draw(handle uint64, pixels []byte, width, height int) error {
	code, _, _ := syscall.SyscallN(d.table.raw, uintptr(handle), uintptr(unsafe.Pointer(&pixels[0])), uintptr(width), uintptr(height), 4)
	runtime.KeepAlive(pixels)
	return overlayResult("submit image", code)
}

func (d *nativeDriver) poll(handle uint64, event *nativeEvent) bool {
	result, _, _ := syscall.SyscallN(d.table.poll, uintptr(handle), uintptr(unsafe.Pointer(event)), unsafe.Sizeof(*event))
	return byte(result) != 0 // OpenVR returns a C bool, not a Windows BOOL.
}

func (d *nativeDriver) active(handle uint64) bool {
	result, _, _ := syscall.SyscallN(d.table.active, uintptr(handle))
	return byte(result) != 0
}

func (d *nativeDriver) destroy(handle uint64) error {
	code, _, _ := syscall.SyscallN(d.table.destroy, uintptr(handle))
	return overlayResult("destroy dashboard", code)
}

func (d *nativeDriver) close() error {
	d.shutdown.Call()
	return d.dll.Release()
}
