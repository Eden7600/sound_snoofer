package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"strings"
	"testing"
)

func (c *testClient) SetNumber(p string, v int) error {
	c.writes++
	c.s.Numbers[p] = float32(v)
	return nil
}
func TestCommandsShareSavedIntent(t *testing.T) {
	for _, cmd := range []string{"plan", "apply", "watch"} {
		t.Run(cmd, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "voice.json")
			data := []byte(`{"version":1,"studio":{"asio":[{"asio_pattern":"Volt ASIO","presence_pattern":"Volt input","inputs":[1,2]}],"playback":[{"driver":"wdm","pattern":"speakers"}],"fallback_mic":[{"driver":"wdm","pattern":"webcam"}],"playback_sources":["virtual:1"],"voice":{}}}`)
			os.WriteFile(path, data, 0600)
			c, _ := config.LoadEffective(path)
			i := c.VoiceIntent()
			i.Source = "lav"
			i.Mode = "direct"
			if _, e := config.SaveIntent(path, c, i, c.StateToken); e != nil {
				t.Fatal(e)
			}
			s := model.Snapshot{Edition: 3, Assignments: map[string]string{}, Numbers: map[string]float32{}, Devices: []model.Device{{Name: "Volt ASIO", Driver: "asio", Direction: "output"}, {Name: "Volt input", Driver: "wdm", Direction: "input", Available: true}, {Name: "speakers", Driver: "wdm", Direction: "output", Available: true}, {Name: "webcam", Driver: "wdm", Direction: "input", Available: true}}}
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
			s.Numbers["Strip[5].A2"] = 1
			s.Numbers["Strip[1].B3"] = 1
			client := &testClient{s: s}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			deps := Deps{Open: func(string) (Client, error) { return client, nil }, Acquire: func() (func(), error) { return func() {}, nil }, Clock: &testClock{cancel: cancel}}
			var out, errout bytes.Buffer
			code := Run(ctx, []string{cmd, "--config", path, "--json"}, &out, &errout, deps)
			if code != 0 || !strings.Contains(out.String(), `"preferred"`) || !strings.Contains(out.String(), `"lav"`) || client.writes != 0 {
				t.Fatal(cmd, code, out.String(), errout.String(), client.writes)
			}
		})
	}
}
func TestCorruptStateRejectedBeforeMixer(t *testing.T) {
	for _, cmd := range []string{"plan", "apply", "watch"} {
		_, deps, path, _ := setup(t)
		os.WriteFile(path, []byte(`{"version":1,"studio":{"asio":[{"asio_pattern":"Volt","presence_pattern":"Volt","inputs":[1,2]}],"playback":[{"driver":"wdm","pattern":"speaker"}],"fallback_mic":[{"driver":"wdm","pattern":"webcam"}],"voice":{}}}`), 0600)
		os.WriteFile(path+".state.json", []byte("invalid"), 0600)
		deps.Open = func(string) (Client, error) { t.Fatal("opened mixer"); return nil, nil }
		var out bytes.Buffer
		if Run(context.Background(), []string{cmd, "--config", path}, &out, &out, deps) != 2 {
			t.Fatal(out.String())
		}
	}
}
