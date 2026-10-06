package hue

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/ipv4"
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

// queryInterface is an interface and the IPv4 address its query socket binds.
type queryInterface struct {
	Interface net.Interface
	Address   net.IP
}

// queryInterfaces selects up, multicast-capable, non-loopback interfaces with
// a routable IPv4 address. The OS default multicast route alone can miss the
// bridge network on hosts with VPN, virtual and wireless adapters.
func queryInterfaces(interfaces []net.Interface, addresses func(net.Interface) ([]net.Addr, error)) []queryInterface {
	var out []queryInterface
	for _, candidate := range interfaces {
		if candidate.Flags&net.FlagUp == 0 || candidate.Flags&net.FlagMulticast == 0 || candidate.Flags&net.FlagLoopback != 0 {
			continue
		}
		list, err := addresses(candidate)
		if err != nil {
			continue
		}
		for _, address := range list {
			network, ok := address.(*net.IPNet)
			if !ok {
				continue
			}
			ip := network.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			out = append(out, queryInterface{Interface: candidate, Address: ip})
			break
		}
	}
	return out
}

// discoverMDNS sends an mDNS PTR query for Hue bridges on every eligible
// interface and collects the source addresses of matching replies during window.
func discoverMDNS(ctx context.Context, window time.Duration) ([]string, error) {
	query, err := mdnsQuery()
	if err != nil {
		return nil, err
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	targets := queryInterfaces(interfaces, func(i net.Interface) ([]net.Addr, error) { return i.Addrs() })
	deadline := time.Now().Add(window)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	var (
		mu       sync.Mutex
		found    = map[string]bool{}
		failures []error
		readers  sync.WaitGroup
	)
	for _, target := range targets {
		conn, err := sendQuery(target, query, deadline)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
		readers.Add(1)
		go func() {
			defer readers.Done()
			defer stop()
			defer conn.Close()
			buffer := make([]byte, 9000)
			for {
				n, source, err := conn.ReadFromUDP(buffer)
				if err != nil {
					return // Deadline, cancellation or socket error ends this interface.
				}
				if isHueResponse(buffer[:n]) {
					mu.Lock()
					found[source.IP.String()] = true
					mu.Unlock()
				}
			}
		}()
	}
	readers.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(targets) == 0 || len(failures) == len(targets) {
		return nil, fmt.Errorf("discovery: no usable network interface: %w", errors.Join(failures...))
	}
	addresses := make([]string, 0, len(found))
	for address := range found {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses, nil
}

// sendQuery binds to one interface address and multicasts the query from it.
func sendQuery(target queryInterface, query []byte, deadline time.Time) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: target.Address})
	if err != nil {
		return nil, fmt.Errorf("discovery on %s: %w", target.Interface.Name, err)
	}
	packets := ipv4.NewPacketConn(conn)
	if err := packets.SetMulticastInterface(&target.Interface); err != nil {
		conn.Close()
		return nil, fmt.Errorf("discovery on %s: %w", target.Interface.Name, err)
	}
	if err := packets.SetMulticastTTL(255); err != nil {
		conn.Close()
		return nil, fmt.Errorf("discovery on %s: %w", target.Interface.Name, err)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, fmt.Errorf("discovery on %s: %w", target.Interface.Name, err)
	}
	group := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	if _, err := conn.WriteToUDP(query, group); err != nil {
		conn.Close()
		return nil, fmt.Errorf("discovery on %s: %w", target.Interface.Name, err)
	}
	return conn, nil
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
