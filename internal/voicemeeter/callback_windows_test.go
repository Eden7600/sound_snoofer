//go:build windows && amd64

package voicemeeter

import (
	"os"
	"runtime"
	"testing"
	"time"

	"sound-snoofer/internal/ownership"
)

// TestNativeCallbackMonitor exercises the production bridge without any audio setter.
func TestNativeCallbackMonitor(t *testing.T) {
	if os.Getenv("SNOOFER_CALLBACK_NATIVE") != "1" {
		t.Skip("opt-in native callback lifecycle")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	release, err := ownership.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if release != nil {
			release()
		}
	}()
	c, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			release = nil
			t.Error(err)
		}
	}()
	if err := c.SetCallback(true, nil); err != nil {
		t.Fatal(err)
	}
	a := c.api.(*winAPI)
	first := a.CallbackStatus()
	if first == nil || !first.Active || first.Error != "" {
		t.Fatal(first)
	}
	// A second enable must not attempt to register over our existing slot.
	if err := c.SetCallback(true, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	observed, err := c.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	next := observed.Callback
	if next == nil {
		t.Fatal("callback evidence missing from main snapshot")
	}
	if next.Error != "" {
		t.Fatal(next.Error)
	}
	t.Logf("production callback buffers: %d -> %d, synchronized: %d -> %d", first.Buffers, next.Buffers, first.Synced, next.Synced)
	// Restarting after a stream change keeps the same slot and resumes delivery.
	if err := c.RestartCallback(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	restarted := a.CallbackStatus()
	if restarted == nil || !restarted.Active || restarted.Error != "" {
		t.Fatal(restarted)
	}
	t.Logf("after restart: buffers %d -> %d, starting %d -> %d", next.Buffers, restarted.Buffers, next.Starting, restarted.Starting)
	if err := c.SetCallback(false, nil); err != nil {
		t.Fatal(err)
	}
	if a.CallbackStatus() != nil {
		t.Fatal("monitor remained active")
	}
	// Register again to exercise shutdown/re-enable ownership.
	if err := c.SetCallback(true, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.SetCallback(false, nil); err != nil {
		t.Fatal(err)
	}
}
