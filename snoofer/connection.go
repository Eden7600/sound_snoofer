package snoofer

import "time"

// ConnectionState is the typed health of an external integration. Surfaces
// derive tone from it; Control.Value carries the short display word.
type ConnectionState string

// Connection states. Disconnected means an expected peer is absent; Off means
// the integration is disabled or unused and is not a fault.
const (
	ConnectionConnected    ConnectionState = "connected"
	ConnectionReady        ConnectionState = "ready" // Stateless integration that is usable.
	ConnectionConnecting   ConnectionState = "connecting"
	ConnectionAttention    ConnectionState = "attention" // Working but degraded.
	ConnectionDisconnected ConnectionState = "disconnected"
	ConnectionOff          ConnectionState = "off"
	ConnectionUnconfigured ConnectionState = "unconfigured"
	ConnectionError        ConnectionState = "error"
	ConnectionUnknown      ConnectionState = "unknown"
)

// Connection reports one external integration (an app, device, OS service or
// companion library) for diagnostics. Providers publish it on a status-only
// control of Kind "connection" and must never include credentials.
type Connection struct {
	State        ConnectionState
	Endpoint     string    // What is talked to: path, address, URL, device or process.
	Since        time.Time // When State was entered; zero when unknown.
	LastActivity time.Time // Last successful exchange; zero when none.
	LastError    string    // Most recent failure, kept after recovery.
	LastErrorAt  time.Time
	Details      []ConnectionDetail // Ordered, provider-chosen facts.
}

// ConnectionDetail is one labelled fact in a connection report.
type ConnectionDetail struct {
	Label, Value string
}

// clone returns an independent copy so providers and readers never share Details.
func (c *Connection) clone() *Connection {
	if c == nil {
		return nil
	}
	out := *c
	out.Details = append([]ConnectionDetail(nil), c.Details...)
	return &out
}
