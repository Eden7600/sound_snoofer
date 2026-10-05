package control

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type measuredClient struct {
	*ruleClient
	snapshot model.Snapshot
}

func (b *measuredClient) Snapshot() (model.Snapshot, error) {
	s := b.snapshot
	s.Numbers = maps.Clone(s.Numbers)
	s.Assignments = maps.Clone(s.Assignments)
	return s, nil
}
func (b *measuredClient) ParameterSnapshot() (model.Snapshot, error) { return b.Snapshot() }
func (b *measuredClient) Set(target string, d model.Device) error {
	b.snapshot.Assignments[target] = d.Name
	return nil
}
func (b *measuredClient) SetNumber(p string, v int) error {
	b.snapshot.Numbers[p] = float32(v)
	return nil
}
func TestChoiceAckAndVerifiedReadbackLatency(t *testing.T) {
	c, e := config.Decode([]byte(ruleConfig))
	if e != nil {
		t.Fatal(e)
	}
	c.PollMS = 100
	c.VR = &config.VR{Input: 4, Headsets: []config.Headset{{ID: "headset", Playback: "^VR Phones$"}}}
	if e = c.Validate(); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	data, e := json.Marshal(c)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	c, e = config.LoadEffective(path)
	if e != nil {
		t.Fatal(e)
	}
	base := &ruleClient{&fakeClient{}}
	snapshot, _ := base.Snapshot()
	snapshot.SteamVR = &model.ProcessStatus{Known: true, Running: true}
	snapshot.Devices = append(snapshot.Devices, model.Device{Name: "VR Phones", Driver: "wdm", Direction: "output", Available: true})
	b := &measuredClient{ruleClient: base, snapshot: snapshot}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	deps := Dependencies{Open: func(string) (Client, error) { return b, nil }, Acquire: func() (func(), error) { return func() {}, nil }, Load: config.LoadEffective}
	go Work(ctx, c, path, "", true, deps, actions, states, done)
	defer func() { cancel(); <-done }()
	wait := func(predicate func(State) bool) State {
		deadline := time.NewTimer(4 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case s := <-states:
				if predicate(s) {
					return s
				}
			case <-deadline.C:
				t.Fatal("ack/readback exceeded 4s bound")
				return State{}
			}
		}
	}
	settled := func(s State) bool {
		return s.Connected && s.Plan != nil && !s.Plan.HasChanges() && !s.Plan.HasUnresolved() && s.Error == ""
	}
	s := wait(settled)
	for n, edit := range []SettingEdit{{Row: "monitor", Value: "pre"}, {Row: "output", Value: "VR Phones"}} {
		before := time.Now()
		actions <- Action{Kind: editRule, Row: edit.Row, Value: edit.Value, Revision: s.Revision, ID: uint64(n + 1)}
		s = wait(func(s State) bool { return s.EditAck == uint64(n+1) })
		ack := time.Now()
		if s.EditError != "" {
			t.Fatal(s.EditError)
		}
		if !settled(s) {
			s = wait(settled)
		}
		verified := time.Now()
		t.Logf("%s input-to-ack=%s ack-to-verified=%s (fake native readback, real save/worker)", edit.Row, ack.Sub(before), verified.Sub(ack))
	}
	loaded, e := config.LoadEffective(path)
	if e != nil || loaded.StateError != "" || loaded.VoiceIntent().PlaybackDevice != "VR Phones" || s.Plan.Topology.PlaybackTarget == "" {
		t.Fatal("VR picker/save/load/plan mismatch", e, loaded.StateError)
	}
}
