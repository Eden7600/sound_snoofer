package config

import (
	"strings"
	"testing"
)

const speakersID = "{0.0.0.00000000}.{b3e9496a-c7da-4c76-92ce-6c50a81f3df0}"

func identityConfig(t *testing.T) Config {
	t.Helper()
	b := strings.Replace(string(DefaultBytes()), `"pattern": "(?i)airpods"`, `"id": "`+speakersID+`", "name": "Speakers (2- Arena)"`, 1)
	b = strings.Replace(b, `"move_playback_routing": true`, `"move_playback_routing": true, "outputs": [{"id": "music", "name": "Music", "device": "Speakers (2- Arena)", "device_id": "`+speakersID+`"}]`, 1)
	c, err := Decode([]byte(b))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestIdentityResolution(t *testing.T) {
	c := identityConfig(t)
	entry := c.Studio.Playback[0]
	if entry.ID != speakersID || !entry.Regex.MatchString("Speakers (2- Arena)") {
		t.Fatal("unresolved identity matches its label", entry)
	}
	// Renamed: matches the new name only.
	resolved := c.Resolve(DeviceNames{ByID: map[string]string{speakersID: "Speakers (3- Arena)"}})
	if re := resolved.Studio.Playback[0].Regex; !re.MatchString("Speakers (3- Arena)") || re.MatchString("Speakers (2- Arena)") || re.MatchString("Speakers (3- Arena) 2") {
		t.Fatal("rename")
	}
	if o := resolved.Studio.Outputs[0]; o.Device != "Speakers (3- Arena)" || o.Inactive {
		t.Fatal(o)
	}
	// Resolution never changes the saved configuration.
	if c.Studio.Playback[0].Regex.MatchString("Speakers (3- Arena)") || c.Studio.Outputs[0].Device != "Speakers (2- Arena)" {
		t.Fatal("base mutated")
	}
	// Disconnected: keeps its label for bus ownership; the slot is inactive.
	gone := c.Resolve(DeviceNames{})
	if !gone.Studio.Playback[0].Regex.MatchString("Speakers (2- Arena)") || !gone.Studio.Outputs[0].Inactive {
		t.Fatal("disconnected")
	}
	// Another connected device took the label: the entry matches nothing.
	taken := c.Resolve(DeviceNames{Available: map[string]bool{"Speakers (2- Arena)": true}})
	if taken.Studio.Playback[0].Regex.MatchString("Speakers (2- Arena)") {
		t.Fatal("label collision matched another device")
	}
	// Pattern entries are untouched.
	if !taken.Studio.Playback[1].Regex.MatchString("SteelSeries Arena 9") {
		t.Fatal("pattern entry changed")
	}
}

func TestIdentityValidation(t *testing.T) {
	for _, bad := range []string{
		`"id": "x", "pattern": "y"`,
		`"id": "x"`,
		`"driver2": "x"`,
	} {
		b := strings.Replace(string(DefaultBytes()), `"pattern": "(?i)airpods"`, bad, 1)
		if _, err := Decode([]byte(b)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	b := strings.Replace(string(DefaultBytes()), `"asio_pattern": "(?i)^Universal Audio Volt$"`, `"asio_id": "{7FA0A3EC}", "asio_name": "Universal Audio Volt"`, 1)
	if _, err := Decode([]byte(b)); err != nil {
		t.Fatal("interface by identity", err)
	}
}
