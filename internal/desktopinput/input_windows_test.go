//go:build windows && amd64

package desktopinput

import (
	"os"
	"testing"
	"unsafe"
)

// ABI-only check: no lazy procedure is resolved or called.
func TestWindowsInputABI(t *testing.T) {
	var event input
	if unsafe.Sizeof(event) != 40 || unsafe.Offsetof(event.Keyboard) != 8 || unsafe.Sizeof(event.Keyboard) != 24 || unsafe.Offsetof(event.Keyboard.Extra) != 16 {
		t.Fatal("Windows amd64 INPUT ABI mismatch")
	}
}

// This opt-in diagnostic reads identity only; it never calls Send or changes focus.
func TestNativeCaptureReadOnly(t *testing.T) {
	if os.Getenv("SNOOFER_VRCLIPBOARD_CAPTURE_CHECK") != "1" {
		t.Skip("native foreground observation requires explicit opt-in")
	}
	target, err := Capture()
	if err != nil {
		t.Fatalf("foreground capture failed: %v", err)
	}
	t.Logf("Captured foreground application: %s", target.Label)
}

func TestIntegrityQueryBufferAlignment(t *testing.T) {
	var buffer integrityBuffer
	if unsafe.Alignof(buffer) < 8 || unsafe.Sizeof(buffer) < 256 {
		t.Fatal("token query buffer must be pointer-aligned with room for its SID")
	}
}
