package audio

import (
	"path/filepath"
	"testing"

	"sound-snoofer/internal/config"
)

func TestPolicyOwnershipSurvivesDisabledVR(t *testing.T) {
	i := &Instance{statePath: filepath.Join(t.TempDir(), "audio")}
	p := VRPolicy{Devices: config.VR{Input: 4}, Microphones: []string{"normal"}, Playback: []config.Candidate{{Driver: "wdm", Pattern: "^VR output$"}}, Choices: config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := i.savePolicyOwnership(p); err != nil {
		t.Fatal(err)
	}
	owned, err := loadPolicyOwnership(i.statePath)
	if err != nil {
		t.Fatal(err)
	}
	if owned.Devices.Input != 4 || len(owned.Playback) != 1 || !owned.Playback[0].Regex.MatchString("VR output") {
		t.Fatal(owned)
	}
	p.Devices.Input = 5
	if i.savePolicyOwnership(p) == nil {
		t.Fatal("forgot previously owned input")
	}
}
