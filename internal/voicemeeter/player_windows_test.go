//go:build windows && amd64

package voicemeeter

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Explicit opt-in: decode/run the supplied clip at -100 dB on the named endpoint.
// It never touches recorder transport or mixer routing.
func TestNativeClipPlayback(t *testing.T) {
	path := os.Getenv("SNOOFER_SOUNDBOARD_TEST_CLIP")
	if path == "" {
		t.Skip("set SNOOFER_SOUNDBOARD_TEST_CLIP for muted native playback")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dllPath, err := filepath.Abs("../../bin/snoofer-soundboard.dll")
	if err != nil {
		t.Fatal(err)
	}
	dll, err := windows.LoadDLL(dllPath)
	if err != nil {
		t.Fatal(err)
	}
	proc, err := dll.FindProc("SBDevices")
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, 16384)
	code, _, _ := proc.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if err := clipResult(code); err != nil {
		t.Fatal(err)
	}
	names := windows.UTF16ToString(buffer)
	if err := dll.Release(); err != nil {
		t.Fatal(err)
	}
	renderer := "DirectSound: Voicemeeter VAIO3 Input (VB-Audio Voicemeeter VAIO)"
	if !strings.Contains(names, renderer+"\n") {
		t.Fatalf("renderer missing; available:\n%s", names)
	}
	player, err := OpenClipPlayer(dllPath, renderer)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := player.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := player.Play(path, true); err != nil {
		t.Fatal(err)
	}
	// A second play must replace the graph without leaking or overlap.
	if err := player.Play(path, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		active, err := player.Poll()
		if err != nil {
			t.Fatal(err)
		}
		if !active {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("clip did not finish")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := player.Play(path+".missing", true); err == nil {
		t.Fatal("missing file played")
	}
	if err := player.Stop(); err != nil {
		t.Fatal(err)
	}
	missing, err := OpenClipPlayer(dllPath, "DirectSound: Missing Snoofer renderer")
	if err != nil {
		t.Fatal(err)
	}
	if err := missing.Play(path, true); err == nil {
		t.Error("missing endpoint fell back")
	}
	if err := missing.Close(); err != nil {
		t.Error(err)
	}
}
