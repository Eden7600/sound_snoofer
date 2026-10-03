package controller

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"voice-snooter/internal/config"
	"voice-snooter/internal/model"
)

type fakeClock struct {
	now    time.Time
	onWait func()
}

func (f *fakeClock) Now() time.Time { return f.now }
func (f *fakeClock) Wait(ctx context.Context, d time.Duration) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	f.now = f.now.Add(d)
	if f.onWait != nil {
		f.onWait()
	}
	return ctx.Err()
}

type write struct {
	target string
	device model.Device
}
type fakeBackend struct {
	s          model.Snapshot
	err        error
	writes     []write
	setErr     string
	pending    *write
	reads      int
	delay      int
	never      bool
	onSnapshot func(*fakeBackend)
}

func (b *fakeBackend) Snapshot() (model.Snapshot, error) {
	b.reads++
	if b.onSnapshot != nil {
		b.onSnapshot(b)
	}
	if b.err != nil {
		return model.Snapshot{}, b.err
	}
	if b.pending != nil && !b.never {
		if b.delay == 0 {
			b.s.Assignments[b.pending.target] = b.pending.device.Name
			b.pending = nil
		} else {
			b.delay--
		}
	}
	s := b.s
	s.Assignments = map[string]string{}
	for k, v := range b.s.Assignments {
		s.Assignments[k] = v
	}
	s.Devices = append([]model.Device{}, b.s.Devices...)
	return s, nil
}
func (b *fakeBackend) Set(target string, d model.Device) error {
	b.writes = append(b.writes, write{target, d})
	if b.setErr == target {
		return errors.New("set failed")
	}
	w := write{target, d}
	b.pending = &w
	return nil
}
func fixture(t *testing.T) (*Controller, *fakeBackend, *fakeClock) {
	t.Helper()
	cfg, e := config.Decode([]byte(`{"version":1,"verify_ms":300,"routes":[{"target":"input:1","candidates":[{"driver":"wdm","pattern":"Volt"},{"driver":"wdm","pattern":"webcam"}]},{"target":"A1","candidates":[{"driver":"wdm","pattern":"AirPods"},{"driver":"wdm","pattern":"speakers"}]}]}`))
	if e != nil {
		t.Fatal(e)
	}
	b := &fakeBackend{s: model.Snapshot{Edition: 2, Assignments: map[string]string{"input:1": "webcam", "A1": "speakers", "A2": "untouched"}, Devices: []model.Device{{Name: "Volt", Direction: "input", Driver: "wdm", Available: true}, {Name: "webcam", Direction: "input", Driver: "wdm", Available: true}, {Name: "AirPods", Direction: "output", Driver: "wdm", Available: true}, {Name: "speakers", Direction: "output", Driver: "wdm", Available: true}}}}
	clock := &fakeClock{now: time.Unix(0, 0)}
	return &Controller{Config: cfg, Backend: b, Clock: clock}, b, clock
}
func TestApplyVerifiedMinimalAndIdempotent(t *testing.T) {
	c, b, _ := fixture(t)
	b.delay = 2
	p, e := c.Plan()
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if len(b.writes) != 2 || b.writes[0].target != "input:1" || b.writes[1].target != "A1" || b.s.Assignments["A2"] != "untouched" {
		t.Fatal(b)
	}
	p, _ = c.Plan()
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if len(b.writes) != 2 {
		t.Fatal("repeated writes")
	}
}
func TestPartialFailureStopsPass(t *testing.T) {
	c, b, _ := fixture(t)
	b.setErr = "input:1"
	p, _ := c.Plan()
	e := c.Apply(context.Background(), p)
	if e == nil || len(b.writes) != 1 {
		t.Fatal(e, b.writes)
	}
	c, b, _ = fixture(t)
	b.setErr = "A1"
	p, _ = c.Plan()
	e = c.Apply(context.Background(), p)
	if e == nil || !strings.Contains(e.Error(), "1 assignments verified") || b.s.Assignments["A1"] != "speakers" {
		t.Fatal(e)
	}
}
func TestVerificationTimeoutAndDisconnect(t *testing.T) {
	c, b, clock := fixture(t)
	b.never = true
	p, _ := c.Plan()
	e := c.Apply(context.Background(), p)
	if e == nil || !strings.Contains(e.Error(), "timed out") || len(b.writes) != 1 {
		t.Fatal(e)
	}
	if clock.now.Sub(time.Unix(0, 0)) != 300*time.Millisecond {
		t.Fatal(clock.now)
	}
	c, b, _ = fixture(t)
	p, _ = c.Plan()
	b.onSnapshot = func(b *fakeBackend) {
		if len(b.writes) > 0 {
			b.err = errors.New("disconnected")
		}
	}
	if e = c.Apply(context.Background(), p); e == nil || len(b.writes) != 1 {
		t.Fatal(e)
	}
}
func TestPlanChangedBeforeWrite(t *testing.T) {
	c, b, _ := fixture(t)
	p, _ := c.Plan()
	b.s.Devices[0].Available = false
	if e := c.Apply(context.Background(), p); !errors.Is(e, ErrPlanChanged) {
		t.Fatal(e)
	}
	if len(b.writes) != 0 {
		t.Fatal(b.writes)
	}
}
func TestDryRunAndDebounce(t *testing.T) {
	c, b, clock := fixture(t)
	for i := 0; i < 5; i++ {
		c.Step(context.Background(), false)
		clock.now = clock.now.Add(time.Second)
	}
	if len(b.writes) != 0 {
		t.Fatal("dry-run wrote")
	}
	c, b, clock = fixture(t)
	c.Step(context.Background(), true)
	clock.now = clock.now.Add(time.Second)
	c.Step(context.Background(), true)
	if len(b.writes) != 0 {
		t.Fatal("early write")
	}
	clock.now = clock.now.Add(time.Second)
	c.Step(context.Background(), true)
	if len(b.writes) != 2 {
		t.Fatal(b.writes)
	}
}
func TestFlappingAndInvalidSnapshotsResetDebounce(t *testing.T) {
	c, b, clock := fixture(t)
	c.Step(context.Background(), true)
	clock.now = clock.now.Add(time.Second)
	b.s.Devices[0].Available = false
	c.Step(context.Background(), true)
	clock.now = clock.now.Add(time.Second)
	b.s.Devices[0].Available = true
	c.Step(context.Background(), true)
	if len(b.writes) != 0 {
		t.Fatal("flap caused write")
	}
	clock.now = clock.now.Add(time.Second)
	b.err = errors.New("partial inventory")
	c.Step(context.Background(), true)
	clock.now = clock.now.Add(3 * time.Second)
	b.err = nil
	c.Step(context.Background(), true)
	if len(b.writes) != 0 {
		t.Fatal("failure retained debounce")
	}
	clock.now = clock.now.Add(2 * time.Second)
	c.Step(context.Background(), true)
	if len(b.writes) != 2 {
		t.Fatal(b.writes)
	}
}
func TestReconnectEditionAndRetry(t *testing.T) {
	c, b, clock := fixture(t)
	b.err = errors.New("offline")
	for i, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second} {
		if got := c.Step(context.Background(), true); got != want {
			t.Fatalf("retry %d: %s", i, got)
		}
	}
	b.err = nil
	b.s.Edition = 3
	c.Step(context.Background(), true)
	if c.retry != 0 || len(b.writes) != 0 {
		t.Fatal("did not reset after recovery")
	}
	clock.now = clock.now.Add(2 * time.Second)
	c.Step(context.Background(), true)
	if len(b.writes) != 2 {
		t.Fatal(b.writes)
	}
	b.s.Edition = 1
	clock.now = clock.now.Add(3 * time.Second)
	c.Step(context.Background(), true)
	if len(b.writes) != 2 {
		t.Fatal("unsupported edition wrote")
	}
}
func TestCancellationAndNoCandidate(t *testing.T) {
	c, b, _ := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := c.Watch(ctx, true); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	p, _ := c.Plan()
	if e := c.Apply(ctx, p); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if len(b.writes) != 0 {
		t.Fatal(b.writes)
	}
	b.s.Devices = nil
	p, _ = c.Plan()
	if e := c.Apply(context.Background(), p); e == nil {
		t.Fatal("unresolved apply reported success")
	}
	if len(b.writes) != 0 {
		t.Fatal(b.writes)
	}
}
func TestStableWatchDoesNotFloodLogs(t *testing.T) {
	c, b, clock := fixture(t)
	b.s.Assignments["input:1"] = "Volt"
	b.s.Assignments["A1"] = "AirPods"
	count := 0
	c.Emit = func(Event) { count++ }
	for i := 0; i < 10; i++ {
		c.Step(context.Background(), true)
		clock.now = clock.now.Add(time.Second)
	}
	if count != 1 || len(b.writes) != 0 {
		t.Fatal(count, b.writes)
	}
}

func TestSetterFailureBackoffSurvivesDebounce(t *testing.T) {
	c, b, clock := fixture(t)
	b.setErr = "input:1"
	for _, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second} {
		c.Step(context.Background(), true)
		clock.now = clock.now.Add(2 * time.Second)
		if got := c.Step(context.Background(), true); got != want {
			t.Fatalf("got retry %s want %s", got, want)
		}
		clock.now = clock.now.Add(want)
	}
	b.setErr = ""
	c.Step(context.Background(), true)
	clock.now = clock.now.Add(2 * time.Second)
	c.Step(context.Background(), true)
	if c.retry != 0 {
		t.Fatal("successful apply did not reset backoff")
	}
}
