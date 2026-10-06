package hue

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const hueService = "_hue._tcp.local."

var errNoBridge = errors.New("no bridge found")

// multipleBridgesError reports an ambiguous discovery result; no bridge is chosen.
type multipleBridgesError struct {
	Candidates []string // "address (bridge ID)" entries
}

func (e *multipleBridgesError) Error() string {
	return "multiple bridges: " + strings.Join(e.Candidates, ", ") + "; set address to choose"
}

// bridgeMismatchError reports an addressed bridge whose identity differs from the paired bridge.
type bridgeMismatchError struct {
	Address, Want, Got string
}

func (e *bridgeMismatchError) Error() string {
	return fmt.Sprintf("bridge at %s is %s, not paired bridge %s", e.Address, e.Got, e.Want)
}

// discoverFunc returns candidate bridge addresses on the local network.
type discoverFunc func(context.Context) ([]string, error)

// discoverMDNS sends one mDNS PTR query for Hue bridges and collects the
// source addresses of matching responses during window.
func discoverMDNS(ctx context.Context, window time.Duration) ([]string, error) {
	query, err := mdnsQuery()
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(window)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
	defer stop()
	group := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	if _, err := conn.WriteToUDP(query, group); err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	found := map[string]bool{}
	buffer := make([]byte, 9000)
	for {
		n, source, err := conn.ReadFromUDP(buffer)
		if err != nil {
			var timeout net.Error
			if errors.As(err, &timeout) && timeout.Timeout() {
				break
			}
			return nil, fmt.Errorf("discovery: %w", err)
		}
		if isHueResponse(buffer[:n]) {
			found[source.IP.String()] = true
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	addresses := make([]string, 0, len(found))
	for address := range found {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses, nil
}

func mdnsQuery() ([]byte, error) {
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	if err := builder.StartQuestions(); err != nil {
		return nil, err
	}
	question := dnsmessage.Question{Name: dnsmessage.MustNewName(hueService), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET}
	if err := builder.Question(question); err != nil {
		return nil, err
	}
	return builder.Finish()
}

// isHueResponse reports whether a DNS message answers with a Hue PTR record.
func isHueResponse(packet []byte) bool {
	var parser dnsmessage.Parser
	header, err := parser.Start(packet)
	if err != nil || !header.Response {
		return false
	}
	if err := parser.SkipAllQuestions(); err != nil {
		return false
	}
	for {
		answer, err := parser.AnswerHeader()
		if err != nil {
			return false
		}
		if answer.Type == dnsmessage.TypePTR && strings.EqualFold(answer.Name.String(), hueService) {
			return true
		}
		if err := parser.SkipAnswer(); err != nil {
			return false
		}
	}
}

// bridgeTarget is a resolved bridge address and its observed identity.
type bridgeTarget struct {
	Address string
	identity
}

// resolve finds the bridge to use. A configured address is authoritative; a
// non-empty bridgeID restricts discovery and addressing to that bridge.
func resolve(ctx context.Context, address, bridgeID string, discover discoverFunc) (bridgeTarget, error) {
	if address != "" {
		id, err := probe(ctx, address)
		if err != nil {
			return bridgeTarget{}, err
		}
		if bridgeID != "" && id.BridgeID != bridgeID {
			return bridgeTarget{}, &bridgeMismatchError{Address: address, Want: bridgeID, Got: id.BridgeID}
		}
		return bridgeTarget{Address: address, identity: id}, nil
	}
	addresses, err := discover(ctx)
	if err != nil {
		return bridgeTarget{}, err
	}
	var candidates []bridgeTarget
	for _, candidate := range addresses {
		id, err := probe(ctx, candidate)
		if err != nil {
			continue
		}
		if bridgeID != "" && id.BridgeID == bridgeID {
			return bridgeTarget{Address: candidate, identity: id}, nil
		}
		candidates = append(candidates, bridgeTarget{Address: candidate, identity: id})
	}
	if bridgeID != "" || len(candidates) == 0 {
		return bridgeTarget{}, errNoBridge
	}
	if len(candidates) > 1 {
		described := make([]string, 0, len(candidates))
		for _, c := range candidates {
			described = append(described, c.Address+" ("+c.BridgeID+")")
		}
		return bridgeTarget{}, &multipleBridgesError{Candidates: described}
	}
	return candidates[0], nil
}
