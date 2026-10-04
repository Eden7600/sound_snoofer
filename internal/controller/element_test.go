package controller

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sound-snoofer/internal/model"
)

func TestElementFallbackRestoresPreferences(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Monitor = "post"
	c.Config.Intent.Recording.MicTap = "post"
	preferred := c.Config.Intent.Clone()
	apply := func() {
		t.Helper()
		p, err := c.Plan()
		if err != nil {
			t.Fatal(err)
		}
		if err = c.Apply(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	for _, status := range []*model.ProcessStatus{{Known: true}, nil, {Error: "denied"}, {Known: true, Running: true}} {
		b.s.Element = status
		apply()
		post := float32(0)
		pre := float32(1)
		if status != nil && status.Running {
			post, pre = 1, 0
		}
		if b.s.Numbers["Strip[6].B3"] != post || b.s.Numbers["Strip[0].B3"] != pre || b.s.Numbers["Strip[6].B1"] != post || b.s.Numbers["Strip[0].B1"] != pre || b.s.Numbers["Strip[6].A2"] != post || b.s.Numbers["Strip[0].A2"] != pre {
			t.Fatal("incorrect fallback", status, b.s.Numbers)
		}
		if !reflect.DeepEqual(preferred, c.Config.Intent) {
			t.Fatal("preference mutated")
		}
	}
	c.Config.Intent.Mode = "direct"
	apply()
	if c.Config.Intent.Monitor != "post" || c.Config.Intent.Recording.MicTap != "post" || b.s.Numbers["Strip[0].B1"] != 1 {
		t.Fatal("explicit Direct lost preference")
	}
	c.Config.Intent.Source = "off"
	c.Config.Intent.Enabled = false
	b.s.Element = nil
	apply()
	if b.s.Numbers["Strip[0].B3"] != 0 || b.s.Numbers["Strip[0].B1"] != 0 {
		t.Fatal("Off overridden")
	}
}
func TestElementExitDuringTransition(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Monitor = "post"
	p, _ := c.Plan()
	b.after = func() { b.s.Element = &model.ProcessStatus{Known: true} }
	if err := c.Apply(context.Background(), p); !errors.Is(err, ErrPlanChanged) {
		t.Fatal(err)
	}
}
func TestRehearsalHostFallback(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Recording.ToVST = true
	c.Config.Intent.NormalizeRecordingStage()
	apply := func() {
		t.Helper()
		p, err := c.Plan()
		if err != nil {
			t.Fatal(err)
		}
		if err = c.Apply(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	b.r.Values["Recorder.play"] = 1
	b.r.Values["Recorder.stop"] = 0
	b.s.Element = &model.ProcessStatus{Known: true}
	apply()
	if b.r.Values["Recorder.B2"] != 0 || b.r.State() != "Playing" || b.s.Numbers["Strip[0].B3"] != 1 {
		t.Fatal("rehearsal not detached")
	}
	if err := c.PlaySnippet(context.Background(), true); err == nil || !strings.Contains(err.Error(), "Element unavailable") {
		t.Fatal(err)
	}
	b.s.Element = &model.ProcessStatus{Known: true, Running: true}
	apply()
	if b.r.Values["Recorder.B2"] != 1 || b.s.Numbers["Strip[0].B3"] != 0 {
		t.Fatal("rehearsal not restored")
	}
	for _, p := range b.writesRecorder {
		if p == "Recorder.replay" || p == "Recorder.stop" || p == "Recorder.record" {
			t.Fatal("automatic transport", p)
		}
	}
}

func TestElementReturnDoesNotInterruptFallbackCapture(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Recording.ToVST = true
	c.Config.Intent.NormalizeRecordingStage()
	b.s.Element = &model.ProcessStatus{Known: true}
	p, err := c.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if err = c.Record(context.Background(), true, true); err != nil {
		t.Fatal(err)
	}
	b.s.Element = &model.ProcessStatus{Known: true, Running: true}
	p, err = c.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if b.r.State() != "Recording" || b.r.Values["Recorder.B2"] != 0 || b.s.Numbers["Strip[0].B2"] != 1 {
		t.Fatal("rehearsal interrupted capture")
	}
}
