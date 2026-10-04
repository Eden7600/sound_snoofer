package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func TestVoiceDebounceDriftAndInventory(t *testing.T) {
	c, b := voiceController(t)
	clock := c.Clock.(*fakeClock)
	c.Step(context.Background(), true)
	if b.attempt != 0 {
		t.Fatal("no debounce")
	}
	clock.now = clock.now.Add(3 * time.Second)
	c.Step(context.Background(), true)
	if b.s.Numbers["Strip[6].B3"] != 1 {
		t.Fatal("not applied")
	}
	b.s.Numbers["Strip[6].B2"] = 1
	c.Step(context.Background(), true)
	clock.now = clock.now.Add(3 * time.Second)
	c.Step(context.Background(), true)
	if b.s.Numbers["Strip[6].B2"] != 0 {
		t.Fatal("loop drift not repaired")
	}
	c, b = voiceController(t)
	b.after = func() { b.s.Devices[0].Available = false }
	p, _ := c.Plan()
	if e := c.Apply(context.Background(), p); e == nil {
		t.Fatal("inventory change ignored")
	}
	if b.attempt != 1 {
		t.Fatal("continued after inventory changed")
	}
}

type voiceBackend struct {
	*topologyBackend
	attempt int
	failAt  int
	neverAt int
	after   func()
	inspect func()
}

func (b *voiceBackend) SetNumber(p string, v int) error {
	b.attempt++
	if b.attempt == b.failAt {
		return fmt.Errorf("injected numeric failure")
	}
	if b.attempt == b.neverAt {
		return nil
	}
	e := b.topologyBackend.SetNumber(p, v)
	if b.inspect != nil {
		b.inspect()
	}
	if b.after != nil {
		b.after()
	}
	return e
}
func voiceController(t *testing.T) (*Controller, *voiceBackend) {
	c, tb := topologyFixture(t)
	tb.s.Edition = 3
	tb.s.Element = &model.ProcessStatus{Known: true, Running: true}
	for _, slot := range model.Slots(3) {
		tb.s.Assignments[slot] = ""
	}
	tb.s.Assignments["A1"] = "Volt ASIO"
	tb.s.Assignments["A2"] = "AirPods"
	tb.s.Assignments["input:3"] = "webcam"
	tb.s.Numbers = map[string]float32{}
	for i, v := range []float32{1, 1, 2, 2} {
		tb.s.Numbers[fmt.Sprintf("Patch.asio[%d]", i)] = v
	}
	for strip := 0; strip < 8; strip++ {
		for bus := 1; bus <= 5; bus++ {
			tb.s.Numbers[fmt.Sprintf("Strip[%d].A%d", strip, bus)] = 0
		}
		for bus := 1; bus <= 3; bus++ {
			tb.s.Numbers[fmt.Sprintf("Strip[%d].B%d", strip, bus)] = 0
		}
	}
	tb.s.Numbers["Strip[5].A2"] = 1
	tb.s.Numbers["Strip[0].B3"] = 1
	c.Config.Studio.Voice = &config.Voice{}
	if e := c.Config.Validate(); e != nil {
		t.Fatal(e)
	}
	b := &voiceBackend{topologyBackend: tb}
	c.Backend = b
	return c, b
}
func TestVoiceTransitionEveryIntermediateState(t *testing.T) {
	c, b := voiceController(t)
	b.inspect = func() {
		n := b.s.Numbers
		if n["Strip[6].B2"] != 0 {
			t.Fatal("feedback")
		}
		if n["Strip[6].B3"] != 0 && (n["Strip[0].B3"] != 0 || n["Strip[1].B3"] != 0 || n["Strip[2].B3"] != 0) {
			t.Fatal("doubled capture")
		}
	}
	for _, mode := range []string{"element", "direct", "element"} {
		c.Config.Intent = c.Config.VoiceIntent()
		c.Config.Intent.Mode = mode
		p, e := c.Plan()
		if e != nil {
			t.Fatal(e)
		}
		if e = c.Apply(context.Background(), p); e != nil {
			t.Fatal(e)
		}
		p, e = c.Plan()
		if e != nil || p.HasChanges() {
			t.Fatal("convergence", e)
		}
		before := b.attempt
		c.Apply(context.Background(), p)
		if before != b.attempt {
			t.Fatal("redundant writes")
		}
	}
	c.Config.Intent.Source = "lav"
	p, _ := c.Plan()
	if e := c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if b.s.Numbers["Strip[1].B2"] != 1 {
		t.Fatal("lav not selected")
	}
}
func TestVoicePhaseFailureAndRestart(t *testing.T) {
	for _, at := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			c, b := voiceController(t)
			b.failAt = at
			p, _ := c.Plan()
			e := c.Apply(context.Background(), p)
			if e == nil || !strings.Contains(e.Error(), "verified") {
				t.Fatal(e)
			}
			if b.attempt != at {
				t.Fatal("continued after failure")
			}
			if b.s.Numbers["Strip[6].B3"] != 0 {
				t.Fatal("enabled return after failure")
			}
			b.failAt = 0
			p, _ = c.Plan()
			if e = c.Apply(context.Background(), p); e != nil {
				t.Fatal("restart", e)
			}
		})
	}
	c, b := voiceController(t)
	b.neverAt = 1
	p, _ := c.Plan()
	if e := c.Apply(context.Background(), p); e == nil {
		t.Fatal("disable timeout ignored")
	}
	if b.attempt != 1 {
		t.Fatal("continued after timeout")
	}
	c, b = voiceController(t)
	b.s.Numbers["Patch.asio[0]"] = 0
	b.failAt = 2
	p, _ = c.Plan()
	if e := c.Apply(context.Background(), p); e == nil {
		t.Fatal("patch failure ignored")
	}
	if b.s.Numbers["Strip[6].B3"] != 0 {
		t.Fatal("enabled after patch failure")
	}
}
func TestVoiceConcurrentDriftAndCancellation(t *testing.T) {
	c, b := voiceController(t)
	b.after = func() { b.s.Numbers["Strip[6].B2"] = 1 }
	p, _ := c.Plan()
	if e := c.Apply(context.Background(), p); e == nil {
		t.Fatal("drift ignored")
	}
	if b.attempt != 1 {
		t.Fatal("enabled after drift")
	}
	c, b = voiceController(t)
	ctx, cancel := context.WithCancel(context.Background())
	b.after = cancel
	p, _ = c.Plan()
	if e := c.Apply(ctx, p); e == nil {
		t.Fatal("cancel ignored")
	}
	if b.attempt != 1 {
		t.Fatal("enabled after cancel")
	}
}
