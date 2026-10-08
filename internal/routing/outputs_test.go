package routing

import (
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// outputFixture: headphones (AirPods) are Playback and the speakers, which
// also match a playback pattern, are the Music slot.
func outputFixture(t *testing.T) (config.Config, model.Snapshot) {
	t.Helper()
	c, s := recordingFixture(t)
	c.Studio.Outputs = []config.Output{{ID: "music", Name: "Music", Device: "speakers", Sources: []string{"virtual:1"}}}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	s.Devices = append(s.Devices, model.Device{Name: "AirPods", Direction: "output", Driver: "wdm", Available: true})
	c.Intent = c.VoiceIntent()
	return c, s
}

func converge(t *testing.T, c config.Config, s *model.Snapshot) Plan {
	t.Helper()
	for n := 0; n < 4; n++ {
		p, e := Build(c, *s)
		if e != nil {
			t.Fatal(e)
		}
		if !p.HasChanges() {
			return p
		}
		applyPlan(s, p)
	}
	t.Fatal("routing did not converge")
	return Plan{}
}

func planned(p Plan, param string) (int, bool) {
	for _, op := range p.Topology.Operations {
		if op.Parameter == param {
			return op.Value, true
		}
	}
	return 0, false
}

func TestOutputSlotBesidePlayback(t *testing.T) {
	for _, profiles := range []*config.Profiles{nil, config.DefaultProfiles()} {
		testOutputSlotBesidePlayback(t, profiles)
	}
}

func testOutputSlotBesidePlayback(t *testing.T, profiles *config.Profiles) {
	c, s := outputFixture(t)
	c.Profiles = profiles
	s.Numbers["Strip[3].A3"] = 1 // A send the user made from an unmanaged strip.
	p := converge(t, c, &s)
	o := p.Topology.Outputs[0]
	if p.Topology.PlaybackTarget != "A2" || s.Assignments["A2"] != "AirPods" || o.Bus != "A3" || o.State != OutputOK || s.Assignments["A3"] != "speakers" {
		t.Fatal(p.Topology.PlaybackTarget, s.Assignments, o)
	}
	// Computer reaches both; the playback toggle and the slot switch are separate.
	if s.Numbers["Strip[5].A2"] != 1 || s.Numbers["Strip[5].A3"] != 1 {
		t.Fatal("computer routing", s.Numbers["Strip[5].A2"], s.Numbers["Strip[5].A3"])
	}
	if _, ok := planned(p, "Strip[3].A3"); ok || s.Numbers["Strip[3].A3"] != 1 {
		t.Fatal("unmanaged send on a slot bus was planned")
	}
	for _, option := range PlaybackOptions(c, s) {
		if option == "speakers" {
			t.Fatal("slot device offered as Playback")
		}
	}
}

func TestOutputMonitorSwitch(t *testing.T) {
	c, s := outputFixture(t)
	c.Intent.Source = "desk"
	c.Intent.Mode = "direct"
	c.Intent.Monitor = "pre"
	converge(t, c, &s)
	if s.Numbers["Strip[0].A2"] != 1 || s.Numbers["Strip[0].A3"] != 0 {
		t.Fatal("monitor must reach headphones only", s.Numbers["Strip[0].A2"], s.Numbers["Strip[0].A3"])
	}
	c.Intent.Outputs["music"][config.SourceMonitor] = true
	converge(t, c, &s)
	if s.Numbers["Strip[0].A3"] != 1 {
		t.Fatal("slot monitor switch ignored")
	}
	// Disabling the mic stack removes monitoring from every slot, not computer audio.
	c.Intent.Enabled = false
	converge(t, c, &s)
	if s.Numbers["Strip[0].A3"] != 0 || s.Numbers["Strip[0].A2"] != 0 || s.Numbers["Strip[5].A3"] != 1 {
		t.Fatal("mic stack disabled", s.Numbers["Strip[0].A3"], s.Numbers["Strip[5].A3"])
	}
}

func TestOutputDeviceDisconnects(t *testing.T) {
	c, s := outputFixture(t)
	converge(t, c, &s)
	s.Devices[3].Available = false // speakers
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	o := p.Topology.Outputs[0]
	if o.State != OutputMissing || o.Bus != "A3" || p.HasUnresolved() || p.Topology.PlaybackTarget != "A2" {
		t.Fatal(o, p.Topology.Unresolved, p.Topology.PlaybackTarget)
	}
	for _, op := range p.Topology.Operations {
		if op.Target == "A3" && op.Change {
			t.Fatal("missing slot device reassigned", op)
		}
	}
	s.Devices[3].Available = true
	if p = converge(t, c, &s); p.Topology.Outputs[0].Bus != "A3" || p.Topology.PlaybackTarget != "A2" {
		t.Fatal("reconnect moved outputs", p.Topology.Outputs[0], p.Topology.PlaybackTarget)
	}
}

func TestPlaybackTakesLastSlotWhenBusesRunOut(t *testing.T) {
	c, s := outputFixture(t)
	converge(t, c, &s)
	s.Assignments["A2"] = "tv" // Playback's bus taken by an unmanaged device.
	s.Assignments["A4"] = "other"
	s.Assignments["A5"] = "another"
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Topology.PlaybackTarget != "A3" || p.Topology.Outputs[0].State != OutputNoOutput || p.Topology.Outputs[0].Bus != "" {
		t.Fatal(p.Topology.PlaybackTarget, p.Topology.Outputs[0])
	}
}

func TestOutputTape(t *testing.T) {
	c, s := outputFixture(t)
	c.Intent.Recording.TapeRoutingManaged = true
	c.Intent.Outputs["music"][config.SourceTape] = true
	converge(t, c, &s)
	c.TapeListening = true
	s.Recorder.Values["Recorder.stop"] = 0
	s.Recorder.Values["Recorder.play"] = 1
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	for bus, want := range map[string]int{"A2": 1, "A3": 1, "A1": 0, "A4": 0} {
		if v, _ := planned(p, "Recorder."+bus); v != want {
			t.Fatalf("Recorder.%s = %d, want %d", bus, v, want)
		}
	}
	if got := strings.Join(p.Topology.OutputBuses(config.SourceTape), ","); got != "A3" {
		t.Fatal(got)
	}
}

// A slot without a device keeps its switches but holds no bus and plans no
// sends; assigning a device later places it.
func TestOutputWithoutDevice(t *testing.T) {
	c, s := outputFixture(t)
	c.Studio.Outputs = append(c.Studio.Outputs, config.Output{ID: "monitor", Name: "Monitor output", Sources: []string{config.SourceMonitor}})
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	c.Intent = c.VoiceIntent()
	before := len(s.Assignments)
	p := converge(t, c, &s)
	o := p.Topology.Outputs[1]
	if o.State != OutputEmpty || o.Bus != "" || !o.Receives(config.SourceMonitor) || len(s.Assignments) != before || s.Assignments["A4"] != "" {
		t.Fatal(o, s.Assignments)
	}
	c.Studio.Outputs[1].Device = "AirPods"
	c.Studio.Playback = c.Studio.Playback[1:] // Playback falls back to the next candidate.
	if p = converge(t, c, &s); p.Topology.Outputs[1].State != OutputOK || p.Topology.Outputs[1].Bus == "" {
		t.Fatal(p.Topology.Outputs[1])
	}
}
