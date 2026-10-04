package controller

import (
	"context"
	"strings"
	"testing"

	"sound-snoofer/internal/model"
)

func TestRehearsalRoutingAndTransport(t *testing.T) {
	c, b := recorderController(t)
	ctx := context.Background()
	c.Config.Intent.Recording.ToVST = true
	c.Config.Intent.NormalizeRecordingStage()
	c.Config.Intent.Recording.Loop = true
	c.Config.Intent.Monitor = "post"
	apply := func() {
		t.Helper()
		p, err := c.Plan()
		if err != nil {
			t.Fatal(err)
		}
		if err = c.Apply(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if c.PlaySnippet(ctx, true) == nil {
		t.Fatal("played before routes converged")
	}
	apply()
	if b.r.Values["Recorder.B2"] != 1 || b.r.Values["Recorder.mode.Loop"] != 1 || b.s.Numbers["Strip[6].A2"] != 1 {
		t.Fatal("missing rehearsal routing")
	}
	for _, p := range []string{"Strip[0].B1", "Strip[0].B2", "Strip[0].B3", "Strip[6].B3", "Strip[6].B2"} {
		if b.s.Numbers[p] != 0 {
			t.Fatal("leaking mic or return", p)
		}
	}
	if c.Record(ctx, true, true) == nil {
		t.Fatal("record during rehearsal")
	}
	if c.PlaySnippet(ctx, false) == nil {
		t.Fatal("preview playback")
	}
	if err := c.PlaySnippet(ctx, true); err != nil {
		t.Fatal(err)
	}
	if b.r.State() != "Playing" {
		t.Fatal(b.r.State())
	}
	c.Config.Intent.Monitor = "pre"
	apply()
	if b.r.Values["Recorder.A2"] != 1 || b.s.Numbers["Strip[6].A2"] != 0 {
		t.Fatal("dry and wet monitoring mixed")
	}
	c.Config.Intent.Recording.ToVST = false
	apply()
	if b.r.Values["Recorder.B2"] != 0 || b.r.Values["Recorder.A2"] != 0 || b.s.Numbers["Strip[0].B2"] != 1 {
		t.Fatal("normal voice not restored")
	}
	if err := c.Record(ctx, false, true); err != nil {
		t.Fatal(err)
	}
	if b.r.State() != "Stopped" {
		t.Fatal(b.r.State())
	}
}
func TestRehearsalPlayNotRetried(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Recording.ToVST = true
	p, _ := c.Plan()
	if err := c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	b.fail = true
	before := len(b.writesRecorder)
	err := c.PlaySnippet(context.Background(), true)
	if err == nil || !strings.Contains(err.Error(), "not retried") || len(b.writesRecorder) != before+1 {
		t.Fatal(err)
	}
}

type parameterCounter struct {
	*recorderFake
	full, fast int
}

func (b *parameterCounter) Snapshot() (model.Snapshot, error) {
	b.full++
	return b.recorderFake.Snapshot()
}
func (b *parameterCounter) ParameterSnapshot() (model.Snapshot, error) {
	b.fast++
	return b.recorderFake.Snapshot()
}
func TestMonitorUsesOnlyBoundaryInventoryReads(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Monitor = "pre"
	p, _ := c.Plan()
	if err := c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	counter := &parameterCounter{recorderFake: b}
	c.Backend = counter
	c.Config.Intent.Monitor = "post"
	p, _ = c.Plan()
	counter.full = 0
	if err := c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if counter.full != 2 || counter.fast != 4 {
		t.Fatalf("full=%d parameters=%d; want 2 boundary reads and 4 changed-send reads", counter.full, counter.fast)
	}
}

func TestRehearsalExitRepairsPartialTapeCleanup(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Intent.Recording.TapeRoutingManaged = true
	b.r.Values["Recorder.A2"] = 1
	b.r.Values["Recorder.B2"] = 0
	p, err := c.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if b.r.Values["Recorder.A2"] != 0 {
		t.Fatal("orphan dry tape send retained")
	}
}
