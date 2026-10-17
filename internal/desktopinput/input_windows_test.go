//go:build windows && amd64

package desktopinput

import (
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
