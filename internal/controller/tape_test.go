package controller

import (
	"context"
	"fmt"
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

// Play routes the tape to tape-enabled output slots as well as Playback,
// and later plans keep it there.
func TestTapeListenBackToOutputSlot(t *testing.T) {
	ctx := context.Background()
	c, b := recorderController(t)
	c.Config.Studio.Outputs = []config.Output{{ID: "music", Name: "Music", Device: "speakers", Sources: []string{config.SourceTape}}}
	if err := c.Config.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Config.Intent = c.Config.VoiceIntent()
	c.Config.Intent.Recording.TapeRoutingManaged = true
	c.Config.Profiles = config.DefaultProfiles()
	converge(t, c)
	if b.s.Assignments["A3"] != "speakers" {
		t.Fatal("slot not placed", b.s.Assignments)
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil {
		t.Fatal(err)
	}
	if b.r.Values["Recorder.A2"] != 1 || b.r.Values["Recorder.A3"] != 1 || b.r.Values["Recorder.A1"] != 0 {
		t.Fatal(b.r.Values["Recorder.A1"], b.r.Values["Recorder.A2"], b.r.Values["Recorder.A3"])
	}
	if p, _ := c.Plan(); p.HasChanges() {
		t.Fatal("planner fights slot listening playback")
	}
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
	// Pause is one stop: Recorder.pause does not work in playback.
	writes := len(b.writesRecorder)
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Stopped" || !c.TapePaused {
		t.Fatal("pause", err, b.r.State(), c.TapePaused)
	}
	if got := b.writesRecorder[writes:]; len(got) != 1 || got[0] != "Recorder.stop" {
		t.Fatal("pause writes", got)
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Playing" || c.TapePaused {
		t.Fatal("resume", err, b.r.State(), c.TapePaused)
	}
	for _, command := range []TapeCommand{TapeRewind, TapeForward} {
		if err := c.Tape(ctx, command, true); err != nil {
			t.Fatal(err)
		}
	}
	// Stop while playing: the first stop pauses, the second returns to 0.
	writes = len(b.writesRecorder)
	if err := c.Tape(ctx, TapeStop, true); err != nil || b.r.State() != "Stopped" || c.TapePaused {
		t.Fatal("stop", err, b.r.State(), c.TapePaused)
	}
	if got := b.writesRecorder[writes:]; len(got) != 2 || got[0] != "Recorder.stop" || got[1] != "Recorder.stop" {
		t.Fatal("stop writes", got)
	}
	converge(t, c)
	if b.r.Values["Recorder.A2"] != 0 || c.TapeListening {
		t.Fatal("tape routing not cleared after stop")
	}
	if err := c.Tape(ctx, TapeStop, true); err == nil || !strings.Contains(err.Error(), "already at start") {
		t.Fatal("stop at the start", err)
	}
}

func TestTapeTransportDuringRecording(t *testing.T) {
	ctx := context.Background()
	c, b := recorderController(t)
	if err := c.Record(ctx, true, true); err != nil {
		t.Fatal(err)
	}
	// Rewind and Fast-forward never touch a recording.
	writes := len(b.writesRecorder)
	for _, command := range []TapeCommand{TapeRewind, TapeForward} {
		if err := c.Tape(ctx, command, true); err == nil || !strings.Contains(err.Error(), "recording") {
			t.Fatal(command, err)
		}
	}
	if len(b.writesRecorder) != writes || b.r.State() != "Recording" {
		t.Fatal("winding disturbed the recording", b.writesRecorder[writes:])
	}
	// Play/Pause pauses and resumes the recording.
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Paused" || b.r.Values["Recorder.record"] != 1 {
		t.Fatal("pause recording", err, b.r.State())
	}
	if err := c.Tape(ctx, TapeRewind, true); err == nil {
		t.Fatal("rewind during a paused recording")
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Recording" {
		t.Fatal("resume recording", err, b.r.State())
	}
	// Stop ends the recording through the verified stop.
	if err := c.Tape(ctx, TapeStop, true); err != nil || b.r.State() != "Stopped" {
		t.Fatal("end recording", err, b.r.State())
	}
	if c.Tape(ctx, TapePlayPause, false) == nil {
		t.Fatal("preview tape command")
	}
}

func TestStopAfterPauseReturnsToStart(t *testing.T) {
	ctx := context.Background()
	c, b := recorderController(t)
	c.Config.Intent.Recording.TapeRoutingManaged = true
	converge(t, c)
	if err := c.Tape(ctx, TapePlayPause, true); err != nil {
		t.Fatal(err)
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || !c.TapePaused {
		t.Fatal("pause", err)
	}
	writes := len(b.writesRecorder)
	if err := c.Tape(ctx, TapeStop, true); err != nil || c.TapePaused {
		t.Fatal("stop after pause", err, c.TapePaused)
	}
	if got := b.writesRecorder[writes:]; len(got) != 1 || got[0] != "Recorder.stop" {
		t.Fatal("one stop returns to the start", got)
	}
	// A state observed elsewhere clears Snoofer's paused knowledge.
	if err := c.Tape(ctx, TapePlayPause, true); err != nil {
		t.Fatal(err)
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || !c.TapePaused {
		t.Fatal("pause again", err)
	}
	b.r.Values["Recorder.play"], b.r.Values["Recorder.stop"] = 1, 0 // Played from Voicemeeter.
	if _, err := c.Plan(); err != nil || c.TapePaused {
		t.Fatal("observed playback kept the tape paused", err)
	}
}

func TestRecordDuringPlaybackStopsTapeFirst(t *testing.T) {
	ctx := context.Background()
	c, b := recorderController(t)
	b.r.Values["Recorder.play"], b.r.Values["Recorder.stop"] = 1, 0
	if err := c.Record(ctx, true, true); err != nil || b.r.State() != "Recording" {
		t.Fatal("record during playback", err, b.r.State())
	}
	if b.writesRecorder[0] != "Recorder.stop" || b.writesRecorder[len(b.writesRecorder)-1] != "Recorder.record" {
		t.Fatal("playback not stopped before recording", b.writesRecorder)
	}
	// A failed stop aborts the Start, and nothing is retried.
	c, b = recorderController(t)
	b.r.Values["Recorder.play"], b.r.Values["Recorder.stop"] = 1, 0
	b.fail = true
	if err := c.Record(ctx, true, true); err == nil || !strings.Contains(err.Error(), "not retried") || len(b.writesRecorder) != 1 {
		t.Fatal("failed stop", err, b.writesRecorder)
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

func TestTapeOwnershipHookRunsOnlyBeforePlayback(t *testing.T) {
	ctx := context.Background()
	c, b := recorderController(t)
	calls := 0
	c.BeforeTapePlay = func() error {
		calls++
		return fmt.Errorf("saved choices unwritable")
	}
	if err := c.Record(ctx, true, true); err != nil {
		t.Fatal(err)
	}
	// Pausing and resuming a recording never needs the ownership save.
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Paused" {
		t.Fatal("pause recording", err, b.r.State())
	}
	if err := c.Tape(ctx, TapePlayPause, true); err != nil || b.r.State() != "Recording" {
		t.Fatal("resume recording", err, b.r.State())
	}
	if err := c.Tape(ctx, TapeStop, true); err != nil || calls != 0 {
		t.Fatal("end recording", err, calls)
	}
	// Starting playback saves first; a failed save routes and plays nothing.
	writes := len(b.writesRecorder)
	if err := c.Tape(ctx, TapePlayPause, true); err == nil || calls != 1 || len(b.writesRecorder) != writes {
		t.Fatal("playback without ownership save", err, calls, b.writesRecorder[writes:])
	}
}
