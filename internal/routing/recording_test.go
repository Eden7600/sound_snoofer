package routing

import (
	"fmt"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"strings"
	"testing"
)

func recordingFixture(t *testing.T) (config.Config, model.Snapshot) {
	c, s := voiceFixture(t)
	c.Studio.Recording = &config.Recording{}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	r := model.RecorderSnapshot{Values: map[string]float32{}}
	for _, p := range model.RecorderParameters() {
		r.Values[p] = 0
	}
	r.Values["Recorder.stop"] = 1
	s.Recorder = &r
	return c, s
}

func TestRecordingFallbackAndAvailability(t *testing.T) {
	c, s := recordingFixture(t)
	c.Intent = c.VoiceIntent()
	c.Intent.Recording.MicEnabled = true
	c.Intent.Recording.ComputerEnabled = true
	apply := func() {
		p, e := Build(c, s)
		if e != nil {
			t.Fatal(e)
		}
		applyPlan(&s, p)
	}
	apply()
	s.Devices[1].Available = false
	apply()
	if s.Numbers["Strip[2].B1"] != 1 || s.Numbers["Strip[0].B1"] != 0 {
		t.Fatal("fallback")
	}
	s.Devices[1].Available = true
	apply()
	if s.Numbers["Strip[0].B1"] != 1 || s.Numbers["Strip[2].B1"] != 0 {
		t.Fatal("reconnect")
	}
	s.Devices[1].Available = false
	s.Devices = append(s.Devices, s.Devices[2])
	apply()
	if s.Numbers["Strip[2].B1"] != 0 {
		t.Fatal("ambiguous capture")
	}
	s.Devices[2].Available = false
	s.Devices[4].Available = false
	s.Devices[3].Available = false
	apply()
	if s.Numbers["Strip[5].B1"] != 1 {
		t.Fatal("computer lost with devices")
	}
	for _, n := range []int{0, 1, 2, 6} {
		if s.Numbers[fmt.Sprintf("Strip[%d].B1", n)] != 0 {
			t.Fatal("missing mic still captured")
		}
	}
}
func TestRecordingMatrix(t *testing.T) {
	for _, mode := range []string{"direct", "element"} {
		for _, tap := range []string{"pre", "post"} {
			for _, source := range []string{"desk", "lav", "webcam"} {
				for bits := 0; bits < 8; bits++ {
					t.Run(fmt.Sprint(mode, tap, source, bits), func(t *testing.T) {
						c, s := recordingFixture(t)
						i := c.VoiceIntent()
						c.Intent = i
						i.Mode = mode
						i.Source = source
						i.Enabled = bits&1 != 0
						i.Recording.MicEnabled = bits&2 != 0
						i.Recording.ComputerEnabled = bits&4 != 0
						i.Recording.MicTap = tap
						i.Playback["virtual:1"] = false
						p, e := Build(c, s)
						if e != nil {
							t.Fatal(e)
						}
						applyPlan(&s, p)
						for n := 0; n < 8; n++ {
							want := float32(0)
							mic := map[string]int{"desk": 0, "lav": 1, "webcam": 2}[source]
							if tap == "post" && mode == "element" {
								mic = 6
							}
							if (i.Recording.ComputerEnabled && n == 5) || (i.Enabled && i.Recording.MicEnabled && n == mic) {
								want = 1
							}
							if s.Numbers[fmt.Sprintf("Strip[%d].B1", n)] != want {
								t.Fatal(n, want, s.Numbers)
							}
						}
						p, e = Build(c, s)
						if e != nil || p.HasChanges() {
							t.Fatal("not converged", e)
						}
					})
				}
			}
		}
	}
}
func TestRecordingTransitionIsolationAndConflict(t *testing.T) {
	c, s := recordingFixture(t)
	c.Intent = c.VoiceIntent()
	c.Intent.Recording.MicEnabled = true
	p, _ := Build(c, s)
	applyPlan(&s, p)
	c.Intent.Recording.MicTap = "post"
	p, _ = Build(c, s)
	for _, op := range p.Topology.Transition {
		if op.Change && !strings.HasSuffix(op.Parameter, ".B1") {
			t.Fatal("voice interrupted", op)
		}
	}
	s.Recorder.Values["Recorder.stop"] = 0
	s.Recorder.Values["Recorder.record"] = 1
	p, _ = Build(c, s)
	if p.Topology.Recording.Blocked == "" {
		t.Fatal("conflict ignored")
	}
	for _, op := range p.Topology.Operations {
		if recordCell(op.Parameter) {
			t.Fatal("B1 not frozen")
		}
	}
	s.Recorder = nil
	p, e := Build(c, s)
	if e != nil || p.Topology.Voice == nil || p.Topology.Recording.Blocked == "" {
		t.Fatal("recorder failure blocked legacy voice", e)
	}
}
