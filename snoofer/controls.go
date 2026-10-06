package snoofer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// Control is an immutable snapshot. Operations are press, adjust or set.
type Control struct {
	ViewData                                    json.RawMessage `json:",omitempty"` // Optional provider-owned structured view; core treats it as opaque.
	Artwork                                     string          // Optional base64 PNG thumbnail, square and at most 64px; immutable across snapshots.
	Animation                                   []ArtworkFrame  `json:"-"` // Optional frames that animate Artwork; immutable, and kept out of polled GUI state.
	ShortLabel                                  string          // Optional label for icon-bearing compact surfaces.
	Collection                                  string          // Optional stable ID of a set of similar controls, which deck regions fill.
	CollectionLabel                             string          // Optional editor name for Collection.
	Meter                                       Meter
	Connection                                  *Connection `json:",omitempty"` // Optional external integration report (Kind "connection").
	SurfaceOnly                                 bool
	Hidden                                      bool // Optional: no useful place on surfaces right now; the deck renders a blank key and GUIs omit it. Publish it unavailable.
	OptionLabels                                map[string]string
	EnterOnly                                   bool
	Epoch                                       uint64
	Subdued                                     bool
	ID, Label, Group, Kind, Value, Status, Icon string
	Options                                     []string
	Operations                                  []string
	Available                                   bool
	Revision                                    uint64
}

// ArtworkFrame is one animation frame in the Artwork thumbnail contract,
// shown for Delay.
type ArtworkFrame struct {
	Artwork string
	Delay   time.Duration
}

// Meter is optional display-only telemetry in dBFS; it never changes command identity.
type Meter struct {
	Present, Known bool
	DB             float64
	At             time.Time
}

// Request identifies the published control against which input was generated.
type Request struct {
	ID               string
	Revision         uint64
	Operation, Value string
	Delta            int
}

type entry struct {
	control Control
	invoke  func(context.Context, Request) error
}

// Controls serializes dispatch with registration/removal. Handlers must only
// enqueue bounded work, never perform device I/O or reenter this registry.
type Controls struct {
	mu       sync.Mutex
	entries  map[string]entry
	revision uint64
	closed   bool
}

// NewControls creates an empty registry.
func NewControls() *Controls { return &Controls{entries: map[string]entry{}} }

// Publish replaces a provider snapshot. IDs must belong to that provider.
func (c *Controls) Publish(provider string, controls []Control, invoke func(context.Context, Request) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("host is stopping")
	}
	if !validID(provider) {
		return fmt.Errorf("invalid provider ID %q", provider)
	}
	seen := map[string]bool{}
	for _, v := range controls {
		if !strings.HasPrefix(v.ID, provider+".") || seen[v.ID] {
			return fmt.Errorf("invalid or duplicate control %s", v.ID)
		}
		seen[v.ID] = true
	}
	for id := range c.entries {
		if strings.HasPrefix(id, provider+".") && !seen[id] {
			delete(c.entries, id)
		}
	}
	for _, v := range controls {
		old, ok := c.entries[v.ID]
		previous := old.control
		previous.Revision = 0
		previous.Meter = v.Meter // Signal motion must not invalidate pending control input.
		v.Revision = 0
		if ok && reflect.DeepEqual(previous, v) {
			v.Revision = old.control.Revision
		} else {
			c.revision++
			v.Revision = c.revision
		}
		v.ViewData = append(json.RawMessage(nil), v.ViewData...)
		v.Options = append([]string(nil), v.Options...)
		v.OptionLabels = maps.Clone(v.OptionLabels)
		v.Operations = append([]string(nil), v.Operations...)
		v.Connection = v.Connection.clone()
		c.entries[v.ID] = entry{v, invoke}
	}
	return nil
}

// Snapshot returns independent values sorted by group and ID.
func (c *Controls) Snapshot() []Control {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Control, 0, len(c.entries))
	for _, e := range c.entries {
		v := e.control
		v.ViewData = append(json.RawMessage(nil), v.ViewData...)
		v.Options = append([]string(nil), v.Options...)
		v.OptionLabels = maps.Clone(v.OptionLabels)
		v.Operations = append([]string(nil), v.Operations...)
		v.Connection = v.Connection.clone()
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Dispatch rejects stale, unavailable and unsupported input.
func (c *Controls) Dispatch(ctx context.Context, r Request) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("host is stopping")
	}
	e, ok := c.entries[r.ID]
	if !ok || !e.control.Available {
		return errors.New("control unavailable")
	}
	if r.Revision != e.control.Revision {
		return errors.New("control changed; try again")
	}
	supported := false
	for _, op := range e.control.Operations {
		supported = supported || op == r.Operation
	}
	if !supported || e.invoke == nil {
		return errors.New("unsupported operation")
	}
	if r.Operation == "set" && len(e.control.Options) > 0 {
		valid := false
		for _, v := range e.control.Options {
			valid = valid || v == r.Value
		}
		if !valid {
			return errors.New("invalid selection")
		}
	}
	return e.invoke(ctx, r)
}

// Remove invalidates a provider's controls.
func (c *Controls) Remove(provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.entries {
		if strings.HasPrefix(id, provider+".") {
			delete(c.entries, id)
		}
	}
}

// Close prevents further input during shutdown.
func (c *Controls) Close() { c.mu.Lock(); defer c.mu.Unlock(); c.closed = true }
