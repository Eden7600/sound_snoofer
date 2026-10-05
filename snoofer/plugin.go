// Package snoofer hosts explicitly compiled, trusted plugins.
package snoofer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Instance owns all resources acquired during Start. Stop must honor its deadline.
type Instance interface{ Stop(context.Context) error }

// Plugin contains inert metadata. Start must clean up partial initialization on error.
type Plugin struct {
	Command   func(context.Context, Services, json.RawMessage, []string) error
	Validate  func(json.RawMessage) error
	Defaults  json.RawMessage
	ID, Label string
	Requires  []string
	Start     func(context.Context, Services, json.RawMessage, map[string]Instance) (Instance, error)
}

// Services contains core facilities. Only declared plugin dependencies arrive separately.
type Services struct {
	SaveSettings func(string, json.RawMessage, json.RawMessage) error
	Controls     *Controls
	Path         string
	Live         bool
}

// Host serializes lifecycle operations. Plugins must not reenter lifecycle methods.
type Host struct {
	failed     map[string]error
	mu         sync.Mutex
	plugins    map[string]Plugin
	duplicates map[string]bool
	instances  map[string]Instance
	status     map[string]string
	order      []string
	config     Config
	services   Services
	ctx        context.Context
	cancel     context.CancelFunc
	stopping   bool
}

// New constructs a host without starting plugins.
func New(config Config, services Services, plugins ...Plugin) *Host {
	if services.Controls == nil {
		services.Controls = NewControls()
	}
	h := &Host{config: config, services: services, plugins: map[string]Plugin{}, duplicates: map[string]bool{}, instances: map[string]Instance{}, status: map[string]string{}}
	for _, p := range plugins {
		if _, exists := h.plugins[p.ID]; exists {
			h.duplicates[p.ID] = true
		}
		h.plugins[p.ID] = p
	}
	h.services.SaveSettings = h.saveSettings
	return h
}

// Start starts independent valid branches, retaining failures in Status.
func (h *Host) Start(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ctx != nil {
		return
	}
	h.ctx, h.cancel = context.WithCancel(ctx)
	h.startAll()
}

func (h *Host) startAll() {
	h.failed = map[string]error{}
	ids := make([]string, 0, len(h.config.Plugins))
	for id := range h.config.Plugins {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if h.config.Plugins[id].Enabled {
			_ = h.start(id, map[string]bool{})
		} else {
			h.status[id] = "Disabled"
		}
	}
}

func (h *Host) start(id string, visiting map[string]bool) error {
	if h.instances[id] != nil {
		return nil
	}
	if err := h.failed[id]; err != nil {
		return err
	}
	fail := func(err error) error { h.failed[id] = err; h.status[id] = err.Error(); return err }
	p, ok := h.plugins[id]
	if !ok {
		return fail(fmt.Errorf("%s is not compiled into this build", id))
	}
	if h.duplicates[id] || !validID(id) || p.Start == nil {
		return fail(fmt.Errorf("invalid or duplicate plugin %q", id))
	}
	if !h.config.Plugins[id].Enabled {
		return fail(fmt.Errorf("%s is disabled", id))
	}
	if visiting[id] {
		return fail(fmt.Errorf("dependency cycle at %s", id))
	}
	visiting[id] = true
	defer delete(visiting, id)
	deps := map[string]Instance{}
	for _, dep := range p.Requires {
		if err := h.start(dep, visiting); err != nil {
			return fail(fmt.Errorf("%s requires %s: %w", id, dep, err))
		}
		deps[dep] = h.instances[dep]
	}
	if p.Validate != nil {
		if err := p.Validate(h.config.Plugins[id].Settings); err != nil {
			return fail(err)
		}
	}
	instance, err := p.Start(h.ctx, h.services, h.config.Plugins[id].Settings, deps)
	if err != nil {
		h.services.Controls.Remove(id)
		return fail(err)
	}
	if instance == nil {
		h.services.Controls.Remove(id)
		return fail(fmt.Errorf("%s returned no instance", id))
	}
	h.instances[id] = instance
	h.order = append(h.order, id)
	h.status[id] = "Running"
	return nil
}

// Retry starts failed branches without duplicating healthy instances.
func (h *Host) Retry() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ctx != nil && !h.stopping {
		h.startAll()
	}
}

// Status returns an independent presentation snapshot.
func (h *Host) Status() map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := map[string]string{}
	for id, s := range h.status {
		result[id] = s
	}
	return result
}

// Stop quiesces work before reverse dependency cleanup. Errors block relaunch.
func (h *Host) Stop(ctx context.Context) error {
	h.mu.Lock()
	if h.stopping {
		h.mu.Unlock()
		return errors.New("host already stopping")
	}
	h.stopping = true
	order := append([]string(nil), h.order...)
	h.mu.Unlock()
	h.services.Controls.Close()
	if h.cancel != nil {
		h.cancel()
	}
	var failures []error
	for n := len(order) - 1; n >= 0; n-- {
		id := order[n]
		done := make(chan error, 1)
		go func(instance Instance) { done <- instance.Stop(ctx) }(h.instances[id])
		select {
		case err := <-done:
			if err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", id, err))
			}
		case <-ctx.Done():
			return errors.Join(append(failures, ctx.Err())...)
		}
	}
	return errors.Join(failures...)
}

// Selection computes enable/disable closure without running plugin code.
func (h *Host) Selection(id string, enabled bool) (Config, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	next := h.config.Clone()
	if _, ok := h.plugins[id]; !ok {
		return next, fmt.Errorf("plugin %s is not compiled", id)
	}
	seen := map[string]bool{}
	var enable func(string) error
	enable = func(key string) error {
		if seen[key] {
			return fmt.Errorf("dependency cycle at %s", key)
		}
		p, ok := h.plugins[key]
		if !ok || h.duplicates[key] {
			return fmt.Errorf("unavailable dependency %s", key)
		}
		seen[key] = true
		defer delete(seen, key)
		for _, dep := range p.Requires {
			if err := enable(dep); err != nil {
				return err
			}
		}
		value := next.Plugins[key]
		if len(value.Settings) == 0 {
			value.Settings = h.initialSettings(key)
		}
		value.Enabled = true
		next.Plugins[key] = value
		return nil
	}
	if enabled {
		err := enable(id)
		return next, err
	}
	value := next.Plugins[id]
	value.Enabled = false
	next.Plugins[id] = value
	changed := true
	for changed {
		changed = false
		for key, p := range h.plugins {
			if !next.Plugins[key].Enabled {
				continue
			}
			for _, dep := range p.Requires {
				if !next.Plugins[dep].Enabled {
					value := next.Plugins[key]
					if len(value.Settings) == 0 {
						value.Settings = h.initialSettings(key)
					}
					value.Enabled = false
					next.Plugins[key] = value
					changed = true
					break
				}
			}
		}
	}
	return next, nil
}
