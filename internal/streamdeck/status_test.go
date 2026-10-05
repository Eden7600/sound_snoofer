package streamdeck

import (
	"bytes"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

func statusFixture() control.State {
	return control.State{Live: true, Connected: true, Intent: &config.Intent{Mode: "direct", Monitor: "off", Recording: &config.RecordingChoices{MicTap: "pre"}}, Snapshot: model.Snapshot{Numbers: map[string]float32{"Strip[0].Mute": 0, "Strip[6].Mute": 0, "Bus[0].Mute": 0, "Bus[1].Mute": 0}}, Plan: &routing.Plan{Topology: &routing.Topology{PlaybackTarget: "A2", Voice: &routing.VoiceStatus{Strip: 0}}}}
}

func TestUnrelatedStatusLeavesImagesUnchanged(t *testing.T) {
	s := statusFixture()
	baseline, _ := display(s, bindings)
	s.NoticeKind = control.NoticeError
	s.Notice = "Unrelated diagnostic"
	images, _ := display(s, bindings)
	for i := range images {
		if !bytes.Equal(images[i], baseline[i]) {
			t.Fatalf("global notice changed key %d", i)
		}
	}
	s.Plan.Topology.Operations = []routing.Operation{{Parameter: "Strip[0].A2", Change: true}}
	monitorView := present(s, bindings)
	images, _ = display(s, bindings)
	for i, binding := range bindings {
		changed := !bytes.Equal(images[i], baseline[i])
		if changed != (binding == "monitor") {
			t.Errorf("monitor edit changed %s: %v", binding, changed)
		}
	}
	s.Plan.Topology.Operations = []routing.Operation{{Parameter: "Recorder.mode.Loop", Change: true}}
	loopView := present(s, bindings)
	if loopView == monitorView {
		t.Fatal("cache missed change between pending settings")
	}
	images, _ = display(s, bindings)
	for i, binding := range bindings {
		changed := !bytes.Equal(images[i], baseline[i])
		if changed != (binding == "record-loop") {
			t.Errorf("loop edit changed %s: %v", binding, changed)
		}
	}
}

func TestBindingFeedbackAndAvailability(t *testing.T) {
	s := statusFixture()
	baseline := present(s, bindings)
	s.Feedback = map[string]control.Feedback{"record-start": {Kind: control.NoticeError}, "gain:mic": {Kind: control.NoticePending}}
	view := present(s, bindings)
	for i, binding := range bindings {
		if binding == "record-start" {
			if view.Keys[i].Value != "ERROR" {
				t.Fatal(view.Keys[i])
			}
			continue
		}
		if view.Keys[i] != baseline.Keys[i] {
			t.Errorf("feedback leaked to %s", binding)
		}
	}
	if view.Knobs[0] != baseline.Knobs[0] || view.Knobs[1] != baseline.Knobs[1] || view.Knobs[2].Value != "PENDING" {
		t.Fatal("gain feedback leaked", view.Knobs)
	}
	s.Feedback = nil
	s.Connected = false
	for _, key := range []string{"monitor", "mic-mute", "record-start"} {
		if got := keyValue(s, key); got != "UNAVAIL" {
			t.Errorf("%s = %s", key, got)
		}
	}
	if keyValue(s, "media-play") != "PRESS" || keyValue(s, "open-controls") != "OPEN" {
		t.Fatal("independent controls unavailable")
	}
	s.Live = false
	if keyValue(s, "monitor") != "PREVIEW" {
		t.Fatal("preview hidden")
	}
}

func TestNativeMuteIsScoped(t *testing.T) {
	s := statusFixture()
	s.Intent.MicMuted = true
	if keyValue(s, "mic-mute") != "PENDING" || keyValue(s, "speaker-mute") != "Off" {
		t.Fatal("mute pending leaked")
	}
	s.Snapshot.Numbers["Strip[0].Mute"] = 1
	if keyValue(s, "mic-mute") != "PENDING" {
		t.Fatal("ignored Element return readback")
	}
	s.Snapshot.Numbers["Strip[6].Mute"] = 1
	if keyValue(s, "mic-mute") != "On" {
		t.Fatal("applied mute pending")
	}
	s.Intent.MicMuted = false
	if keyValue(s, "mic-mute") != "On" {
		t.Fatal("native mute hidden")
	}
}
