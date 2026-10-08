package routing

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// genericFixture: a USB microphone on input 1, a stereo pair on interface
// channels 3/4 on input 2, and a mono mic the interface does not carry.
func genericFixture(t *testing.T) (config.Config, model.Snapshot) {
	t.Helper()
	c, s := voiceFixture(t)
	c.Studio.FallbackMic = nil
	c.Studio.Microphones = []config.Microphone{
		{ID: "yeti", Name: "Yeti", Devices: []config.Candidate{{Driver: "wdm", Pattern: "Yeti"}}},
		{ID: "pair", Name: "Stereo pair"},
		{ID: "spare", Name: "Spare"},
	}
	c.Studio.ASIO[0].Inputs = config.MicInputs{"pair": {3, 4}}
	c.Profiles = &config.Profiles{Microphones: []string{"yeti", "pair"}}
	c.Studio.Voice.Source = "auto"
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "auto"
	s.Devices = append(s.Devices, model.Device{Name: "Yeti", Direction: "input", Driver: "wdm", Available: true})
	for n := 4; n < 10; n++ {
		s.Numbers[fmt.Sprintf("Patch.asio[%d]", n)] = 0
	}
	return c, s
}

func TestGenericMicrophoneRouting(t *testing.T) {
	c, s := genericFixture(t)
	if got := MicrophoneOptions(c, s); !slices.Equal(got, []string{"yeti", "pair", "off"}) {
		t.Fatal("options", got)
	}
	p := converge(t, c, &s)
	if p.Topology.Voice.Effective != "yeti" || p.Topology.Voice.Strip != 0 || s.Assignments["input:1"] != "Yeti" {
		t.Fatal(p.Topology.Voice, s.Assignments["input:1"])
	}
	// The pair feeds its own input in stereo; the unmapped mic is silent;
	// the device microphone's patch cells are left alone.
	if !slices.Equal(patches(s), []float32{0, 0, 3, 4}) || s.Numbers["Patch.asio[4]"] != 0 || s.Assignments["input:2"] != "" || s.Assignments["input:3"] != "" {
		t.Fatal(patches(s), s.Assignments)
	}
	if !slices.Equal(p.Topology.MicStrips, []int{0, 1, 2, 6}) {
		t.Fatal(p.Topology.MicStrips)
	}
	// The USB microphone disconnects: Auto moves to the pair.
	s.Devices[len(s.Devices)-1].Available = false
	if p = converge(t, c, &s); p.Topology.Voice.Effective != "pair" || p.Topology.Voice.Strip != 1 {
		t.Fatal(p.Topology.Voice)
	}
}

func TestGenericMicrophoneConflicts(t *testing.T) {
	c, s := genericFixture(t)
	s.Assignments["input:1"] = "Some other mic"
	if _, e := Build(c, s); e == nil || !strings.Contains(e.Error(), "input:1 is occupied") {
		t.Fatal("overwrote an unmanaged input", e)
	}
	c, s = genericFixture(t)
	c.VR = &config.VR{Input: 4}
	c.Studio.Microphones = append(c.Studio.Microphones, config.Microphone{ID: "extra", Name: "Extra"})
	if _, e := Build(c, s); e == nil || !strings.Contains(e.Error(), "VR input 4") {
		t.Fatal("microphone on the VR input", e)
	}
}
