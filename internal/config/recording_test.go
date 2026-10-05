package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordingProfileAndPersistence(t *testing.T) {
	raw := strings.Replace(voiceJSON, `"voice":{}`, `"voice":{},"recording":{}`, 1)
	c, e := Decode([]byte(raw))
	if e != nil {
		t.Fatal(e)
	}
	i := c.VoiceIntent()
	if i.Recording.MicEnabled || i.Recording.ComputerEnabled || i.Recording.MicTap != "pre" {
		t.Fatal(i)
	}
	if c.ValidateEdition(2) == nil {
		t.Fatal("Banana accepted")
	}
	for _, sources := range []string{`["virtual:2"]`, `["virtual:1","virtual:1"]`, `["input:1"]`} {
		if _, e := Decode([]byte(strings.Replace(raw, `"recording":{}`, `"recording":{"computer_sources":`+sources+`}`, 1))); e == nil {
			t.Fatal(sources)
		}
	}
	if _, e := Decode([]byte(strings.Replace(raw, `"voice":{},`, "", 1))); e == nil {
		t.Fatal("missing voice accepted")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(voiceJSON), 0600)
	old, _ := LoadEffective(path)
	prior := old.VoiceIntent()
	prior.Source = "lav"
	prior.Mode = "direct"
	if _, e := SaveIntent(path, old, prior, old.StateToken); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(path, []byte(raw), 0600)
	c, e = LoadEffective(path)
	if e != nil || c.StateError != "" {
		t.Fatal(c, e)
	}
	i = c.VoiceIntent()
	if i.Source != "lav" || i.Mode != "direct" || i.Recording.MicEnabled {
		t.Fatal(i)
	}
	i.Recording.MicEnabled = true
	i.Recording.MicTap = "post"
	if _, e := SaveIntent(path, c, i, c.StateToken); e != nil {
		t.Fatal(e)
	}
	c, _ = LoadEffective(path)
	if !c.VoiceIntent().Recording.MicEnabled {
		t.Fatal("not saved")
	}
	i.Recording.MicTap = "invalid"
	if i.Validate(c) == nil {
		t.Fatal("bad tap")
	}
	c.Studio.Recording = nil
	if i.Validate(c) == nil {
		t.Fatal("missing profile")
	}
}

func TestRehearsalNormalizesOffAndDirect(t *testing.T) {
	for _, source := range []string{"desk", "off"} {
		for _, mode := range []string{"element", "direct"} {
			i := &Intent{Source: source, Enabled: source != "off", Mode: mode, Recording: &RecordingChoices{ToVST: true, Loop: true, MicTap: "pre"}}
			i.NormalizeRehearsal()
			if i.Recording.ToVST != (source == "desk" && mode == "element") || !i.Recording.Loop {
				t.Fatal(i)
			}
		}
	}
}
