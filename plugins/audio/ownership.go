package audio

import (
	"encoding/json"
	"fmt"
	"os"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/storage"
)

type policyOwnership struct {
	Devices  config.VR          `json:"devices"`
	Playback []config.Candidate `json:"playback"`
}

func loadPolicyOwnership(path string) (*policyOwnership, error) {
	data, err := os.ReadFile(path + ".vr-ownership.json")
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var owned policyOwnership
	if err = storage.Decode(data, &owned); err != nil {
		return nil, fmt.Errorf("audio policy ownership: %w", err)
	}
	policy := config.ProfilePolicy{Devices: owned.Devices, Playback: owned.Playback, Microphones: []string{"normal"}, Choices: config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}}
	if err = policy.Validate(); err != nil {
		return nil, err
	}
	owned.Devices, owned.Playback = policy.Devices, policy.Playback
	return &owned, nil
}

// savePolicyOwnership records cleanup metadata before a policy can assign routes.
func (i *Instance) savePolicyOwnership(policy VRPolicy) error {
	if i.statePath == "" {
		return nil
	}
	owned := policyOwnership{Devices: policy.Devices, Playback: policy.Playback}
	data, err := json.Marshal(owned)
	if err != nil {
		return err
	}
	if string(data) == i.ownedPolicy {
		return nil
	}
	if i.ownedInput != 0 && i.ownedInput != policy.Devices.Input {
		return fmt.Errorf("VR input changed from %d to %d; clear the old owned input and monitoring sends before replacing %s.vr-ownership.json", i.ownedInput, policy.Devices.Input, i.statePath)
	}
	if err = storage.Replace(i.statePath+".vr-ownership.json", append(data, '\n')); err != nil {
		return err
	}
	i.ownedPolicy, i.ownedInput = string(data), policy.Devices.Input
	return nil
}
