package control

import (
	"context"
	"path/filepath"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sync/atomic"
	"testing"
	"time"
)

type meteredClient struct {
	*ruleClient
	reads   atomic.Int32
	samples atomic.Int32
}

func (b *meteredClient) Snapshot() (model.Snapshot, error) {
	b.reads.Add(1)
	return b.ruleClient.Snapshot()
}
func (b *meteredClient) GainLevels(int) map[string]float32 {
	b.samples.Add(1)
	return map[string]float32{"Bus[0].Gain": 0.5}
}
func TestMetersPreserveRoutingCadenceAndStop(t *testing.T) {
	cfg, err := config.Decode([]byte(ruleConfig))
	if err != nil {
		t.Fatal(err)
	}
	cfg.PollMS = 1000
	b := &meteredClient{ruleClient: &ruleClient{&fakeClient{}}}
	ctx, cancel := context.WithCancel(context.Background())
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{Meters: true, Open: func(string) (Client, error) { return b, nil }}
	go Work(ctx, cfg, filepath.Join(t.TempDir(), "config.json"), "", false, deps, make(chan Action), states, done)
	defer func() { cancel(); <-done }()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for b.samples.Load() < 5 {
		select {
		case <-states:
		case <-timeout.C:
			t.Fatal("meter polling stopped")
		}
	}
	if b.reads.Load() > 3 {
		t.Fatal("meter polling accelerated routing", b.reads.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	samples := b.samples.Load()
	time.Sleep(120 * time.Millisecond)
	if b.samples.Load() != samples {
		t.Fatal("meter reads after shutdown")
	}
}
