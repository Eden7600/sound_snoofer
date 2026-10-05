package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOffSourcePersistenceAndCompatibility(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(voiceJSON), 0600)
	c, _ := LoadEffective(path)
	i := c.VoiceIntent()
	i.Enabled = false
	if _, e := SaveIntent(path, c, i, c.StateToken); e != nil {
		t.Fatal(e)
	}
	c, _ = LoadEffective(path)
	if c.VoiceIntent().Source != "desk" || c.VoiceIntent().MicActive() {
		t.Fatal("disabled stack lost target")
	}
	i = c.VoiceIntent()
	i.Source = "lav"
	i.Enabled = true
	if _, e := SaveIntent(path, c, i, c.StateToken); e != nil {
		t.Fatal(e)
	}
	c, _ = LoadEffective(path)
	if !c.VoiceIntent().MicActive() || c.VoiceIntent().Source != "lav" {
		t.Fatal("restore failed")
	}
	i.Source = "off"
	i.Enabled = true
	if i.MicActive() {
		t.Fatal("Off enabled")
	}
	if _, e := SaveIntent(path, c, i, c.StateToken); e != nil {
		t.Fatal(e)
	}
	c, _ = LoadEffective(path)
	if c.VoiceIntent().MicActive() {
		t.Fatal("Off lost on reload")
	}
	c, e := Decode([]byte(strings.Replace(voiceJSON, `"voice":{}`, `"voice":{"source":"off"}`, 1)))
	if e != nil || c.VoiceIntent().MicActive() {
		t.Fatal("config Off", e)
	}
}

func TestDisabledTargetsAndMuteSurviveSaveReload(t *testing.T) {
	c, err := Decode([]byte(voiceJSON))
	if err != nil {
		t.Fatal(err)
	}
	i := c.VoiceIntent()
	i.Enabled = false
	i.Source = "webcam"
	i.MicMuted = true
	i.VRProfile = &ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}
	path := filepath.Join(t.TempDir(), "audio")
	if _, err = SaveIntent(path, c, i, "missing"); err != nil {
		t.Fatal(err)
	}
	loaded := LoadChoices(path, c)
	actual := loaded.VoiceIntent()
	if loaded.StateError != "" || actual.Enabled || actual.Source != "webcam" || !actual.MicMuted || actual.VRProfile.Source != "auto" {
		t.Fatal(loaded.StateError, actual)
	}
	actual.VRProfile.Source = "off"
	actual.Enabled = true
	if _, err = SaveIntent(path, c, actual, loaded.StateToken); err != nil {
		t.Fatal(err)
	}
	legacy := LoadChoices(path, c).VoiceIntent()
	if legacy.Enabled || legacy.Source != "webcam" || legacy.VRProfile.Source != "auto" {
		t.Fatal("legacy VR Off re-enabled or lost Normal target", legacy)
	}
}
