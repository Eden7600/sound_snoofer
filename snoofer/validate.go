package snoofer

import (
	"errors"
	"fmt"
)

// ValidateEnabled checks enabled settings and dependencies without starting plugins.
// Disabled and uncompiled disabled settings never invoke plugin validators.
func ValidateEnabled(c Config, plugins ...Plugin) error {
	byID := map[string]Plugin{}
	duplicate := map[string]bool{}
	for _, p := range plugins {
		if _, ok := byID[p.ID]; ok {
			duplicate[p.ID] = true
		}
		byID[p.ID] = p
	}
	visited := map[string]bool{}
	done := map[string]bool{}
	var check func(string) error
	check = func(id string) error {
		if done[id] {
			return nil
		}
		p, ok := byID[id]
		if !ok || duplicate[id] {
			return fmt.Errorf("unavailable plugin %s", id)
		}
		if !c.Plugins[id].Enabled {
			return fmt.Errorf("dependency %s is disabled", id)
		}
		if visited[id] {
			return fmt.Errorf("dependency cycle at %s", id)
		}
		visited[id] = true
		defer delete(visited, id)
		for _, dep := range p.Requires {
			if err := check(dep); err != nil {
				return err
			}
		}
		if p.Validate != nil {
			if err := p.Validate(c.Plugins[id].Settings); err != nil {
				return fmt.Errorf("%s: %w", id, err)
			}
		}
		done[id] = true
		return nil
	}
	var failures []error
	for id, p := range c.Plugins {
		if p.Enabled {
			if err := check(id); err != nil {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
}
