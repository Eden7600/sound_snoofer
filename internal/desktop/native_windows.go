//go:build windows

package desktop

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var kernel = windows.NewLazySystemDLL("kernel32.dll")
var user = windows.NewLazySystemDLL("user32.dll")

// Console supplies terminal handles for a GUI-subsystem executable. Existing
// redirected standard streams are retained for explicit CLI commands.
func Console() error {
	kernel.NewProc("AttachConsole").Call(^uintptr(0))
	window, _, _ := kernel.NewProc("GetConsoleWindow").Call()
	if window == 0 {
		ok, _, err := kernel.NewProc("AllocConsole").Call()
		if ok == 0 {
			return fmt.Errorf("allocate controls console: %w", err)
		}
	}
	var err error
	if _, e := os.Stdin.Stat(); e != nil {
		os.Stdin, err = os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err != nil {
			return err
		}
	}
	if _, e := os.Stdout.Stat(); e != nil {
		os.Stdout, err = os.OpenFile("CONOUT$", os.O_RDWR, 0)
		if err != nil {
			return err
		}
	}
	if _, e := os.Stderr.Stat(); e != nil {
		os.Stderr = os.Stdout
	}
	return nil
}

func controlsConsole() (*os.File, *os.File, error) {
	ok, _, err := kernel.NewProc("AllocConsole").Call()
	if ok == 0 {
		return nil, nil, fmt.Errorf("allocate controls console: %w", err)
	}
	in, e := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if e != nil {
		return nil, nil, e
	}
	out, e := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if e != nil {
		in.Close()
		return nil, nil, e
	}
	return in, out, nil
}

func focusConsole() {
	window, _, _ := kernel.NewProc("GetConsoleWindow").Call()
	if window != 0 {
		user.NewProc("ShowWindow").Call(window, 9) // SW_RESTORE
		user.NewProc("SetForegroundWindow").Call(window)
	}
}

// ShowError makes startup errors visible even without a console.
func ShowError(err error) {
	message, _ := windows.UTF16PtrFromString(err.Error())
	title, _ := windows.UTF16PtrFromString("Sound Snoofer")
	user.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(title)), 0x10)
}

// The event's handle lifetime is the instance lease. Local scopes it to the
// logon session; the SID prevents different users sharing a profile from colliding.
func instance(path string, live bool) (windows.Handle, bool, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return 0, false, err
	}
	token, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return 0, false, err
	}
	key := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(absolute)) + fmt.Sprint(live)))
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf(`Local\SoundSnoofer.Tray.%s.%x`, token.User.Sid.String(), key[:12]))
	handle, err := windows.CreateEvent(nil, 0, 0, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if e := windows.SetEvent(handle); e != nil {
			windows.CloseHandle(handle)
			return 0, false, e
		}
		return handle, true, nil
	}
	return handle, false, err
}

// verifyTrayIcon compensates for systray v1.2.2 logging rather than returning
// Shell errors. Its Windows adapter uses SystrayClass and icon ID 100. Query only
// our owning thread, never another application's notification icons.
func verifyTrayIcon(thread uint32) error {
	var window uintptr
	callback := windows.NewCallback(func(hwnd, lparam uintptr) uintptr {
		var class [64]uint16
		user.NewProc("GetClassNameW").Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
		if windows.UTF16ToString(class[:]) == "SystrayClass" {
			window = hwnd
			return 0
		}
		return 1
	})
	user.NewProc("EnumThreadWindows").Call(uintptr(thread), callback, 0)
	if window == 0 {
		return fmt.Errorf("tray notification window was not created")
	}
	// NOTIFYICONIDENTIFIER uses native pointer alignment on Windows amd64.
	identifier := struct {
		Size   uint32
		Window uintptr
		ID     uint32
		GUID   windows.GUID
	}{Window: window, ID: 100}
	identifier.Size = uint32(unsafe.Sizeof(identifier))
	var rectangle struct{ Left, Top, Right, Bottom int32 }
	result, _, _ := windows.NewLazySystemDLL("shell32.dll").NewProc("Shell_NotifyIconGetRect").Call(uintptr(unsafe.Pointer(&identifier)), uintptr(unsafe.Pointer(&rectangle)))
	if int32(result) < 0 {
		return fmt.Errorf("Windows did not register the tray icon (HRESULT 0x%08x)", uint32(result))
	}
	return nil
}

// consoleFontInfo mirrors CONSOLE_FONT_INFOEX (84 bytes); COORD is two int16s.
type consoleFontInfo struct {
	Size, Index    uint32
	Width, Height  int16
	Family, Weight uint32
	Face           [32]uint16
}

// configureControlsFont changes only this freshly allocated console. Unsupported
// hosts retain their original font; font selection must never block audio controls.
func configureControlsFont(out *os.File) {
	original := consoleFontInfo{}
	original.Size = uint32(unsafe.Sizeof(original))
	get := kernel.NewProc("GetCurrentConsoleFontEx")
	set := kernel.NewProc("SetCurrentConsoleFontEx")
	ok, _, _ := get.Call(out.Fd(), 0, uintptr(unsafe.Pointer(&original)))
	if ok == 0 {
		return
	}
	for _, name := range []string{"Cascadia Mono", "Consolas"} {
		font := consoleFontInfo{Size: original.Size, Height: 18, Family: 0x36, Weight: 400}
		face, _ := windows.UTF16FromString(name)
		copy(font.Face[:], face)
		ok, _, _ := set.Call(out.Fd(), 0, uintptr(unsafe.Pointer(&font)))
		if ok == 0 {
			continue
		}
		actual := consoleFontInfo{Size: original.Size}
		ok, _, _ = get.Call(out.Fd(), 0, uintptr(unsafe.Pointer(&actual)))
		if ok != 0 && strings.EqualFold(windows.UTF16ToString(actual.Face[:]), name) {
			return
		}
	}
	// Best effort restoration when the console substitutes an unsupported face.
	set.Call(out.Fd(), 0, uintptr(unsafe.Pointer(&original)))
}
