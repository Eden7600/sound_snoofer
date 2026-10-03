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
	if c.VoiceIntent().Source != "off" || c.VoiceIntent().MicActive() {
		t.Fatal("old disabled state not Off")
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
