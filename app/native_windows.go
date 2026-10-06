//go:build windows

package app

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

// detachConsole releases only this process; it never hides a shared terminal.
func detachConsole() error {
	ok, _, err := kernel.NewProc("FreeConsole").Call()
	if ok == 0 {
		return fmt.Errorf("detach console: %w", err)
	}
	return nil
}

// ShowError makes startup errors visible even without a console.
func ShowError(err error) {
	message, _ := windows.UTF16PtrFromString(err.Error())
	title, _ := windows.UTF16PtrFromString("Snoofer")
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

// brandDesktop uses this executable's existing mascot resource on its own GUI window.
func brandDesktop() {
	callback := windows.NewCallback(func(window, _ uintptr) uintptr {
		var processID uint32
		user.NewProc("GetWindowThreadProcessId").Call(window, uintptr(unsafe.Pointer(&processID)))
		if processID == uint32(os.Getpid()) {
			setWindowIcon(window)
		}
		return 1
	})
	user.NewProc("EnumWindows").Call(callback, 0)
}

// Resource group 1 is the repository's existing ICO. Shared handles live until exit.
func setWindowIcon(window uintptr) {
	if window == 0 {
		return
	}
	module, _, _ := kernel.NewProc("GetModuleHandleW").Call(0)
	if module == 0 {
		return
	}
	for _, kind := range []struct{ icon, widthMetric, heightMetric uintptr }{
		{0, 49, 50}, // ICON_SMALL; SM_CXSMICON / SM_CYSMICON
		{1, 11, 12}, // ICON_BIG; SM_CXICON / SM_CYICON
	} {
		width, _, _ := user.NewProc("GetSystemMetrics").Call(kind.widthMetric)
		height, _, _ := user.NewProc("GetSystemMetrics").Call(kind.heightMetric)
		icon, _, _ := user.NewProc("LoadImageW").Call(module, 1, 1, width, height, 0x8000)
		if icon == 0 {
			// Presentation only: a development build without resources remains usable.
			continue
		}
		var previous uintptr
		// WM_SETICON; SMTO_ABORTIFHUNG. Hosts that reject the message keep their
		// own branding; never stall controls waiting for another process's window.
		user.NewProc("SendMessageTimeoutW").Call(window, 0x0080, kind.icon, icon, 2, 200, uintptr(unsafe.Pointer(&previous)))
	}
}
