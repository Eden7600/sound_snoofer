package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"sound-snoofer/internal/browserbridge"
	"sound-snoofer/snoofer"
)

type bridgeCleanupFailure struct {
	err     error
	bounded bool
}

func (i *bridgeCleanupFailure) Stop(ctx context.Context) error {
	deadline, ok := ctx.Deadline()
	i.bounded = ok && time.Until(deadline) > 0 && time.Until(deadline) <= 5*time.Second
	return i.err
}

func TestStopHostClosesSharedBridgeEvenWhenPluginCleanupFails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	b := browserbridge.New(context.Background(), json.RawMessage(fmt.Sprintf(`{"port":%d}`, port)))
	t.Cleanup(func() { b.Close(context.Background()) })
	lease, err := b.AcquireMeet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("plugin cleanup failed")
	instance := &bridgeCleanupFailure{err: failure}
	host := snoofer.New(snoofer.Config{Plugins: map[string]snoofer.PluginConfig{"test": {Enabled: true}}}, snoofer.Services{Browser: b}, snoofer.Plugin{
		ID: "test", Start: func(context.Context, snoofer.Services, json.RawMessage, map[string]snoofer.Instance) (snoofer.Instance, error) {
			return instance, nil
		},
	})
	host.Start(context.Background())
	if err := stopHost(host, b); !errors.Is(err, failure) {
		t.Fatal("cleanup failure hidden", err)
	}
	if !instance.bounded {
		t.Fatal("host cleanup lacks independent bounded deadline")
	}
	if _, ok := <-lease.Updates(); ok {
		t.Fatal("shared lease not closed after host error")
	}
	if _, err := b.AcquireMedia(context.Background()); err == nil {
		t.Fatal("owner accepts work after cleanup")
	}
	again, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal("bridge port still owned", err)
	}
	again.Close()
}
