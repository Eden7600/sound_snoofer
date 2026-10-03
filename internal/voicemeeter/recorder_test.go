package voicemeeter

import (
	"strings"
	"testing"
	"voice-snooter/internal/model"
)

type badRecorderAPI struct{ *fakeAPI }

func (a badRecorderAPI) GetNumber(p string) (float32, int32) {
	if strings.HasPrefix(p, "Recorder.") {
		return 0, -3
	}
	return 0, 0
}
func TestRecorderAllowlistAndIsolation(t *testing.T) {
	c, _ := connect(&fakeAPI{edition: 3})
	defer c.Close()
	for _, s := range model.RecorderSetup() {
		if e := c.SetRecorder(s.Parameter, s.Value); e != nil {
			t.Fatal(e)
		}
		if c.SetRecorder(s.Parameter, s.Value+1) == nil {
			t.Fatal(s)
		}
	}
	for _, p := range []string{"Recorder.pause", "Recorder.FileType", "Recorder.A1", "Recorder.record;Command.Restart"} {
		if c.SetRecorder(p, 1) == nil {
			t.Fatal(p)
		}
	}
	if c.SetRecorder("Recorder.record", 0) == nil {
		t.Fatal("bad REC")
	}
	bad, _ := connect(badRecorderAPI{&fakeAPI{edition: 3}})
	defer bad.Close()
	if _, e := bad.Recorder(); e == nil {
		t.Fatal("missing error")
	}
	if _, e := bad.Snapshot(); e != nil {
		t.Fatal("legacy broken", e)
	}
}
