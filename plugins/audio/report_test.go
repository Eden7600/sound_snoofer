package audio

import (
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
)

func find(t *testing.T, list []snoofer.Control, id string) snoofer.Control {
	t.Helper()
	for _, c := range list {
		if c.ID == id {
			if c.Kind != "connection" || !c.SurfaceOnly || len(c.Operations) != 0 || c.Connection == nil {
				t.Fatalf("%s is not a report: %+v", id, c)
			}
			return c
		}
	}
	t.Fatalf("missing %s", id)
	return snoofer.Control{}
}

func detail(c snoofer.Control, label string) string {
	for _, d := range c.Connection.Details {
		if d.Label == label {
			return d.Value
		}
	}
	return ""
}

func TestVoicemeeterReportLifecycle(t *testing.T) {
	var r reporter
	t0 := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	connected := control.State{Connected: true, Live: true, ObservedAt: t0, Health: "Healthy",
		Snapshot: model.Snapshot{Edition: 3},
		Remote:   model.RemoteInfo{DLLPath: `C:\VB\VoicemeeterRemote64.dll`, Login: 0, Version: "3.1.1.4"}}
	vm := find(t, r.reports(connected, t0), "audio.app-voicemeeter")
	if vm.Value != "Potato" || vm.Connection.State != snoofer.ConnectionConnected || vm.Connection.Endpoint != `C:\VB\VoicemeeterRemote64.dll` ||
		detail(vm, "Version") != "3.1.1.4" || detail(vm, "Login") != "Voicemeeter running" || detail(vm, "Required") != "Yes" || !vm.Connection.LastActivity.Equal(t0) {
		t.Fatalf("connected report %+v", vm.Connection)
	}
	lost := connected
	lost.Connected = false
	lost.Recorder = &model.RecorderSnapshot{Error: "VBVMR_IsParametersDirty: Voicemeeter is disconnected"}
	vm = find(t, r.reports(lost, t0.Add(time.Minute)), "audio.app-voicemeeter")
	if vm.Connection.State != snoofer.ConnectionDisconnected || !strings.Contains(vm.Connection.LastError, "disconnected") || !vm.Connection.Since.Equal(t0.Add(time.Minute)) {
		t.Fatalf("lost report %+v", vm.Connection)
	}
	vm = find(t, r.reports(connected, t0.Add(2*time.Minute)), "audio.app-voicemeeter")
	if vm.Connection.State != snoofer.ConnectionConnected || vm.Connection.LastError == "" || !vm.Connection.Since.Equal(t0.Add(2*time.Minute)) {
		t.Fatalf("recovered report lost history %+v", vm.Connection)
	}
}

func TestAudioIntegrationReports(t *testing.T) {
	var r reporter
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	s := control.State{Connected: true, ObservedAt: now,
		Snapshot: model.Snapshot{Edition: 3, Element: &model.ProcessStatus{Known: true},
			Numbers: map[string]float32{"Bus[0].device.sr": 48000}, Assignments: map[string]string{"A1": "Volt"}},
		Plan:           &routing.Plan{Topology: &routing.Topology{ASIOActive: true, ASIOName: "Universal Audio Volt"}},
		DefaultKind:    windowsaudio.Verified,
		DefaultsDetail: windowsaudio.Result{Status: "Verified", Kind: windowsaudio.Verified, Playback: "Voicemeeter Input (VB-Audio Voicemeeter VAIO)"},
		Intent:         &config.Intent{}}
	list := r.reports(s, now)
	if c := find(t, list, "audio.app-callback"); c.Connection.State != snoofer.ConnectionOff {
		t.Fatalf("callback without monitor %+v", c.Connection)
	}
	if c := find(t, list, "audio.app-recorder"); c.Connection.State != snoofer.ConnectionOff {
		t.Fatalf("recorder without recording intent %+v", c.Connection)
	}
	if c := find(t, list, "audio.app-asio"); c.Connection.State != snoofer.ConnectionConnected || c.Connection.Endpoint != "Universal Audio Volt" || detail(c, "Sample rate") != "48000 Hz" {
		t.Fatalf("asio %+v", c.Connection)
	}
	if c := find(t, list, "audio.app-element"); c.Connection.State != snoofer.ConnectionDisconnected || c.Value != "Not running" {
		t.Fatalf("element %+v", c.Connection)
	}
	if c := find(t, list, "audio.app-windows-defaults"); c.Connection.State != snoofer.ConnectionConnected || detail(c, "Playback target") == "" {
		t.Fatalf("defaults %+v", c.Connection)
	}

	s.Snapshot.Callback = &model.CallbackStatus{Active: true, Buffers: 120}
	s.Snapshot.Element = &model.ProcessStatus{Known: true, Running: true}
	s.DefaultKind = windowsaudio.Attention
	s.DefaultsDetail = windowsaudio.Result{Status: "Default protection suspended: repeated contention", Kind: windowsaudio.Attention, Suspended: true}
	s.Intent = &config.Intent{Recording: &config.RecordingChoices{}}
	s.Recorder = &model.RecorderSnapshot{Error: "recorder read failed"}
	list = r.reports(s, now.Add(time.Second))
	if c := find(t, list, "audio.app-callback"); c.Connection.State != snoofer.ConnectionConnected || detail(c, "Buffers") != "120" || c.Connection.LastActivity.IsZero() {
		t.Fatalf("active callback %+v", c.Connection)
	}
	if c := find(t, list, "audio.app-element"); c.Connection.State != snoofer.ConnectionConnected {
		t.Fatalf("running element %+v", c.Connection)
	}
	if c := find(t, list, "audio.app-windows-defaults"); c.Connection.State != snoofer.ConnectionAttention || detail(c, "Suspended") != "Yes" || !strings.Contains(c.Connection.LastError, "contention") {
		t.Fatalf("suspended defaults %+v", c.Connection)
	}
	if c := find(t, list, "audio.app-recorder"); c.Connection.State != snoofer.ConnectionError || c.Connection.LastError != "recorder read failed" {
		t.Fatalf("recorder error %+v", c.Connection)
	}
	s.Connected = false
	list = r.reports(s, now.Add(2*time.Second))
	for _, id := range []string{"audio.app-callback", "audio.app-asio", "audio.app-element"} {
		if c := find(t, list, id); c.Connection.State != snoofer.ConnectionUnknown {
			t.Fatalf("%s while Voicemeeter is disconnected: %+v", id, c.Connection)
		}
	}
}

func TestStalledEngineReports(t *testing.T) {
	var r reporter
	now := time.Date(2026, 10, 6, 9, 28, 0, 0, time.UTC)
	s := control.State{Connected: true, Live: true, Stalled: true, ObservedAt: now, Health: control.StallMessage,
		Snapshot: model.Snapshot{Edition: 3, Callback: &model.CallbackStatus{Active: true, Starting: 1}}}
	list := r.reports(s, now)
	vm, cb := find(t, list, "audio.app-voicemeeter"), find(t, list, "audio.app-callback")
	if vm.Value != "Engine stalled" || vm.Connection.State != snoofer.ConnectionAttention || vm.Connection.LastError != control.StallMessage {
		t.Fatalf("voicemeeter %+v", vm.Connection)
	}
	if cb.Value != "Stalled" || cb.Connection.State != snoofer.ConnectionAttention {
		t.Fatalf("callback %+v", cb.Connection)
	}
	s.Stalled = false
	s.Snapshot.Callback.Buffers = 940
	vm, cb = find(t, r.reports(s, now.Add(time.Minute)), "audio.app-voicemeeter"), find(t, r.reports(s, now.Add(time.Minute)), "audio.app-callback")
	if vm.Connection.State != snoofer.ConnectionConnected || vm.Connection.LastError != control.StallMessage || cb.Connection.State != snoofer.ConnectionConnected {
		t.Fatalf("recovery lost history: %+v %+v", vm.Connection, cb.Connection)
	}
}
