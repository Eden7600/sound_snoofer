package hue

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func TestProbeReportsIdentityAndFingerprint(t *testing.T) {
	bridge := newFakeBridge(t, "001788fffe000001")
	id, err := probe(context.Background(), bridge.address())
	if err != nil {
		t.Fatal(err)
	}
	if id.BridgeID != "001788fffe000001" || id.Fingerprint != bridge.fingerprint() {
		t.Fatalf("identity %+v, want fingerprint %s", id, bridge.fingerprint())
	}
}

func TestPinnedClientRejectsChangedCertificate(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	client := newClient(bridge.address(), strings.Repeat("0", 64), bridge.key)
	defer client.close()
	_, err := client.Resources(context.Background())
	if !errors.Is(err, ErrCertificateChanged) {
		t.Fatalf("got %v, want ErrCertificateChanged", err)
	}
	if bridge.streamCount() != 0 || len(bridge.putLog()) != 0 {
		t.Fatal("authenticated request reached a mismatched bridge")
	}
}

func TestRequestKeyWaitsForLinkButton(t *testing.T) {
	bridge := newFakeBridge(t, "b1")
	_, err := requestKey(context.Background(), bridge.address(), bridge.fingerprint(), "snoofer#test")
	if !errors.Is(err, ErrLinkButton) {
		t.Fatalf("got %v, want ErrLinkButton", err)
	}
	bridge.set(func(b *fakeBridge) { b.linkPressed = true })
	key, err := requestKey(context.Background(), bridge.address(), bridge.fingerprint(), "snoofer#test")
	if err != nil || key != bridge.key {
		t.Fatalf("key %q, err %v", key, err)
	}
}

func TestClientResourcesAndPut(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	client := newClient(bridge.address(), bridge.fingerprint(), bridge.key)
	defer client.close()
	items, err := client.Resources(context.Background())
	if err != nil || len(items) != len(studio()) {
		t.Fatalf("%d resources, err %v", len(items), err)
	}
	bridge.set(func(b *fakeBridge) { b.throttle = 1 })
	body := map[string]any{"on": map[string]bool{"on": false}}
	if err := client.Put(context.Background(), "grouped_light", "gl-1", body); !errors.Is(err, ErrThrottled) {
		t.Fatalf("got %v, want ErrThrottled", err)
	}
	if err := client.Put(context.Background(), "grouped_light", "gl-1", body); err != nil {
		t.Fatal(err)
	}
	wrongKey := newClient(bridge.address(), bridge.fingerprint(), "other")
	defer wrongKey.close()
	if _, err := wrongKey.Resources(context.Background()); err == nil {
		t.Fatal("rejected key loaded resources")
	}
}

func TestEventsDeliverUpdatesAndEndOnGap(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	client := newClient(bridge.address(), bridge.fingerprint(), bridge.key)
	defer client.close()
	received := make(chan []event, 4)
	done := make(chan error, 1)
	go func() {
		done <- client.Events(context.Background(), func(events []event) error {
			received <- events
			return nil
		})
	}()
	waitFor(t, func() bool { return bridge.streamCount() == 1 })
	bridge.update(resource{ID: "gl-1", Type: "grouped_light", On: &onState{On: false}})
	select {
	case events := <-received:
		if len(events) != 1 || events[0].Type != "update" || events[0].Data[0].On.On {
			t.Fatalf("events %+v", events)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
	bridge.closeStreams()
	select {
	case err := <-done:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("stream ended with %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not end")
	}
}

func TestReadEventsJoinsDataLines(t *testing.T) {
	input := ": hi\n\nid: 1\ndata: [{\"type\":\"update\",\ndata: \"data\":[{\"id\":\"x\",\"type\":\"light\"}]}]\n\n"
	var got []event
	err := readEvents(strings.NewReader(input), func(events []event) error {
		got = append(got, events...)
		return nil
	})
	if !errors.Is(err, io.EOF) || len(got) != 1 || got[0].Data[0].ID != "x" {
		t.Fatalf("events %+v, err %v", got, err)
	}
	if err := readEvents(strings.NewReader("data: {bad\n\n"), func([]event) error { return nil }); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("malformed event accepted: %v", err)
	}
}

func TestResolveOutcomes(t *testing.T) {
	ctx := context.Background()
	one := newFakeBridge(t, "b1")
	two := newFakeBridge(t, "b2")
	list := func(addresses ...string) discoverFunc {
		return func(context.Context) ([]string, error) { return addresses, nil }
	}
	if _, err := resolve(ctx, "", "", list()); !errors.Is(err, errNoBridge) {
		t.Fatalf("none: %v", err)
	}
	target, err := resolve(ctx, "", "", list(one.address(), "127.0.0.1:1"))
	if err != nil || target.BridgeID != "b1" || target.Address != one.address() {
		t.Fatalf("one: %+v %v", target, err)
	}
	var multiple *multipleBridgesError
	if _, err := resolve(ctx, "", "", list(one.address(), two.address())); !errors.As(err, &multiple) || len(multiple.Candidates) != 2 {
		t.Fatalf("many: %v", err)
	}
	target, err = resolve(ctx, "", "b2", list(one.address(), two.address()))
	if err != nil || target.Address != two.address() {
		t.Fatalf("paired among many: %+v %v", target, err)
	}
	if _, err := resolve(ctx, "", "b3", list(one.address())); !errors.Is(err, errNoBridge) {
		t.Fatalf("paired missing: %v", err)
	}
	var mismatch *bridgeMismatchError
	if _, err := resolve(ctx, one.address(), "b2", list()); !errors.As(err, &mismatch) {
		t.Fatalf("address mismatch: %v", err)
	}
}

func TestHueResponseRecognition(t *testing.T) {
	query, err := mdnsQuery()
	if err != nil {
		t.Fatal(err)
	}
	if isHueResponse(query) {
		t.Fatal("query treated as response")
	}
	response := func(name string) []byte {
		builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true, Authoritative: true})
		if err := builder.StartAnswers(); err != nil {
			t.Fatal(err)
		}
		header := dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(name), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET, TTL: 120}
		target := dnsmessage.MustNewName("Hue Bridge - 1A2B3C." + name)
		if err := builder.PTRResource(header, dnsmessage.PTRResource{PTR: target}); err != nil {
			t.Fatal(err)
		}
		packet, err := builder.Finish()
		if err != nil {
			t.Fatal(err)
		}
		return packet
	}
	if !isHueResponse(response(hueService)) {
		t.Fatal("Hue PTR response not recognized")
	}
	if isHueResponse(response("_googlecast._tcp.local.")) {
		t.Fatal("other service recognized")
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
