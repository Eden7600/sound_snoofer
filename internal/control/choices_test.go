package control

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func TestResetCorruptSavedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(ruleConfig), 0600)
	os.WriteFile(path+".state.json", []byte("broken"), 0600)
	c, e := config.LoadEffective(path)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &ruleClient{&fakeClient{}}
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Load: config.LoadEffective}
	states := make(chan State, 1)
	actions := make(chan Action, 1)
	done := make(chan struct{})
	go Work(ctx, c, path, "", false, deps, actions, states, done)
	s := nextState(t, states)
	actions <- Action{Kind: resetChoices, Revision: s.Revision}
	s = nextState(t, states)
	if s.StateError != "" || s.Intent.Source != "desk" || client.writes.Load() != 0 {
		t.Fatal(s)
	}
	saved, e := config.LoadEffective(path)
	if e != nil || saved.StateError != "" {
		t.Fatal(saved, e)
	}
	os.WriteFile(path+".state.json", []byte("broken again"), 0600)
	actions <- Reload
	s = nextState(t, states)
	if s.StateError != "" || s.Intent.Source != "desk" || !strings.Contains(s.Notice, "reload failed") {
		t.Fatal("invalid reload replaced active config", s)
	}
	actions <- Action{Kind: resetChoices, Revision: s.Revision}
	s = nextState(t, states)
	if s.StateError != "" || !strings.Contains(s.Notice, "Saved") {
		t.Fatal("could not reset externally corrupted state", s)
	}
	cancel()
	<-done
}

const ruleConfig = `{"version":1,"poll_ms":60000,"studio":{"asio":[{"asio_pattern":"Volt ASIO","presence_pattern":"Volt input","inputs":[1,2]}],"playback":[{"driver":"wdm","pattern":"speakers"}],"fallback_mic":[{"driver":"wdm","pattern":"webcam"}],"playback_sources":["virtual:1"],"voice":{}}}`

type ruleClient struct{ *fakeClient }

func (c *ruleClient) Snapshot() (model.Snapshot, error) {
	s := model.Snapshot{Element: &model.ProcessStatus{Known: true, Running: true}, Edition: 3, Assignments: map[string]string{}, Numbers: map[string]float32{}, Devices: []model.Device{{Name: "Volt ASIO", Driver: "asio", Direction: "output"}, {Name: "Volt input", Driver: "wdm", Direction: "input", Available: true}, {Name: "speakers", Driver: "wdm", Direction: "output", Available: true}, {Name: "webcam", Driver: "wdm", Direction: "input", Available: true}}}
	for _, slot := range model.Slots(3) {
		s.Assignments[slot] = ""
	}
	s.Assignments["A1"] = "Volt ASIO"
	s.Assignments["A2"] = "speakers"
	s.Assignments["input:3"] = "webcam"
	for n, v := range []float32{1, 1, 2, 2} {
		s.Numbers[fmt.Sprintf("Patch.asio[%d]", n)] = v
	}
	for strip := 0; strip < 8; strip++ {
		for bus := 1; bus <= 5; bus++ {
			s.Numbers[fmt.Sprintf("Strip[%d].A%d", strip, bus)] = 0
		}
		for bus := 1; bus <= 3; bus++ {
			s.Numbers[fmt.Sprintf("Strip[%d].B%d", strip, bus)] = 0
		}
	}
	return s, nil
}
func (c *ruleClient) SetNumber(string, int) error { c.writes.Add(1); return nil }
func TestRuleWorkerSaveBeforeApplyAndStaleCommands(t *testing.T) {
	c, e := config.Decode([]byte(ruleConfig))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &ruleClient{&fakeClient{}}
	saves := 0
	fail := false
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Load: func(string) (config.Config, error) { return c, nil }, Save: func(_ string, _ config.Config, i *config.Intent, _ string) (string, error) {
		saves++
		if client.writes.Load() != 0 {
			t.Fatal("write before save")
		}
		if fail {
			return "", fmt.Errorf("disk full")
		}
		return "saved", nil
	}}
	states := make(chan State, 1)
	actions := make(chan Action, 8)
	done := make(chan struct{})
	go Work(ctx, c, "unused", "", false, deps, actions, states, done)
	s := nextState(t, states)
	actions <- Action{Kind: editRule, Row: "source", Value: "lav", Revision: s.Revision}
	s = nextState(t, states)
	if s.Intent.Source != "lav" || saves != 1 || client.writes.Load() != 0 {
		t.Fatal(s)
	}
	actions <- Action{Kind: editRule, Row: "source", Value: "webcam", Revision: s.Revision - 1}
	s = nextState(t, states)
	if s.Intent.Source != "lav" || saves != 1 || !strings.Contains(s.Notice, "Stale") {
		t.Fatal(s)
	}
	fail = true
	actions <- Action{Kind: editRule, Row: "mode", Value: "direct", Revision: s.Revision}
	s = nextState(t, states)
	if s.Intent.Mode != "element" || !strings.Contains(s.Notice, "disk full") {
		t.Fatal(s)
	}
	actions <- Reload
	s = nextState(t, states)
	if s.Intent.Source != "desk" {
		t.Fatal("reload", s)
	}
	cancel()
	<-done
}

func TestInvalidSavedChoicesBlockLive(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	c.StateError = "corrupt saved state"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps := Dependencies{Open: func(string) (Client, error) { return &ruleClient{&fakeClient{}}, nil }, Acquire: func() (func(), error) { t.Fatal("acquired with corrupt state"); return nil, nil }}
	states := make(chan State, 1)
	done := make(chan struct{})
	go Work(ctx, c, "unused", "", true, deps, make(chan Action), states, done)
	s := nextState(t, states)
	if s.Live || s.StateError == "" {
		t.Fatal(s)
	}
	cancel()
	<-done
}
