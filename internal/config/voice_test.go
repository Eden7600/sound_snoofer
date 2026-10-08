package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"sound-snoofer/internal/ownership"
)

const voiceJSON = `{"version":1,"studio":{"asio":[{"asio_pattern":"Volt ASIO","presence_pattern":"Volt input","inputs":[1,2]}],"playback":[{"driver":"wdm","pattern":"speakers"}],"fallback_mic":[{"driver":"wdm","pattern":"webcam"}],"playback_sources":["virtual:1"],"voice":{}}}`

func TestVoiceConfig(t *testing.T) {
	c, e := Decode([]byte(voiceJSON))
	if e != nil {
		t.Fatal(e)
	}
	i := c.VoiceIntent()
	if !i.Enabled || i.Source != "desk" || i.Mode != "element" || i.Monitor != "off" {
		t.Fatal(i)
	}
	if c.ValidateEdition(2) == nil {
		t.Fatal("Banana accepted")
	}
	for _, change := range []string{`"source":"unknown"`, `"mode":"unknown"`, `"monitor":"unknown"`} {
		if _, e = Decode([]byte(strings.Replace(voiceJSON, `"voice":{}`, `"voice":{`+change+`}`, 1))); e == nil {
			t.Fatal(change)
		}
	}
	if _, e = Decode([]byte(strings.Replace(voiceJSON, "virtual:1", "virtual:2", 1))); e == nil {
		t.Fatal("AUX accepted")
	}
}

// The embedded default is the effective factory configuration.
func TestVoiceExample(t *testing.T) {
	c, e := Decode(DefaultBytes())
	if e != nil {
		t.Fatal(e)
	}
	i := c.VoiceIntent()
	if i == nil || i.Source != "desk" || i.Mode != "element" || i.Monitor != "off" {
		t.Fatal(i)
	}
	if c.Profiles == nil || !slices.Equal(c.Profiles.Microphones, DefaultProfiles().Microphones) {
		t.Fatal("default configuration must state the Normal microphone priority", c.Profiles)
	}
}

func TestAudioConfigRejectsDeckProfiles(t *testing.T) {
	b := strings.Replace(string(DefaultBytes()), `"version": 1,`, `"version": 1, "stream_deck": {"profiles": []},`, 1)
	if _, e := Decode([]byte(b)); e == nil || !strings.Contains(e.Error(), "unknown field") {
		t.Fatal("unread stream_deck schema accepted", e)
	}
}
func TestIntentPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(voiceJSON), 0600)
	c, e := LoadEffective(path)
	if e != nil || c.StateError != "" || c.StateToken != "missing" {
		t.Fatal(c, e)
	}
	i := c.VoiceIntent()
	i.PlaybackDevice = "speakers"
	i.Source = "lav"
	i.Mode = "direct"
	token, e := SaveIntent(path, c, i, c.StateToken)
	if e != nil {
		t.Fatal(e)
	}
	c, e = LoadEffective(path)
	if e != nil || c.StateError != "" || c.VoiceIntent().Source != "lav" || c.VoiceIntent().PlaybackDevice != "speakers" || c.StateToken != token {
		t.Fatal(c, e)
	}
	if _, e = SaveIntent(path, c, i, "missing"); e == nil {
		t.Fatal("lost update")
	}
	release, e := ownership.AcquireState(path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = SaveIntent(path, c, i, token); e == nil {
		t.Fatal("concurrent write")
	}
	release()
	for _, bad := range []string{`garbage`, `{"version":2}`, `{"version":1,"version":1}`, `{"version":1,"enabled":true,"source":"desk","mode":"direct","monitor":"off","playback":{"virtual:3":true}}`} {
		os.WriteFile(path+".state.json", []byte(bad), 0600)
		c, e = LoadEffective(path)
		if e != nil || c.StateError == "" {
			t.Fatal("invalid state accepted", bad)
		}
		base, _ := Load(path)
		if _, e = SaveIntent(path, base, base.VoiceIntent(), c.StateToken); e != nil {
			t.Fatal("reset", e)
		}
	}
	c, _ = LoadEffective(path)
	b, _ := os.ReadFile(path + ".state.json")
	if strings.Contains(string(b), "live") {
		t.Fatal("live persisted")
	}
	if _, e = SaveIntent(filepath.Join(path, "missing"), c, i, "missing"); e == nil {
		t.Fatal("unwritable accepted")
	}
}

func TestPlaybackPreferenceValidation(t *testing.T) {
	c, err := Decode([]byte(voiceJSON))
	if err != nil {
		t.Fatal(err)
	}
	intent := c.VoiceIntent()
	if intent.PlaybackDevice != "" {
		t.Fatal("legacy intent must be automatic")
	}
	intent.PlaybackDevice = "unmanaged headphones"
	if err := intent.Validate(c); err == nil {
		t.Fatal("unmanaged output accepted")
	}
	intent.PlaybackDevice = "speakers disconnected"
	if err := intent.Validate(c); err != nil {
		t.Fatal("saved preference must survive absence:", err)
	}
}
