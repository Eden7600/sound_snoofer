//go:build windows

package windowsaudio

import (
	"os"
	"runtime"
	"testing"
)

// Opt-in read-only acceptance probe; never sets a Windows default.
func TestNativeEndpointObservation(t *testing.T) {
	if os.Getenv("SNOOFER_READ_ONLY_PROBE") != "1" {
		t.Skip("manual read-only acceptance")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	r, _, _ := ole.NewProc("CoInitializeEx").Call(0, 0)
	if int32(r) < 0 {
		t.Fatal("COM initialization")
	}
	defer ole.NewProc("CoUninitialize").Call()
	b, e := open()
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	for flow := 0; flow < 2; flow++ {
		endpoints, e := b.Endpoints(flow)
		if e != nil {
			t.Fatal(e)
		}
		for _, endpoint := range endpoints {
			t.Logf("direction=%d name=%s", flow, endpoint.Name)
		}
		for role := 0; role < 3; role++ {
			id, e := b.Default(flow, role)
			if e != nil || id == "" {
				t.Fatal(flow, role, e)
			}
		}
		t.Logf("automatic target direction=%d resolved=%t", flow, resolve(endpoints, "", flow) != "")
	}
}
