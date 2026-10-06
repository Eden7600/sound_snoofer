//go:build windows

package windowsaudio

import (
	"math"
	"os"
	"runtime"
	"testing"
)

// Opt-in acceptance probe. SNOOFER_READ_ONLY_PROBE=1 lists live sessions;
// SNOOFER_SESSION_WRITE_PROBE=1 also rewrites the system sounds session's
// current volume and mute, unchanged, to check native argument passing.
func TestNativeSessions(t *testing.T) {
	if os.Getenv("SNOOFER_READ_ONLY_PROBE") != "1" {
		t.Skip("manual acceptance")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	b, err := OpenSessions()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	sessions, err := b.List()
	if err != nil {
		t.Fatal(err)
	}
	var system *Session
	for i, s := range sessions {
		peak, err := b.Peak(s.Key)
		t.Logf("%-24q pid=%-6d active=%-5t vol=%.2f muted=%-5t peak=%.3f icon=%dB device=%q path=%q err=%v",
			s.Name, s.PID, s.Active, s.Volume, s.Muted, peak, len(s.Icon), s.Device, s.Path, err)
		if s.Path == SystemSounds && system == nil {
			system = &sessions[i]
		}
	}
	if os.Getenv("SNOOFER_SESSION_WRITE_PROBE") != "1" || system == nil {
		return
	}
	if err := b.SetVolume(system.Key, system.Volume); err != nil {
		t.Fatal(err)
	}
	if err := b.SetMute(system.Key, system.Muted); err != nil {
		t.Fatal(err)
	}
	again, err := b.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range again {
		if s.Key == system.Key && (math.Abs(s.Volume-system.Volume) > 0.001 || s.Muted != system.Muted) {
			t.Fatalf("round trip changed system sounds: %.3f→%.3f muted %t→%t", system.Volume, s.Volume, system.Muted, s.Muted)
		}
	}
}
