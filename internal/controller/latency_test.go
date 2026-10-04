package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"sound-snoofer/internal/model"
)

func TestDeviceDefaultDebounceBoundary(t *testing.T) {
	c, b := voiceController(t)
	if c.Config.Debounce() != time.Second {
		t.Fatal("default", c.Config.Debounce())
	}
	c.Config.Intent = c.Config.VoiceIntent()
	c.Config.Intent.PlaybackDevice = "speakers"
	clock := c.Clock.(*fakeClock)
	ctx := context.Background()
	c.Step(ctx, true)
	clock.now = clock.now.Add(999 * time.Millisecond)
	c.Step(ctx, true)
	if len(b.writes) != 0 {
		t.Fatal("device applied before deadline")
	}
	clock.now = clock.now.Add(time.Millisecond)
	c.Step(ctx, true)
	if b.s.Assignments["A2"] != "speakers" {
		t.Fatal("device not applied at deadline")
	}
}
func TestDeviceSwitchUsesParameterPolling(t *testing.T) {
	c, b := recorderController(t)
	counter := &parameterCounter{recorderFake: b}
	c.Backend = counter
	c.Config.Intent.PlaybackDevice = "speakers"
	p, err := c.Plan()
	if err != nil {
		t.Fatal(err)
	}
	b.delay = 3
	counter.full = 0
	clock := c.Clock.(*fakeClock)
	before := clock.now
	if err = c.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if counter.full != 4 {
		t.Fatalf("full reads=%d want 4", counter.full)
	}
	if counter.fast < 5 {
		t.Fatal("parameter reads missing", counter.fast)
	}
	if clock.now.Sub(before) != 60*time.Millisecond {
		t.Fatal("slow verification", clock.now.Sub(before))
	}
	if b.s.Assignments["A2"] != "speakers" || b.s.Numbers["Strip[5].A2"] != 1 {
		t.Fatal("playback did not recover")
	}
}

type cachedParameterCounter struct {
	*parameterCounter
	inventory []model.Device
}

func (b *cachedParameterCounter) ParameterSnapshot() (model.Snapshot, error) {
	b.fast++
	s, err := b.recorderFake.Snapshot()
	s.Devices = append([]model.Device{}, b.inventory...)
	return s, err
}
func TestDeviceConfirmationRejectsInventoryChange(t *testing.T) {
	c, b := recorderController(t)
	counter := &cachedParameterCounter{parameterCounter: &parameterCounter{recorderFake: b}, inventory: append([]model.Device{}, b.s.Devices...)}
	c.Backend = counter
	c.Config.Intent.PlaybackDevice = "speakers"
	p, err := c.Plan()
	if err != nil {
		t.Fatal(err)
	}
	b.onSnapshot = func(_ *fakeBackend) {
		if b.pending != nil {
			b.s.Devices[0].ID = "changed"
		}
	}
	err = c.Apply(context.Background(), p)
	if !errors.Is(err, ErrPlanChanged) {
		t.Fatal("inventory drift ignored", err)
	}
	if b.s.Numbers["Strip[5].A2"] != 0 {
		t.Fatal("restored sends before confirmation")
	}
}
