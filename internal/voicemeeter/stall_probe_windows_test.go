//go:build windows && amd64

package voicemeeter

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TestStalledEngineProbe is opt-in and never invokes a native setter.
func TestStalledEngineProbe(t *testing.T) {
	logPath := os.Getenv("SNOOFER_STALL_PROBE")
	if logPath == "" {
		t.Skip("explicit read-only incident capture only")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	log := func(format string, args ...any) {
		fmt.Fprintf(f, "%s ", time.Now().UTC().Format(time.RFC3339Nano))
		fmt.Fprintf(f, format+"\n", args...)
		f.Sync()
	}
	log("BEGIN discover")
	path, err := discover()
	if err != nil {
		t.Fatal(err)
	}
	log("END discover %s; BEGIN LoadDLL", path)
	dll, err := windows.LoadDLL(path)
	if err != nil {
		t.Fatal(err)
	}
	a := &winAPI{dll: dll, procs: map[string]*windows.Proc{}}
	defer dll.Release()
	for _, name := range exports {
		p, e := dll.FindProc("VBVMR_" + name)
		if e != nil {
			t.Fatal(e)
		}
		a.procs[name] = p
	}
	level, err := dll.FindProc("VBVMR_GetLevel")
	if err != nil {
		t.Fatal(err)
	}
	log("BEGIN Login")
	code := a.Login()
	log("END Login code=%d", code)
	if code != 0 && code != 1 {
		t.Fatal(code)
	}
	defer func() { log("BEGIN Logout"); log("END Logout code=%d", a.Logout()) }()
	for n := 0; n < 12; n++ {
		log("BEGIN Refresh sample=%d", n)
		code = a.Refresh()
		log("END Refresh code=%d", code)
		log("BEGIN Edition")
		edition, status := a.Edition()
		log("END Edition value=%d code=%d", edition, status)
		for _, p := range []string{"Bus[0].device.sr", "Bus[1].device.sr", "Strip[0].device.sr", "Option.sr", "Recorder.record", "Recorder.stop", "Recorder.pause", "Recorder.play"} {
			log("BEGIN GetNumber %s", p)
			value, status := a.GetNumber(p)
			log("END GetNumber %s value=%g code=%d", p, value, status)
		}
		for _, p := range []string{"Bus[0].device.name", "Bus[1].device.name"} {
			log("BEGIN Get %s", p)
			value, status := a.Get(p)
			log("END Get %s value=%q code=%d", p, value, status)
		}
		// The SDK defines 34 input and 64 output channels for Potato.
		if edition == 3 {
			for kind := 0; kind < 4; kind++ {
				count := 34
				if kind == 3 {
					count = 64
				}
				for channel := 0; channel < count; channel++ {
					var value float32
					log("BEGIN GetLevel type=%d channel=%d", kind, channel)
					code, _, _ := level.Call(uintptr(kind), uintptr(channel), uintptr(unsafe.Pointer(&value)))
					log("END GetLevel type=%d channel=%d value=%g code=%d", kind, channel, value, result(code))
				}
			}
		}
		if n == 0 {
			for _, direction := range []string{"input", "output"} {
				log("BEGIN Count %s", direction)
				count := a.Count(direction)
				log("END Count %s value=%d", direction, count)
				if count < 0 || count > 4096 {
					t.Fatal("invalid device count", count)
				}
				for i := 0; i < int(count); i++ {
					log("BEGIN Device %s %d", direction, i)
					d, status := a.Device(direction, i)
					log("END Device %s %d name=%q driver=%s code=%d", direction, i, d.Name, d.Driver, status)
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}
