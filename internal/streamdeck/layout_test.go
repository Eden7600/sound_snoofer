package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"testing"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
)

func recorderState(state string) *model.RecorderSnapshot {
	r := &model.RecorderSnapshot{Values: map[string]float32{}}
	for _, p := range model.RecorderParameters() {
		r.Values[p] = 0
	}
	switch state {
	case "Stopped":
		r.Values["Recorder.stop"] = 1
	case "Recording":
		r.Values["Recorder.record"] = 1
	case "Paused":
		r.Values["Recorder.record"] = 1
		r.Values["Recorder.pause"] = 1
	case "Playing":
		r.Values["Recorder.play"] = 1
	default:
		return nil
	}
	return r
}
func TestDefaultWorkflowLayout(t *testing.T) {
	if len(bindings) != 36 || bindings[27] != "media-prev" || bindings[28] != "media-play" || bindings[29] != "media-next" {
		t.Fatal(bindings)
	}
	for n, key := range bindings {
		switch key {
		case "record-stop", "record-start", "snippet-play", "snippet-stop", "record-loop", "record-vst", "media-stop":
			t.Fatalf("obsolete control at %d", n)
		}
		if key == "" {
			if _, cmd := Action(Event{Encoder: -1, Key: n, Press: true}, statusFixture()); cmd != "" {
				t.Fatalf("blank %d dispatched", n)
			}
		}
	}
}
func TestRecordToggleFromObservedState(t *testing.T) {
	for _, state := range []string{"Stopped", "Recording", "Paused", "Playing", "Unknown"} {
		s := statusFixture()
		s.Recorder = recorderState(state)
		a, cmd := Action(Event{Encoder: -1, Binding: "record-toggle", Press: true}, s)
		if state == "Unknown" || state == "Playing" {
			if cmd != "" {
				t.Fatal("unsafe toggle", state)
			}
			continue
		}
		want := control.RecordStop
		if state == "Stopped" {
			want = control.RecordStart
		}
		if cmd != "audio" || a.Kind != want {
			t.Fatal(state, a, cmd)
		}
	}
	s := statusFixture()
	s.Recorder = recorderState("Stopped")
	s.Acks = map[string]control.Ack{}
	q := Queue{}
	event := Event{Encoder: -1, Binding: "record-toggle", Press: true}
	q.Push(event, s)
	q.Push(event, s)
	a, ok := q.Next(s)
	if !ok || a.Kind != control.RecordStart {
		t.Fatal(a)
	}
	if _, ok = q.Next(s); ok {
		t.Fatal("transport overlap")
	}
	s.Acks["streamdeck"] = control.Ack{ID: a.ID}
	s.Recorder = recorderState("Recording")
	a, ok = q.Next(s)
	if !ok || a.Kind != control.RecordStop {
		t.Fatal("queued press ignored observed recording", a)
	}
}
func TestSpeakerAndKnobShareMute(t *testing.T) {
	s := statusFixture()
	speaker := Event{Encoder: -1, Binding: "speaker-mute", Press: true}
	knob := Event{Encoder: 1, Press: true}
	for _, e := range []Event{speaker, knob, knob, speaker} {
		before := s.Intent.PlaybackMuted || s.Intent.BusMuted[1]
		a, cmd := Action(e, s)
		if cmd != "audio" {
			t.Fatal("missing action")
		}
		if err := control.EditBatch(s.Intent, a); err != nil {
			t.Fatal(err)
		}
		if s.Intent.PlaybackMuted == before || s.Intent.BusMuted[1] {
			t.Fatal("overlapping mute persisted", s.Intent)
		}
	}
	s.Intent.PlaybackMuted = true
	s.Intent.BusMuted[1] = true
	a, _ := Action(knob, s)
	control.EditBatch(s.Intent, a)
	if s.Intent.PlaybackMuted || s.Intent.BusMuted[1] {
		t.Fatal("cannot clear overlapping requests")
	}
	a, _ = Action(Event{Encoder: 0, Press: true}, s)
	control.EditBatch(s.Intent, a)
	if !s.Intent.BusMuted[0] || s.Intent.PlaybackMuted {
		t.Fatal("non-playback knob changed speaker preference")
	}
}
func TestIconDeckPreview(t *testing.T) {
	s := statusFixture()
	s.Recorder = recorderState("Stopped")
	s.Plan.Topology.Voice.EffectiveMode = "direct"
	s.Snapshot.Numbers["Bus[0].Gain"] = -3
	s.Snapshot.Numbers["Bus[1].Gain"] = -12
	s.Snapshot.Numbers["Strip[0].Gain"] = 0
	tiles, _ := display(s, bindings)
	preview := image.NewRGBA(image.Rect(0, 0, 9*124+12, 4*136+12))
	draw.Draw(preview, preview.Bounds(), &image.Uniform{color.RGBA{5, 9, 13, 255}}, image.Point{}, draw.Src)
	for n, tile := range tiles {
		im, err := jpeg.Decode(bytes.NewReader(tile))
		if err != nil {
			t.Fatal(err)
		}
		if im.Bounds().Dx() != 112 || im.Bounds().Dy() != 112 {
			t.Fatal("invalid tile size")
		}
		x, y := 12+(n%9)*124, 12+(n/9)*136
		// Undo the native panel rotation for the human preview.
		for py := 0; py < 112; py++ {
			for px := 0; px < 112; px++ {
				preview.Set(x+px, y+py, im.At(py, 111-px))
			}
		}
	}
	if path := os.Getenv("SNOOFER_DECK_PREVIEW"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, preview); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecordIconAndFallbackAreObserved(t *testing.T) {
	s := statusFixture()
	s.Recorder = recorderState("Stopped")
	before := present(s, bindings)
	s.Recorder = recorderState("Recording")
	after := present(s, bindings)
	if before.Keys[9].Icon != "record-toggle" || after.Keys[9].Icon != "record-stop" {
		t.Fatal("record icon does not follow transport")
	}
	for n := range after.Keys {
		if n != 9 && before.Keys[n] != after.Keys[n] {
			t.Fatalf("record changed key %d", n)
		}
	}
	s.Intent.Mode = "element"
	s.Plan.Topology.Voice.EffectiveMode = "direct"
	after = present(s, bindings)
	if after.Keys[3].Value != "direct*" || !after.Keys[3].Fallback {
		t.Fatal("processing preference shown as active", after.Keys[3])
	}
}
