package controller

import (
	"context"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
)

func converge(t *testing.T, c *Controller) {
	t.Helper()
	p, err := c.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func TestTapeListenBack(t *testing.T) {
	testTapeListenBack(t, nil)
}

// Profile resolution restores the saved base configuration on every plan; the
// runtime listening flag must survive that.
func TestTapeListenBackWithProfiles(t *testing.T) {
	testTapeListenBack(t, &config.Profiles{Microphones: []string{"desk"}})
}

func testTapeListenBack(t *testing.T, profiles *config.Profiles) {
	ctx := context.Background()
	c, b := recorderController(t)
	c.Config.Intent.Recording.TapeRoutingManaged = true
	c.Config.Profiles = profiles
	converge(t, c)
	b.r.Values["Recorder.A1"] = 1 // Stale routing from elsewhere.
	if err := c.Tape(ctx, TapePlayPause, true); err != nil {
		t.Fatal(err)
	}
	if b.r.State() != "Playing" || b.r.Values["Recorder.A2"] != 1 || b.r.Values["Recorder.A1"] != 0 || !c.TapeListening {
		t.Fatal(b.r.State(), b.r.Values["Recorder.A1"], b.r.Values["Recorder.A2"])
	}
	if p, _ := c.Plan(); p.HasChanges() {
		t.Fatal("planner fights listening playback")
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Paused" {
		t.Fatal("pause", err, b.r.State())
	}
	if p, _ := c.Plan(); p.HasChanges() {
		t.Fatal("planner fights paused playback")
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Playing" {
		t.Fatal("resume", err, b.r.State())
	}
	for _, command := range []TapeCommand{TapeRewind, TapeForward} {
		if err := c.Tape(ctx, command, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Tape(ctx, TapeStop, true); err != nil || b.r.State() != "Stopped" {
		t.Fatal("stop", err, b.r.State())
	}
	converge(t, c)
	if b.r.Values["Recorder.A2"] != 0 || c.TapeListening {
		t.Fatal("tape routing not cleared after stop")
	}
	if c.Tape(ctx, TapeStop, true) == nil {
		t.Fatal("stop while not playing")
	}
}

func TestTapeNeverTouchesRecording(t *testing.T) {
	ctx := context.Background()
	c, b := recorderController(t)
	if err := c.Record(ctx, true, true); err != nil {
		t.Fatal(err)
	}
	writes := len(b.writesRecorder)
	for _, command := range []TapeCommand{TapePlayPause, TapeStop, TapeRewind, TapeForward} {
		if err := c.Tape(ctx, command, true); err == nil || !strings.Contains(err.Error(), "recording") {
			t.Fatal(command, err)
		}
	}
	if len(b.writesRecorder) != writes || b.r.State() != "Recording" {
		t.Fatal("tape command disturbed the recording", b.writesRecorder[writes:])
	}
	if c.Tape(ctx, TapePlayPause, false) == nil {
		t.Fatal("preview tape command")
	}
}

func TestTapeRespectsPausedSendsAndRehearsal(t *testing.T) {
	ctx := context.Background()
	c, b := recorderController(t)
	c.Config.Intent.PauseSends = true
	if err := c.Tape(ctx, TapePlayPause, true); err == nil || !strings.Contains(err.Error(), "Sends paused") || len(b.writesRecorder) != 0 {
		t.Fatal(err, b.writesRecorder)
	}
	c.Config.Intent.PauseSends = false
	c.Config.Intent.Mode = "element"
	c.Config.Intent.Recording.ToVST = true
	if err := c.Tape(ctx, TapePlayPause, true); err == nil || !strings.Contains(err.Error(), "VST") {
		t.Fatal(err)
	}
}
