//go:build windows && amd64

package voicemeeter

import (
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"sound-snoofer/internal/model"
)

type winAPI struct {
	dll   *windows.DLL
	procs map[string]*windows.Proc
}

var exports = []string{"Login", "Logout", "IsParametersDirty", "GetVoicemeeterType", "Input_GetDeviceNumber", "Output_GetDeviceNumber", "Input_GetDeviceDescW", "Output_GetDeviceDescW", "GetParameterStringW", "SetParameterStringW", "GetParameterFloat", "SetParameters"}

func Open(path string) (*Client, error) {
	if path == "" {
		var err error
		path, err = discover()
		if err != nil {
			return nil, err
		}
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("DLL path must be absolute: %q", path)
	}
	dll, e := windows.LoadDLL(path)
	if e != nil {
		return nil, fmt.Errorf("load Voicemeeter DLL %q: %w (use --dll with an installed 64-bit Remote DLL)", path, e)
	}
	a := &winAPI{dll: dll, procs: map[string]*windows.Proc{}}
	for _, name := range exports {
		p, e := dll.FindProc("VBVMR_" + name)
		if e != nil {
			dll.Release()
			return nil, fmt.Errorf("DLL %q missing %s: %w", path, name, e)
		}
		a.procs[name] = p
	}
	client, err := connect(a)
	if err == nil {
		client.elementProbe = elementProcess
		client.vrProbe = steamVRProcess
	}
	return client, err
}
func discover() (string, error) {
	const key = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\VB:Voicemeeter {17359A74-1236-5467}`
	for _, view := range []uint32{registry.WOW64_32KEY, registry.WOW64_64KEY} {
		k, e := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.QUERY_VALUE|view)
		if e != nil {
			continue
		}
		uninstall, _, e := k.GetStringValue("UninstallString")
		k.Close()
		if e != nil {
			continue
		}
		// Quoted paths may include switches. Official installer also uses an
		// unquoted path containing spaces, so do not split on whitespace.
		uninstall = strings.TrimSpace(uninstall)
		if strings.HasPrefix(uninstall, `"`) {
			if end := strings.Index(uninstall[1:], `"`); end >= 0 {
				uninstall = uninstall[1 : end+1]
			}
		}
		if filepath.IsAbs(uninstall) && strings.EqualFold(filepath.Ext(uninstall), ".exe") {
			return filepath.Join(filepath.Dir(uninstall), "VoicemeeterRemote64.dll"), nil
		}
	}
	return "", fmt.Errorf("Voicemeeter installation not found in registry; supply --dll with the absolute path to VoicemeeterRemote64.dll")
}

// API status is a signed Windows LONG, not GetLastError. These exports do not
// define GetLastError; windows.Proc.Call's third return value is irrelevant.
func result(r uintptr) int32     { return int32(uint32(r)) }
func (a *winAPI) Login() int32   { r, _, _ := a.procs["Login"].Call(); return result(r) }
func (a *winAPI) Logout() int32  { r, _, _ := a.procs["Logout"].Call(); return result(r) }
func (a *winAPI) Refresh() int32 { r, _, _ := a.procs["IsParametersDirty"].Call(); return result(r) }
func (a *winAPI) Edition() (int32, int32) {
	var v int32
	r, _, _ := a.procs["GetVoicemeeterType"].Call(uintptr(unsafe.Pointer(&v)))
	return v, result(r)
}
func prefix(direction string) string {
	if direction == "input" {
		return "Input"
	}
	return "Output"
}
func (a *winAPI) Count(direction string) int32 {
	r, _, _ := a.procs[prefix(direction)+"_GetDeviceNumber"].Call()
	return result(r)
}
func (a *winAPI) Device(direction string, i int) (model.Device, int32) {
	var kind int32
	var name, id [256]uint16
	r, _, _ := a.procs[prefix(direction)+"_GetDeviceDescW"].Call(uintptr(i), uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&id[0])))
	driver := map[int32]string{1: "mme", 3: "wdm", 4: "ks", 5: "asio"}[kind]
	if driver == "" {
		driver = fmt.Sprintf("unknown:%d", kind)
	}
	return model.Device{Name: windows.UTF16ToString(name[:]), ID: windows.UTF16ToString(id[:]), Driver: driver, Direction: direction}, result(r)
}
func (a *winAPI) Get(param string) (string, int32) {
	p, e := syscall.BytePtrFromString(param)
	if e != nil {
		return "", -3
	}
	var value [512]uint16
	r, _, _ := a.procs["GetParameterStringW"].Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&value[0])))
	return windows.UTF16ToString(value[:]), result(r)
}
func (a *winAPI) Set(param, value string) int32 {
	p, e := syscall.BytePtrFromString(param)
	if e != nil {
		return -3
	}
	v, e := windows.UTF16PtrFromString(value)
	if e != nil {
		return -3
	}
	r, _, _ := a.procs["SetParameterStringW"].Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(v)))
	return result(r)
}
func (a *winAPI) Release() error { return a.dll.Release() }

func (a *winAPI) GetNumber(param string) (float32, int32) {
	p, e := syscall.BytePtrFromString(param)
	if e != nil {
		return 0, -3
	}
	var v float32
	r, _, _ := a.procs["GetParameterFloat"].Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&v)))
	return v, result(r)
}
func (a *winAPI) SetNumber(param string, value int) int32 {
	// Client validates the parameter and integer domain. Device names never
	// enter this script, avoiding float-register ABI differences in syscall.
	p, e := syscall.BytePtrFromString(param + "=" + strconv.Itoa(value) + ";")
	if e != nil {
		return -3
	}
	r, _, _ := a.procs["SetParameters"].Call(uintptr(unsafe.Pointer(p)))
	return result(r)
}

func (a *winAPI) SetScalar(param string, value float32) int32 {
	p, e := syscall.BytePtrFromString(param + "=" + strconv.FormatFloat(float64(value), 'f', -1, 32) + ";")
	if e != nil {
		return -3
	}
	r, _, _ := a.procs["SetParameters"].Call(uintptr(unsafe.Pointer(p)))
	return result(r)
}
