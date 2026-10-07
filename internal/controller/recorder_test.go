package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type recorderFake struct {
	afterRecorder func()
	*voiceBackend
	r              model.RecorderSnapshot
	writesRecorder []string
	fail, ignore   bool
}

func (b *recorderFake) Recorder() (model.RecorderSnapshot, error) { return b.r, nil }
func (b *recorderFake) SetRecorder(p string, v int) error {
	b.writesRecorder = append(b.writesRecorder, p)
	if b.fail {
		return fmt.Errorf("lost connection")
	}
	if b.ignore {
		return nil
	}
	b.r.Values[p] = float32(v)
	if b.afterRecorder != nil {
		b.afterRecorder()
	}
	if p == "Recorder.replay" {
		b.r.Values["Recorder.play"] = 1
		b.r.Values["Recorder.stop"] = 0
	}
	if p == "Recorder.record" {
		b.r.Values["Recorder.stop"] = 0
	}
	if p == "Recorder.play" {
		b.r.Values["Recorder.stop"] = 0
		b.r.Values["Recorder.pause"] = 0
	}
	if p == "Recorder.stop" {
		b.r.Values["Recorder.record"] = 0
		b.r.Values["Recorder.pause"] = 0
		b.r.Values["Recorder.play"] = 0
	}
	return nil
}

func TestPartialPreparationStopsOnExternalTransport(t *testing.T) {
	c, b := recorderController(t)
	b.afterRecorder = func() { b.r.Values["Recorder.play"] = 1; b.r.Values["Recorder.stop"] = 0 }
	if e := c.Record(context.Background(), true, true); e == nil {
		t.Fatal("continued preparation")
	}
	if len(b.writesRecorder) != 1 {
		t.Fatal(b.writesRecorder)
	}
}

func TestMicOffOverridesRecorderConflict(t *testing.T) {
	c, b := recorderController(t)
	b.r.Error = "recorder unavailable"
	c.Config.Intent.Source = "off"
	c.Config.Intent.Enabled = true // Off wins even over contradictory legacy enabled.
	p, e := c.Plan()
	if e != nil {
		t.Fatal(e)
	}
	_ = c.Apply(context.Background(), p)
	for _, n := range []int{0, 1, 2, 6} {
		for bus := 1; bus <= 3; bus++ {
			if b.s.Numbers[fmt.Sprintf("Strip[%d].B%d", n, bus)] != 0 {
				t.Fatal("microphone remained connected")
			}
		}
	}
	if len(b.writesRecorder) != 0 {
		t.Fatal("transport changed")
	}
}
func TestRecordingDisableFailureAndDrift(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Recording.MicTap = "post"
	p, _ := c.Plan()
	b.failAt = b.attempt + 1
	if e := c.Apply(context.Background(), p); e == nil {
		t.Fatal("disable failure ignored")
	}
	if b.s.Numbers["Strip[6].B1"] != 0 {
		t.Fatal("enabled new tap after failure")
	}
	b.failAt = 0
	p, _ = c.Plan()
	if e := c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	b.s.Numbers["Strip[7].B1"] = 1
	p, _ = c.Plan()
	if e := c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if b.s.Numbers["Strip[7].B1"] != 0 {
		t.Fatal("B1 drift")
	}
	if len(b.writesRecorder) != 0 {
		t.Fatal("routing started recorder")
	}
}
func recorderController(t *testing.T) (*Controller, *recorderFake) {
	c, b := voiceController(t)
	c.Config.Studio.Recording = &config.Recording{}
	c.Config.Validate()
	c.Config.Intent = c.Config.VoiceIntent()
	c.Config.Intent.Recording.MicEnabled = true
	r := model.RecorderSnapshot{Values: map[string]float32{}}
	for _, p := range model.RecorderParameters() {
		r.Values[p] = 0
	}
	r.Values["Recorder.stop"] = 1
	rb := &recorderFake{voiceBackend: b, r: r}
	c.Backend = rb
	p, e := c.Plan()
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	return c, rb
}
func TestRecorderTransportPreparation(t *testing.T) {
	c, b := recorderController(t)
	b.r.Values["Recorder.B1"] = 1
	b.r.Values["Recorder.ArmBus[0]"] = 1
	b.r.Values["Recorder.A1"] = 1
	b.r.Values["Recorder.FileType"] = 77
	if e := c.Record(context.Background(), true, true); e != nil {
		t.Fatal(e)
	}
	if b.r.State() != "Recording" || !b.r.Ready() {
		t.Fatal(b.r)
	}
	if b.r.Values["Recorder.A1"] != 1 || b.r.Values["Recorder.FileType"] != 77 {
		t.Fatal("unowned changed")
	}
	n := len(b.writesRecorder)
	if e := c.Record(context.Background(), true, true); e != nil || len(b.writesRecorder) != n {
		t.Fatal("Start repeated", e)
	}
	b.r.Values["Recorder.mode.recbus"] = 0
	if e := c.Record(context.Background(), true, true); e == nil {
		t.Fatal("active conflict")
	}
	if e := c.Record(context.Background(), false, true); e != nil {
		t.Fatal(e)
	}
	n = len(b.writesRecorder)
	c.Record(context.Background(), false, true)
	if len(b.writesRecorder) != n {
		t.Fatal("Stop repeated")
	}
	b.r.Values["Recorder.B2"] = 1
	c.protectRecorder(context.Background())
	if b.r.Values["Recorder.B2"] != 0 {
		t.Fatal("playback not protected")
	}
}
func TestPausedSendsLeaveRecorderRouting(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.PauseSends = true
	b.r.Values["Recorder.B2"] = 1
	if e := c.protectRecorder(context.Background()); e != nil || b.r.Values["Recorder.B2"] != 1 {
		t.Fatal("tape protection wrote while sends paused", e)
	}
	e := c.Record(context.Background(), true, true)
	if e == nil || !strings.Contains(e.Error(), "Sends paused") || len(b.writesRecorder) != 0 {
		t.Fatal("Start prepared the recorder while sends paused", e, b.writesRecorder)
	}
	// An already prepared recorder still starts.
	for _, s := range model.RecorderSetup() {
		b.r.Values[s.Parameter] = float32(s.Value)
	}
	if e := c.Record(context.Background(), true, true); e != nil || b.r.State() != "Recording" {
		t.Fatal(e, b.r.State())
	}
}
func TestRecorderGuardsAndNoRetry(t *testing.T) {
	for _, kind := range []string{"dry", "empty", "pending", "paused", "playing", "unknown", "fail", "timeout"} {
		t.Run(kind, func(t *testing.T) {
			c, b := recorderController(t)
			live := true
			switch kind {
			case "dry":
				live = false
			case "empty":
				c.Config.Intent.Recording.MicEnabled = false
				p, _ := c.Plan()
				c.Apply(context.Background(), p)
			case "pending":
				b.s.Numbers["Strip[0].B1"] = 0
			case "paused":
				b.r.Values["Recorder.pause"] = 1
			case "playing":
				b.r.Values["Recorder.play"] = 1
			case "unknown":
				b.r.Error = "read failed"
			case "fail", "timeout":
				for _, s := range model.RecorderSetup() {
					b.r.Values[s.Parameter] = float32(s.Value)
				}
				b.fail = kind == "fail"
				b.ignore = kind == "timeout"
			}
			e := c.Record(context.Background(), true, live)
			if e == nil {
				t.Fatal("guard missing")
			}
			if kind == "fail" || kind == "timeout" {
				if len(b.writesRecorder) != 1 || !strings.Contains(e.Error(), "not retried") {
					t.Fatal(e, b.writesRecorder)
				}
				c.Step(context.Background(), false)
				if len(b.writesRecorder) != 1 {
					t.Fatal("auto retry")
				}
			} else if len(b.writesRecorder) != 0 {
				t.Fatal("unexpected writes")
			}
		})
	}
}
