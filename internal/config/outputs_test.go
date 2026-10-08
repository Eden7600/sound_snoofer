package config

import (
	"strings"
	"testing"
)

const outputsJSON = `,"outputs":[{"id":"music","name":"Music","device":"Speakers (Arena)","sources":["virtual:1","tape"]}]`

func outputConfig(t *testing.T, outputs string) (Config, error) {
	t.Helper()
	b := strings.Replace(string(DefaultBytes()), `"move_playback_routing": true`, `"move_playback_routing": true`+outputs, 1)
	return Decode([]byte(b))
}

func TestOutputsValidation(t *testing.T) {
	c, err := outputConfig(t, outputsJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Studio.OutputDevice("Speakers (Arena)") || c.Studio.OutputDevice("") || c.Studio.OutputDevice("Speakers") {
		t.Fatal("output device lookup")
	}
	if got := strings.Join(c.Studio.OutputSources(), ","); got != "virtual:1,monitor,soundboard,tape" {
		t.Fatal(got)
	}
	for _, bad := range []string{
		`,"outputs":[{"id":"Music","name":"Music","device":"x"}]`,
		`,"outputs":[{"id":"a","name":"","device":"x"}]`,
		`,"outputs":[{"id":"a","name":"Seventeen chars!!","device":"x"}]`,
		`,"outputs":[{"id":"a","name":"A","device":"x"},{"id":"a","name":"B","device":"y"}]`,
		`,"outputs":[{"id":"a","name":"A","device":"x"},{"id":"b","name":"B","device":"x"}]`,
		`,"outputs":[{"id":"a","name":"A","device":"x","sources":["virtual:2"]}]`,
		`,"outputs":[{"id":"a","name":"A","device":"x","sources":["monitor","monitor"]}]`,
		`,"outputs":[{"id":"a","name":"A","device":"w"},{"id":"b","name":"B","device":"x"},{"id":"c","name":"C","device":"y"},{"id":"d","name":"D","device":"z"}]`,
	} {
		if _, err := outputConfig(t, bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestOutputChoicesNormalize(t *testing.T) {
	c, err := outputConfig(t, outputsJSON)
	if err != nil {
		t.Fatal(err)
	}
	i := c.VoiceIntent()
	if !i.OutputOn("music", "virtual:1") || !i.OutputOn("music", "tape") || i.OutputOn("music", "monitor") {
		t.Fatal("configured defaults", i.Outputs)
	}
	// Saved switches win; switches for removed outputs are dropped.
	saved := i.Clone()
	saved.Outputs["music"]["monitor"] = true
	saved.Outputs["music"]["virtual:1"] = false
	saved.Outputs["gone"] = map[string]bool{"monitor": true}
	c.Intent = saved
	i = c.VoiceIntent()
	if !i.OutputOn("music", "monitor") || i.OutputOn("music", "virtual:1") || i.Outputs["gone"] != nil {
		t.Fatal(i.Outputs)
	}
	// Clones do not share switches.
	clone := i.Clone()
	clone.Outputs["music"]["monitor"] = false
	if !i.OutputOn("music", "monitor") {
		t.Fatal("clone shares output switches")
	}
	if err := i.Validate(c); err != nil {
		t.Fatal(err)
	}
}

func TestOutputWithoutDevice(t *testing.T) {
	c, err := outputConfig(t, `,"outputs":[{"id":"a","name":"A","sources":["monitor"]},{"id":"b","name":"B"}]`)
	if err != nil {
		t.Fatal("slots without a device must be valid", err)
	}
	if c.Studio.OutputDevice("") || !c.VoiceIntent().OutputOn("a", SourceMonitor) {
		t.Fatal("empty slot")
	}
}
