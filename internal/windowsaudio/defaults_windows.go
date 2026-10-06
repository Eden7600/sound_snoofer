//go:build windows

package windowsaudio

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ole = windows.NewLazySystemDLL("ole32.dll")

type com struct{ vt *[32]uintptr }

// call keeps Go arguments passed as native addresses alive through the COM call.
//
//go:uintptrescapes
func call(o *com, index int, args ...uintptr) (uintptr, error) {
	if o == nil {
		return 0, fmt.Errorf("Windows audio interface unavailable")
	}
	values := append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)
	r, _, _ := syscall.SyscallN(o.vt[index], values...)
	if int32(r) < 0 {
		return r, fmt.Errorf("Windows audio HRESULT 0x%08x", uint32(r))
	}
	return r, nil
}
func release(o *com) {
	if o != nil {
		call(o, 2)
	}
}
func guid(s string) *windows.GUID { g, _ := windows.GUIDFromString(s); return &g }
func create(class, iid string) (*com, error) {
	var out *com
	r, _, _ := ole.NewProc("CoCreateInstance").Call(uintptr(unsafe.Pointer(guid(class))), 0, 1, uintptr(unsafe.Pointer(guid(iid))), uintptr(unsafe.Pointer(&out)))
	if int32(r) < 0 {
		return nil, fmt.Errorf("Windows audio adapter unavailable (0x%08x)", uint32(r))
	}
	return out, nil
}

type native struct{ enumerator, policy *com }

func open() (*native, error) {
	e, err := create("{BCDE0395-E52F-467C-8E3D-C4579291692E}", "{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	if err != nil {
		return nil, err
	}
	return &native{enumerator: e}, nil
}
func (n *native) Close() { release(n.policy); release(n.enumerator) }
func deviceID(d *com) (string, error) {
	var ptr *uint16
	_, err := call(d, 5, uintptr(unsafe.Pointer(&ptr)))
	if err != nil {
		return "", err
	}
	defer ole.NewProc("CoTaskMemFree").Call(uintptr(unsafe.Pointer(ptr)))
	return windows.UTF16PtrToString(ptr), nil
}
func (n *native) Default(flow, role int) (string, error) {
	var d *com
	_, err := call(n.enumerator, 4, uintptr(flow), uintptr(role), uintptr(unsafe.Pointer(&d)))
	if err != nil {
		return "", err
	}
	defer release(d)
	return deviceID(d)
}
func (n *native) Set(id string, role int) error {
	if n.policy == nil {
		p, e := create("{870AF99C-171D-4F9E-AF0D-E63DF40C2BC9}", "{F8679F50-850A-41CF-9C72-430F290290C8}")
		if e != nil {
			return e
		}
		n.policy = p
	}
	ptr, e := windows.UTF16PtrFromString(id)
	if e != nil {
		return e
	}
	_, e = call(n.policy, 13, uintptr(unsafe.Pointer(ptr)), uintptr(role))
	return e
}

type propertyKey struct {
	ID  windows.GUID
	PID uint32
}
type variant struct {
	Type     uint16
	Reserved [3]uint16
	Value    *uint16
	Extra    uintptr
}

func (n *native) Endpoints(flow int) ([]Endpoint, error) {
	var collection *com
	_, err := call(n.enumerator, 3, uintptr(flow), 1, uintptr(unsafe.Pointer(&collection)))
	if err != nil {
		return nil, err
	}
	defer release(collection)
	var count uint32
	if _, err = call(collection, 3, uintptr(unsafe.Pointer(&count))); err != nil {
		return nil, err
	}
	if count > 4096 {
		return nil, fmt.Errorf("invalid endpoint count")
	}
	out := []Endpoint{}
	for i := uint32(0); i < count; i++ {
		var d *com
		if _, e := call(collection, 4, uintptr(i), uintptr(unsafe.Pointer(&d))); e != nil {
			continue
		}
		id, e := deviceID(d)
		name := ""
		var props *com
		if _, err := call(d, 4, 0, uintptr(unsafe.Pointer(&props))); err == nil {
			key := propertyKey{*guid("{A45C254E-DF1C-4EFD-8020-67D146A850E0}"), 14}
			var v variant
			if _, e := call(props, 5, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&v))); e == nil {
				if v.Type == 31 {
					name = windows.UTF16PtrToString(v.Value)
				}
				ole.NewProc("PropVariantClear").Call(uintptr(unsafe.Pointer(&v)))
			}
			release(props)
		}
		release(d)
		if e == nil {
			out = append(out, Endpoint{id, name})
		}
	}
	return out, nil
}
func resolve(endpoints []Endpoint, want string, flow int) string {
	found := ""
	count := 0
	for _, e := range endpoints {
		match := e.ID == want || e.Name == want
		if want == "" {
			name := strings.ToLower(e.Name)
			if flow == 0 {
				match = strings.HasPrefix(name, "voicemeeter input (")
			} else {
				match = strings.HasPrefix(name, "voicemeeter vaio3 output (") || strings.HasPrefix(name, "voicemeeter out b3 (")
			}
		}
		if match {
			found = e.ID
			count++
		}
	}
	if count != 1 {
		return ""
	}
	return found
}
func endpointName(endpoints []Endpoint, id string) string {
	for _, e := range endpoints {
		if id != "" && e.ID == id {
			return e.Name
		}
	}
	return ""
}
func run(ctx context.Context, requests <-chan Request, results chan Result) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(results)
	r, _, _ := ole.NewProc("CoInitializeEx").Call(0, 0)
	if int32(r) < 0 {
		results <- Result{Status: "Windows COM initialization failed", Kind: Attention}
		return
	}
	defer ole.NewProc("CoUninitialize").Call()
	b, err := open()
	if err != nil {
		results <- Result{Status: err.Error(), Kind: Attention}
		return
	}
	defer b.Close()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	request := Request{}
	guard := Guard{}
	var targets, names [2]string
	var scan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case next := <-requests:
			if next != request {
				scan = time.Time{}
			}
			if next.Ack != nil {
				close(next.Ack)
				next.Ack = nil
			}
			request = next
		case <-ticker.C:
		}
		if ctx.Err() != nil {
			return
		}
		if request.Enabled && time.Since(scan) > 5*time.Second {
			scan = time.Now()
			for flow := 0; flow < 2; flow++ {
				endpoints, e := b.Endpoints(flow)
				if e != nil {
					targets[flow], names[flow] = "", ""
					continue
				}
				want := request.Playback
				if flow == 1 {
					want = request.Capture
				}
				targets[flow] = resolve(endpoints, want, flow)
				names[flow] = endpointName(endpoints, targets[flow])
			}
		}
		status := guard.Reconcile(b, request, targets, time.Now())
		status.Playback, status.Capture = names[0], names[1]
		status.Suspended = guard.Suspended[0] || guard.Suspended[1]
		for _, times := range guard.Last {
			for _, at := range times {
				if at.After(status.LastCorrection) {
					status.LastCorrection = at
				}
			}
		}
		select {
		case results <- status:
		default:
			select {
			case <-results:
			default:
			}
			select {
			case results <- status:
			default:
			}
		}
	}
}
