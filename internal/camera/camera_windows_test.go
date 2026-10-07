//go:build windows

package camera

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
)

// Opt-in: SNOOFER_CAMERA_DLL names a built snoofer-camera.dll. With a Link 2
// connected, reads its status packet and privacy control without writing.
func TestNativeCameraRead(t *testing.T) {
	path := os.Getenv("SNOOFER_CAMERA_DLL")
	if path == "" {
		t.Skip("set SNOOFER_CAMERA_DLL to the built companion")
	}
	if unsafe.Sizeof(GUID{}) != 16 {
		t.Fatal("GUID layout", unsafe.Sizeof(GUID{}))
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	lib, err := Load(abs)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lib.Release(); err != nil {
			t.Error(err)
		}
	}()
	if _, err := lib.Open("vid_ffff&pid_ffff"); err != ErrAbsent {
		t.Fatal("absent device", err)
	}
	d, err := lib.Open("vid_2e1a&pid_4c04")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	xu1 := GUID{0xFAF1672D, 0xB71B, 0x4793, [8]byte{0x8C, 0x91, 0x7B, 0x1C, 0x9B, 0x7F, 0x95, 0xF8}}
	xu2 := GUID{0xE307E649, 0x4618, 0xA3FF, [8]byte{0x82, 0xFC, 0x2D, 0x8B, 0x5F, 0x21, 0x67, 0x73}}
	status, err := d.Get(xu1, 0x02)
	if err != nil {
		t.Fatal("status", err)
	}
	t.Logf("status (%d bytes): % x", len(status), status)
	privacy, err := d.Get(xu2, 0x0F)
	if err != nil {
		t.Fatal("privacy", err)
	}
	t.Logf("privacy: % x", privacy)
	framing, err := d.Get(xu1, 0x13)
	t.Logf("framing: % x (%v)", framing, err)
	missing := GUID{Data1: 0x12345678}
	if _, err := d.Get(missing, 1); err != ErrNoControl {
		t.Fatal("unknown property set", err)
	}
}
