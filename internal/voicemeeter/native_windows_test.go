//go:build windows && amd64

package voicemeeter

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingDLLAndRelativePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.dll")
	if _, e := Open(path); e == nil || !strings.Contains(e.Error(), path) {
		t.Fatal(e)
	}
	if _, e := Open("VoicemeeterRemote64.dll"); e == nil || !strings.Contains(e.Error(), "absolute") {
		t.Fatal(e)
	}
}
func TestNativeSignedLong(t *testing.T) {
	if result(uintptr(0xfffffffe)) != -2 {
		t.Fatal("unsigned ABI conversion")
	}
}
