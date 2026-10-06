//go:build windows

package windowsaudio

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// COM identifiers for the session interfaces. Vtable indices below follow
// audiopolicy.h and endpointvolume.h (IUnknown occupies 0–2).
const (
	iidSessionManager2 = "{77AA99A0-1BD6-484F-8BC7-2C654C9A9B6F}"
	iidSessionControl2 = "{BFB7FF88-7239-4FC9-8FA2-07C950BE9C6D}"
	iidSimpleVolume    = "{87CE5498-68D6-44E5-9215-6DA47EF883D8}"
	iidMeter           = "{C02216F6-8C67-4B5B-9D00-D008E73E0064}"
	clsctxAll          = 23
	sessionExpired     = 2
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	gdi32                 = windows.NewLazySystemDLL("gdi32.dll")
	privateExtractIcons   = user32.NewProc("PrivateExtractIconsW")
	getIconInfo           = user32.NewProc("GetIconInfo")
	destroyIcon           = user32.NewProc("DestroyIcon")
	getDC                 = user32.NewProc("GetDC")
	releaseDC             = user32.NewProc("ReleaseDC")
	getDIBits             = gdi32.NewProc("GetDIBits")
	deleteObject          = gdi32.NewProc("DeleteObject")
	errSessionUnavailable = fmt.Errorf("audio session ended")
)

// openSession holds the interfaces of one listed session until the next List.
type openSession struct {
	volume, meter *com
}

// program is the cached identity of an executable.
type program struct {
	modified          time.Time
	description, icon string
}

type nativeSessions struct {
	enumerator *com
	open       map[string]openSession
	programs   map[string]program
}

// OpenSessions initializes COM on the calling goroutine's OS thread, which the
// caller must lock for the backend's whole life, and opens the device
// enumerator. Close releases both.
func OpenSessions() (SessionBackend, error) {
	r, _, _ := ole.NewProc("CoInitializeEx").Call(0, 0)
	if int32(r) < 0 {
		return nil, fmt.Errorf("COM initialization failed (0x%08x)", uint32(r))
	}
	e, err := create("{BCDE0395-E52F-467C-8E3D-C4579291692E}", "{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	if err != nil {
		ole.NewProc("CoUninitialize").Call()
		return nil, err
	}
	return &nativeSessions{enumerator: e, open: map[string]openSession{}, programs: map[string]program{}}, nil
}

func (n *nativeSessions) Close() {
	n.releaseSessions()
	release(n.enumerator)
	n.enumerator = nil
	ole.NewProc("CoUninitialize").Call()
}

func (n *nativeSessions) releaseSessions() {
	for _, s := range n.open {
		release(s.volume)
		release(s.meter)
	}
	n.open = map[string]openSession{}
}

func query(o *com, iid string) (*com, error) {
	var out *com
	if _, err := call(o, 0, uintptr(unsafe.Pointer(guid(iid))), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

// taskString reads a CoTaskMem string out parameter at a vtable index.
func taskString(o *com, index int) (string, error) {
	var ptr *uint16
	if _, err := call(o, index, uintptr(unsafe.Pointer(&ptr))); err != nil {
		return "", err
	}
	if ptr == nil {
		return "", nil
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(ptr))
	return windows.UTF16PtrToString(ptr), nil
}

func friendlyName(d *com) string {
	var props *com
	if _, err := call(d, 4, 0, uintptr(unsafe.Pointer(&props))); err != nil {
		return ""
	}
	defer release(props)
	key := propertyKey{*guid("{A45C254E-DF1C-4EFD-8020-67D146A850E0}"), 14}
	var v variant
	if _, err := call(props, 5, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&v))); err != nil {
		return ""
	}
	defer ole.NewProc("PropVariantClear").Call(uintptr(unsafe.Pointer(&v)))
	if v.Type != 31 {
		return ""
	}
	return windows.UTF16PtrToString(v.Value)
}

func (n *nativeSessions) List() ([]Session, error) {
	n.releaseSessions()
	var collection *com
	if _, err := call(n.enumerator, 3, 0, 1, uintptr(unsafe.Pointer(&collection))); err != nil {
		return nil, err
	}
	defer release(collection)
	var count uint32
	if _, err := call(collection, 3, uintptr(unsafe.Pointer(&count))); err != nil {
		return nil, err
	}
	if count > 4096 {
		return nil, fmt.Errorf("invalid endpoint count")
	}
	out := []Session{}
	for i := uint32(0); i < count; i++ {
		var device *com
		if _, err := call(collection, 4, uintptr(i), uintptr(unsafe.Pointer(&device))); err != nil {
			continue
		}
		out = append(out, n.deviceSessions(device)...)
		release(device)
	}
	return out, nil
}

// deviceSessions lists one device's live sessions. A session that fails to
// answer is skipped; it is usually ending.
func (n *nativeSessions) deviceSessions(device *com) []Session {
	id, err := deviceID(device)
	if err != nil {
		return nil
	}
	name := friendlyName(device)
	var manager *com
	if _, err := call(device, 3, uintptr(unsafe.Pointer(guid(iidSessionManager2))), clsctxAll, 0, uintptr(unsafe.Pointer(&manager))); err != nil {
		return nil
	}
	defer release(manager)
	var sessions *com
	if _, err := call(manager, 5, uintptr(unsafe.Pointer(&sessions))); err != nil {
		return nil
	}
	defer release(sessions)
	var count int32
	if _, err := call(sessions, 3, uintptr(unsafe.Pointer(&count))); err != nil || count < 0 || count > 4096 {
		return nil
	}
	var out []Session
	for j := int32(0); j < count; j++ {
		var control *com
		if _, err := call(sessions, 4, uintptr(j), uintptr(unsafe.Pointer(&control))); err != nil {
			continue
		}
		s, ok := n.session(control, id, name)
		release(control)
		if ok {
			out = append(out, s)
		}
	}
	return out
}

func (n *nativeSessions) session(control *com, deviceID, deviceName string) (Session, bool) {
	c2, err := query(control, iidSessionControl2)
	if err != nil {
		return Session{}, false
	}
	defer release(c2)
	var state int32
	if _, err := call(c2, 3, uintptr(unsafe.Pointer(&state))); err != nil || state == sessionExpired {
		return Session{}, false
	}
	instance, err := taskString(c2, 13)
	if err != nil || instance == "" {
		return Session{}, false
	}
	var pid uint32
	// AUDCLNT_S_NO_SINGLE_PROCESS is a success code; the PID is then unreliable.
	if r, err := call(c2, 14, uintptr(unsafe.Pointer(&pid))); err != nil || r != 0 {
		pid = 0
	}
	system, _ := call(c2, 15)
	display, _ := taskString(c2, 4)
	volume, err := query(c2, iidSimpleVolume)
	if err != nil {
		return Session{}, false
	}
	meter, err := query(c2, iidMeter)
	if err != nil {
		release(volume)
		return Session{}, false
	}
	var level float32
	var muted int32
	_, levelErr := call(volume, 4, uintptr(unsafe.Pointer(&level)))
	_, muteErr := call(volume, 6, uintptr(unsafe.Pointer(&muted)))
	if levelErr != nil || muteErr != nil {
		release(volume)
		release(meter)
		return Session{}, false
	}
	path := SystemSounds
	if system != 0 { // S_OK marks the system sounds session; S_FALSE any other.
		path = processPath(pid)
	}
	p := n.program(path)
	key := deviceID + "|" + instance
	n.open[key] = openSession{volume: volume, meter: meter}
	return Session{Key: key, Device: deviceName, PID: pid, Path: path, Name: programName(path, p.description, display), Icon: p.icon,
		Active: state == 1, Volume: float64(level), Muted: muted != 0}, true
}

// processPath is the executable of a process, or empty when it cannot be read.
func processPath(pid uint32) string {
	if pid == 0 {
		return ""
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

// program returns an executable's description and icon, cached while the
// file's modification time is unchanged.
func (n *nativeSessions) program(path string) program {
	if path == "" || path == SystemSounds {
		return program{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return program{}
	}
	if p, ok := n.programs[path]; ok && p.modified.Equal(info.ModTime()) {
		return p
	}
	p := program{modified: info.ModTime(), description: fileDescription(path), icon: exeIcon(path)}
	n.programs[path] = p
	return p
}

// fileDescription reads FileDescription from the first translation of an
// executable's version resource.
func fileDescription(path string) string {
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil || size == 0 || size > 16*1024*1024 {
		return ""
	}
	data := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&data[0])); err != nil {
		return ""
	}
	var translation *[2]uint16
	var length uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&data[0]), `\VarFileInfo\Translation`, unsafe.Pointer(&translation), &length); err != nil || length < 4 {
		return ""
	}
	var text *uint16
	sub := fmt.Sprintf(`\StringFileInfo\%04x%04x\FileDescription`, translation[0], translation[1])
	if err := windows.VerQueryValue(unsafe.Pointer(&data[0]), sub, unsafe.Pointer(&text), &length); err != nil || length == 0 {
		return ""
	}
	return windows.UTF16PtrToString(text)
}

type iconInfo struct {
	Icon               int32
	HotspotX, HotspotY uint32
	Mask, Color        windows.Handle
}

type bitmapInfoHeader struct {
	Size                        uint32
	Width, Height               int32
	Planes, BitCount            uint16
	Compression, SizeImage      uint32
	XPerMeter, YPerMeter        int32
	ColorsUsed, ColorsImportant uint32
}

// iconSize is the thumbnail edge, matching the control artwork contract.
const iconSize = 64

// exeIcon extracts an executable's first icon at 64 px as a base64 PNG.
func exeIcon(path string) string {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var icon windows.Handle
	count, _, _ := privateExtractIcons.Call(uintptr(unsafe.Pointer(name)), 0, iconSize, iconSize, uintptr(unsafe.Pointer(&icon)), 0, 1, 0)
	if count == 0 || count == math.MaxUint32 || icon == 0 {
		return ""
	}
	defer destroyIcon.Call(uintptr(icon))
	var info iconInfo
	if ok, _, _ := getIconInfo.Call(uintptr(icon), uintptr(unsafe.Pointer(&info))); ok == 0 {
		return ""
	}
	defer deleteObject.Call(uintptr(info.Mask))
	if info.Color == 0 {
		return "" // Monochrome icons have no colour bitmap; the fallback symbol is clearer.
	}
	defer deleteObject.Call(uintptr(info.Color))
	color, ok := bitmapPixels(info.Color)
	if !ok {
		return ""
	}
	im := image.NewNRGBA(image.Rect(0, 0, iconSize, iconSize))
	hasAlpha := false
	for i := 0; i < len(color); i += 4 {
		hasAlpha = hasAlpha || color[i+3] != 0
	}
	var mask []byte
	if !hasAlpha {
		if mask, ok = bitmapPixels(info.Mask); !ok {
			return ""
		}
	}
	for i := 0; i < len(color); i += 4 {
		alpha := color[i+3]
		if !hasAlpha {
			alpha = 255
			if mask[i] != 0 { // A set mask bit is transparent.
				alpha = 0
			}
		}
		im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = color[i+2], color[i+1], color[i], alpha
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, im); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(encoded.Bytes())
}

// bitmapPixels reads a bitmap as top-down 32-bit BGRA at the icon size.
func bitmapPixels(bitmap windows.Handle) ([]byte, bool) {
	dc, _, _ := getDC.Call(0)
	if dc == 0 {
		return nil, false
	}
	defer releaseDC.Call(0, dc)
	header := bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: iconSize, Height: -iconSize, Planes: 1, BitCount: 32}
	pixels := make([]byte, iconSize*iconSize*4)
	lines, _, _ := getDIBits.Call(dc, uintptr(bitmap), 0, iconSize, uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), 0)
	return pixels, lines == iconSize
}

func (n *nativeSessions) Peak(key string) (float64, error) {
	s, ok := n.open[key]
	if !ok {
		return 0, errSessionUnavailable
	}
	var peak float32
	if _, err := call(s.meter, 3, uintptr(unsafe.Pointer(&peak))); err != nil {
		return 0, err
	}
	return float64(peak), nil
}

// SetVolume passes the float in the second argument slot; Go's Windows
// syscall path mirrors the first four slots into XMM0–3 as the x64 ABI needs.
func (n *nativeSessions) SetVolume(key string, volume float64) error {
	s, ok := n.open[key]
	if !ok {
		return errSessionUnavailable
	}
	_, err := call(s.volume, 3, uintptr(math.Float32bits(float32(min(1, max(0, volume))))), 0)
	return err
}

func (n *nativeSessions) SetMute(key string, muted bool) error {
	s, ok := n.open[key]
	if !ok {
		return errSessionUnavailable
	}
	value := uintptr(0)
	if muted {
		value = 1
	}
	_, err := call(s.volume, 5, value, 0)
	return err
}
