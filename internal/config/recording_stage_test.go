package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectRecordingStageLoadAndSave(t *testing.T) {
	raw := strings.Replace(voiceJSON, `"voice":{}`, `"voice":{},"recording":{}`, 1)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadEffective(path)
	if err != nil {
		t.Fatal(err)
	}
	i := c.VoiceIntent()
	i.Mode = "direct"
	i.Recording = &RecordingChoices{MicEnabled: true, ComputerEnabled: true, MicTap: "post"}
	legacy, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path+".state.json", legacy, 0600); err != nil {
		t.Fatal(err)
	}
	c, err = LoadEffective(path)
	if err != nil || c.StateError != "" || c.Intent.Recording.MicTap != "pre" {
		t.Fatal("legacy normalization", err, c.StateError)
	}
	if _, err = SaveIntent(path, c, i, c.StateToken); err != nil {
		t.Fatal(err)
	}
	if i.Recording.MicTap != "post" {
		t.Fatal("save mutated caller")
	}
	saved, err := os.ReadFile(path + ".state.json")
	if err != nil {
		t.Fatal(err)
	}
	var result Intent
	if err = json.Unmarshal(saved, &result); err != nil {
		t.Fatal(err)
	}
	if result.Recording.MicTap != "pre" || !result.Recording.MicEnabled || !result.Recording.ComputerEnabled {
		t.Fatal(result)
	}
	result.Mode = "element"
	result.NormalizeRecordingStage()
	if result.Recording.MicTap != "pre" {
		t.Fatal("Post was restored automatically")
	}
	i.Recording.MicTap = "invalid"
	if _, err = SaveIntent(path, c, i, c.StateToken); err == nil {
		t.Fatal("invalid stage accepted")
	}
}
