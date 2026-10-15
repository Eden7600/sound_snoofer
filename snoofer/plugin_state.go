package snoofer

// PluginState is lifecycle state, separate from a plugin's descriptive status.
type PluginState string

const (
	PluginMissing  PluginState = "missing"
	PluginDisabled PluginState = "disabled"
	PluginStarting PluginState = "starting"
	PluginRunning  PluginState = "running"
	PluginFailed   PluginState = "failed"
)

// Lifecycle returns independent state for every compiled or configured plugin.
func (h *Host) Lifecycle() map[string]PluginState {
	h.mu.Lock()
	defer h.mu.Unlock()
	states := make(map[string]PluginState, len(h.config.Plugins)+len(h.plugins))
	for id := range h.config.Plugins {
		states[id] = PluginMissing
	}
	for id := range h.plugins {
		state := PluginStarting
		switch {
		case !h.config.Plugins[id].Enabled:
			state = PluginDisabled
		case h.instances[id] != nil:
			state = PluginRunning
		case h.failed[id] != nil:
			state = PluginFailed
		}
		states[id] = state
	}
	return states
}
