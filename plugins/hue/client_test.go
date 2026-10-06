package hue

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
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

func TestQueryInterfacesSelectsEveryMulticastIPv4Adapter(t *testing.T) {
	up := net.FlagUp | net.FlagMulticast
	interfaces := []net.Interface{
		{Index: 1, Name: "Loopback", Flags: up | net.FlagLoopback},
		{Index: 4, Name: "Tailscale", Flags: up},
		{Index: 8, Name: "Ethernet", Flags: up},
		{Index: 9, Name: "Down", Flags: net.FlagMulticast},
		{Index: 10, Name: "PointToPoint", Flags: net.FlagUp},
		{Index: 61, Name: "vEthernet", Flags: up},
		{Index: 70, Name: "LinkLocalOnly", Flags: up},
	}
	addresses := map[string][]net.Addr{
		"Loopback":      {&net.IPNet{IP: net.IPv4(127, 0, 0, 1)}},
		"Tailscale":     {&net.IPNet{IP: net.ParseIP("fd7a::1")}, &net.IPNet{IP: net.IPv4(100, 96, 183, 127)}},
		"Ethernet":      {&net.IPNet{IP: net.IPv4(172, 16, 108, 22)}},
		"Down":          {&net.IPNet{IP: net.IPv4(10, 0, 0, 2)}},
		"PointToPoint":  {&net.IPNet{IP: net.IPv4(10, 0, 0, 3)}},
		"vEthernet":     {&net.IPNet{IP: net.IPv4(172, 21, 128, 1)}},
		"LinkLocalOnly": {&net.IPNet{IP: net.IPv4(169, 254, 1, 1)}},
	}
	got := queryInterfaces(interfaces, func(i net.Interface) ([]net.Addr, error) { return addresses[i.Name], nil })
	var names []string
	for _, q := range got {
		names = append(names, q.Interface.Name+"="+q.Address.String())
	}
	want := "Tailscale=100.96.183.127 Ethernet=172.16.108.22 vEthernet=172.21.128.1"
	if strings.Join(names, " ") != want {
		t.Fatalf("got %v, want %s", names, want)
	}
}

// TestLiveDiscovery checks real-network discovery when SNOOFER_HUE_LIVE is set.
func TestLiveDiscovery(t *testing.T) {
	if os.Getenv("SNOOFER_HUE_LIVE") == "" {
		t.Skip("set SNOOFER_HUE_LIVE=1 to query the local network")
	}
	addresses, err := discoverMDNS(context.Background(), 3*time.Second)
	if err != nil || len(addresses) == 0 {
		t.Fatalf("addresses %v, err %v", addresses, err)
	}
	t.Log("bridges:", addresses)
}
