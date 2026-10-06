package control

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

func callbackFixture() (config.Config, model.Snapshot) {
	c := config.Config{Studio: &config.Studio{ASIO: []config.ASIOInterface{{ASIORegex: regexp.MustCompile("^Universal Audio Volt$"), PresenceRegex: regexp.MustCompile("Volt"), Inputs: [2]int{1, 2}}}}}
	s := model.Snapshot{Edition: 3, Assignments: map[string]string{"A1": "Universal Audio Volt"},
		Devices:  []model.Device{{Name: "Universal Audio Volt", Driver: "asio", Direction: "output"}, {Name: "Volt input", ID: "physical", Driver: "wdm", Direction: "input", Available: true}},
		Callback: &model.CallbackStatus{Active: true, Starting: 1, Buffers: 100, Synced: 100}}
	return c, s
}
func qualify(h *callbackHealth, c config.Config, s model.Snapshot, start time.Time) bool {
	fault := false
	for n := 0; n <= 12; n++ {
		fault, _ = h.update(c, s, start.Add(time.Duration(n)*500*time.Millisecond))
	}
	return fault
}
func TestCallbackDisconnectGraceAndSilence(t *testing.T) {
	c, s := callbackFixture()
	now := time.Now()
	h := callbackHealth{}
	// Advancing silent buffers are healthy regardless of level/sample-rate observations.
	for n := 0; n < 30; n++ {
		s.Callback.Buffers++
		s.Callback.Synced++
		if fault, _ := h.update(c, s, now.Add(time.Duration(n)*500*time.Millisecond)); fault {
			t.Fatal("silence triggered recovery")
		}
	}
	now = now.Add(15 * time.Second)
	if !qualify(&h, c, s, now) {
		t.Fatal("stopped callbacks not detected")
	}
	s.Devices[1].Available = false
	if fault, reason := h.update(c, s, now.Add(6500*time.Millisecond)); fault || !strings.Contains(reason, "not present") {
		t.Fatal(fault, reason)
	}
	s.Devices[1].Available = true
	now = now.Add(7 * time.Second)
	for n := 0; n < 10; n++ {
		if fault, _ := h.update(c, s, now.Add(time.Duration(n)*500*time.Millisecond)); fault {
			t.Fatal("reconnection grace bypassed")
		}
	}
	if !qualify(&h, c, s, now.Add(5*time.Second)) {
		t.Fatal("reconnected stall not detected")
	}
	if fault, _ := h.update(c, s, now.Add(time.Minute)); fault {
		t.Fatal("resume gap bypassed grace")
	}
}
func TestCallbackUnknownIdentityAndLifecycle(t *testing.T) {
	for _, kind := range []string{"missing", "error", "ambiguous", "other-a1", "edition", "ended"} {
		t.Run(kind, func(t *testing.T) {
			c, s := callbackFixture()
			switch kind {
			case "missing":
				s.Callback = nil
			case "error":
				s.Callback.Error = "slot owned"
			case "ambiguous":
				s.Devices = append(s.Devices, s.Devices[1])
			case "other-a1":
				s.Assignments["A1"] = "other"
			case "edition":
				s.Edition = 0
			case "ended":
				s.Callback.Ending = 1
			}
			h := callbackHealth{}
			if qualify(&h, c, s, time.Now()) {
				t.Fatal("unsafe evidence accepted")
			}
		})
	}
	c, s := callbackFixture()
	h := callbackHealth{}
	now := time.Now()
	qualify(&h, c, s, now)
	s.Callback.Changes++
	if fault, _ := h.update(c, s, now.Add(6500*time.Millisecond)); fault {
		t.Fatal("stream change bypassed grace")
	}
}
func TestRecoveryRequiresAdvancingCallbacksAndPersistsUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	r := newRecovery(path)
	b := &restarter{}
	r.callback = callbackHealth{enabled: true, buffers: 100, synced: 100}
	now := time.Now()
	if err := r.restart(b, true, now); err != nil {
		t.Fatal(err)
	}
	_, s := callbackFixture()
	s.Numbers = map[string]float32{"Bus[0].device.sr": 48000}
	r.observe(s, now.Add(2*time.Second))
	r.observe(s, now.Add(3*time.Second))
	if !r.pending {
		t.Fatal("stale sample rate verified restart")
	}
	s.Callback.Buffers++
	s.Callback.Synced++
	r.observe(s, now.Add(4*time.Second))
	s.Callback.Buffers++
	s.Callback.Synced++
	r.observe(s, now.Add(5*time.Second))
	if r.pending || r.status != "Engine restarted; no stall detected" {
		t.Fatal(r.status)
	}
	b.fail = true
	if r.restart(b, true, now.Add(2*time.Minute)) == nil {
		t.Fatal("expected ambiguous result")
	}
	restored := newRecovery(path)
	if restored.automaticPermit(now.Add(4*time.Minute), true, "Stopped") == nil {
		t.Fatal("uncertain command retried after relaunch")
	}
	s.Callback.Buffers++
	s.Callback.Synced++
	restored.observe(s, now.Add(5*time.Minute))
	s.Callback.Buffers++
	s.Callback.Synced++
	restored.observe(s, now.Add(5*time.Minute+time.Second))
	if newRecovery(path).blocked {
		t.Fatal("observed recovery did not clear uncertain outcome")
	}
}

type callbackBackend struct {
	Client
	s         model.Snapshot
	transport model.RecorderSnapshot
	calls     int
}

func (b *callbackBackend) Snapshot() (model.Snapshot, error)         { return b.s, nil }
func (b *callbackBackend) Recorder() (model.RecorderSnapshot, error) { return b.transport, nil }
func (b *callbackBackend) RestartEngine() error                      { b.calls++; return nil }
func TestAutomaticDispatchFreshGuards(t *testing.T) {
	c, s := callbackFixture()
	// A stable off voice profile with Volt playback makes a complete routing plan.
	c.Studio.Voice = &config.Voice{}
	c.Studio.Playback = append(c.Studio.Playback, config.Candidate{Driver: "asio", Pattern: c.Studio.ASIO[0].ASIOPattern, Regex: c.Studio.ASIO[0].ASIORegex})
	c.Intent = &config.Intent{Version: 1, Source: "off", Mode: "direct", Monitor: "off", AutoRecover: true}
	s.Numbers = map[string]float32{}
	for i := 0; i < 4; i++ {
		s.Numbers[fmt.Sprintf("Patch.asio[%d]", i)] = 0
	}
	for i := 0; i < 8; i++ {
		for j := 1; j <= 5; j++ {
			s.Numbers[fmt.Sprintf("Strip[%d].A%d", i, j)] = 0
		}
		for j := 1; j <= 3; j++ {
			s.Numbers[fmt.Sprintf("Strip[%d].B%d", i, j)] = 0
		}
	}
	for _, slot := range model.Slots(3) {
		if slot != "A1" {
			s.Assignments[slot] = ""
		}
	}
	p, err := routing.Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range p.Topology.Operations {
		if op.Device != nil {
			s.Assignments[op.Target] = op.Device.Name
		} else {
			s.Numbers[op.Parameter] = float32(op.Value)
		}
	}
	for _, transport := range []string{"Stopped", "Recording", "Paused", "Playing", "Unknown"} {
		t.Run(transport, func(t *testing.T) {
			b := &callbackBackend{s: s, transport: model.RecorderSnapshot{Values: map[string]float32{}}}
			for _, key := range model.RecorderParameters() {
				b.transport.Values[key] = 0
			}
			switch transport {
			case "Stopped":
				b.transport.Values["Recorder.stop"] = 1
			case "Recording":
				b.transport.Values["Recorder.record"] = 1
			case "Paused":
				b.transport.Values["Recorder.pause"] = 1
			case "Playing":
				b.transport.Values["Recorder.play"] = 1
			}
			r := newRecovery(filepath.Join(t.TempDir(), "config"))
			now := time.Now()
			qualify(&r.callback, c, s, now.Add(-6*time.Second))
			o := &observed{Client: b}
			if r.automaticRestart(o, c, false, now) == nil || b.calls != 0 {
				t.Fatal("preview mutation")
			}
			err := r.automaticRestart(o, c, true, now)
			if transport == "Stopped" {
				if err != nil || b.calls != 1 {
					t.Fatal("eligible recovery", err, b.calls)
				}
			} else if err == nil || b.calls != 0 {
				t.Fatal("transport guard", err, b.calls)
			}
		})
	}
}
func (b *callbackBackend) SetRecorder(string, int) error {
	return fmt.Errorf("unexpected recorder write")
}

// wdmFixture reproduces the 2026-10-06 incident: a WDM SteelSeries on A1, no Volt.
func wdmFixture() (config.Config, model.Snapshot) {
	c, s := callbackFixture()
	s.Assignments["A1"] = "Speakers (3- SteelSeries Arena 9)"
	s.Devices = []model.Device{
		{Name: "Universal Audio Volt", Driver: "asio", Direction: "output", Available: true}, // Installed driver only.
		{Name: "Speakers (3- SteelSeries Arena 9)", ID: "ss", Driver: "wdm", Direction: "output", Available: true},
		{Name: "Speakers (3- SteelSeries Arena 9)", ID: "ss", Driver: "mme", Direction: "output", Available: true},
	}
	return c, s
}

func TestCallbackStallOnWDMA1(t *testing.T) {
	c, s := wdmFixture()
	h := callbackHealth{}
	if !qualify(&h, c, s, time.Now()) {
		t.Fatal("stopped callbacks on a WDM A1 not detected")
	}
	if _, reason := h.update(c, s, time.Now().Add(time.Hour)); reason == "" {
		t.Fatal("no reason")
	}

	c, s = wdmFixture()
	s.Devices = s.Devices[:1] // SteelSeries unplugged; only the Volt ASIO driver remains enumerated.
	h = callbackHealth{}
	if qualify(&h, c, s, time.Now()) {
		t.Fatal("absent WDM A1 qualified")
	}
	if _, reason := callbackTarget(c, s); !strings.Contains(reason, "not present") {
		t.Fatal(reason)
	}

	c, s = wdmFixture()
	s.Devices[1].Driver = "asio" // Same name only as an ASIO entry: not WDM presence evidence.
	s.Devices = s.Devices[:2]
	if key, _ := callbackTarget(c, s); key != "" {
		t.Fatal("ASIO entry accepted as presence of a non-matching A1")
	}

	c, s = wdmFixture()
	delete(s.Assignments, "A1")
	if _, reason := callbackTarget(c, s); reason != "No device assigned to A1" {
		t.Fatal(reason)
	}

	// An installed Volt ASIO driver without physical presence never qualifies.
	c, s = callbackFixture()
	s.Devices[0].Available = true
	s.Devices = s.Devices[:1]
	if key, reason := callbackTarget(c, s); key != "" || !strings.Contains(reason, "not present") {
		t.Fatal(key, reason)
	}

	// A WDM target without a Studio configuration still qualifies.
	c, s = wdmFixture()
	c.Studio = nil
	h = callbackHealth{}
	if !qualify(&h, c, s, time.Now()) {
		t.Fatal("WDM A1 without studio config not detected")
	}
}
