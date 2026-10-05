package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVRPlaybackPreferenceRoundTrip(t *testing.T) {
	c, e := Decode([]byte(voiceJSON))
	if e != nil {
		t.Fatal(e)
	}
	c.VR = &VR{Input: 4, Headsets: []Headset{{ID: "headset", Playback: "^VR Phones$"}}}
	if e = c.Validate(); e != nil {
		t.Fatal(e)
	}
	i := c.VoiceIntent()
	i.PlaybackDevice = "VR Phones"
	if e = i.Validate(c); e != nil {
		t.Fatal("listed VR preference rejected", e)
	}
	p := filepath.Join(t.TempDir(), "config.json")
	// Save/load does not require the endpoint to remain connected.
	b, e := json.Marshal(c)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = SaveIntent(p, c, i, "missing"); e != nil {
		t.Fatal(e)
	}
	loaded, e := LoadEffective(p)
	if e != nil || loaded.StateError != "" || loaded.VoiceIntent().PlaybackDevice != "VR Phones" {
		t.Fatalf("round trip: %v, %#v", e, loaded)
	}
	c.Intent = nil
	i.PlaybackDevice = "unconfigured"
	if i.Validate(c) == nil {
		t.Fatal("unowned endpoint accepted")
	}
}
func FuzzDecode(f *testing.F) {
	f.Add(voiceJSON)
	f.Add("{}")
	f.Add("{\"version\":1,\"version\":2}")
	f.Fuzz(func(t *testing.T, s string) { _, _ = Decode([]byte(s)) })
}
