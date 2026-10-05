package audio

import (
	"fmt"
	"slices"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
)

// VRPolicy is the narrow data contract consumed from the VR dependency.
// The supplying plugin owns process observation; audio owns resolution and writes.
type VRPolicy = config.ProfilePolicy

// SetVRPolicy publishes validated immutable policy. Unknown observations retain
// the last effective activation; they never invent a transition.
func (i *Instance) SetVRPolicy(policy VRPolicy) error {
	policy.Microphones = slices.Clone(policy.Microphones)
	policy.Playback = slices.Clone(policy.Playback)
	policy.Devices.Headsets = slices.Clone(policy.Devices.Headsets)
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("VR policy: %w", err)
	}
	i.mu.Lock()
	if err := i.savePolicyOwnership(policy); err != nil {
		i.mu.Unlock()
		return err
	}
	if !policy.Known {
		if i.policy != nil {
			policy.Running = i.policy.Running
		} else {
			policy.Running = false
		}
	}
	i.policy = &policy
	i.mu.Unlock()
	select {
	case i.actions <- control.Refresh:
	default:
	}
	return nil
}
func (i *Instance) policySnapshot() *config.ProfilePolicy {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.policy
}
