package controller

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"sound-snoofer/internal/config"
)

type tracedRoutingBackend struct {
	*recorderFake
	sends map[string][]int
}

func (b *tracedRoutingBackend) SetNumber(param string, value int) error {
	b.sends[param] = append(b.sends[param], value)
	return b.recorderFake.SetNumber(param, value)
}

func TestSettingChangesWriteOnlyAffectedSends(t *testing.T) {
	cases := []struct {
		name string
		edit func(*config.Intent)
		want map[string][]int
	}{
		{"monitor-off", func(i *config.Intent) { i.Monitor = "off" }, map[string][]int{"Strip[0].A2": {0}}},
		{"monitor-post", func(i *config.Intent) { i.Monitor = "post" }, map[string][]int{"Strip[0].A2": {0}, "Strip[6].A2": {1}}},
		{"record-mic-off", func(i *config.Intent) { i.Recording.MicEnabled = false }, map[string][]int{"Strip[0].B1": {0}}},
		{"record-computer-off", func(i *config.Intent) { i.Recording.ComputerEnabled = false }, map[string][]int{"Strip[5].B1": {0}}},
		{"record-post", func(i *config.Intent) { i.Recording.MicTap = "post" }, map[string][]int{"Strip[0].B1": {0}, "Strip[6].B1": {1}}},
		{"direct", func(i *config.Intent) { i.Mode = "direct" }, map[string][]int{"Strip[0].B2": {0}, "Strip[6].B3": {0}, "Strip[0].B3": {1}}},
		{"lav", func(i *config.Intent) { i.Source = "lav" }, map[string][]int{"Strip[0].B2": {0}, "Strip[1].B2": {1}, "Strip[0].A2": {0}, "Strip[1].A2": {1}, "Strip[0].B1": {0}, "Strip[1].B1": {1}}},
		{"playback-off", func(i *config.Intent) { i.Playback["virtual:1"] = false }, map[string][]int{"Strip[5].A2": {0}}},
		{"mic-off", func(i *config.Intent) { i.Source = "off" }, map[string][]int{"Strip[0].A2": {0}, "Strip[0].B1": {0}, "Strip[0].B2": {0}, "Strip[6].B3": {0}, "Patch.asio[0]": {0}, "Patch.asio[1]": {0}, "Patch.asio[2]": {0}, "Patch.asio[3]": {0}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, b := recorderController(t)
			c.Config.Intent.Monitor = "pre"
			c.Config.Intent.Recording.ComputerEnabled = true
			p, err := c.Plan()
			if err != nil {
				t.Fatal(err)
			}
			if err = c.Apply(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			trace := &tracedRoutingBackend{recorderFake: b, sends: map[string][]int{}}
			c.Backend = trace
			b.inspect = func() {
				for _, bus := range []string{"A2", "B1", "B2", "B3"} {
					count := 0
					for _, strip := range []int{0, 1, 2, 6} {
						if b.s.Numbers[fmt.Sprintf("Strip[%d].%s", strip, bus)] != 0 {
							count++
						}
					}
					if count > 1 {
						t.Fatalf("duplicate voice on %s", bus)
					}
				}
				if b.s.Numbers["Strip[6].B2"] != 0 {
					t.Fatal("feedback")
				}
			}
			tc.edit(c.Config.Intent)
			p, err = c.Plan()
			if err != nil {
				t.Fatal(err)
			}
			if err = c.Apply(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(trace.sends, tc.want) {
				t.Fatalf("writes = %v, want %v", trace.sends, tc.want)
			}
			trace.sends = map[string][]int{}
			p, err = c.Plan()
			if err != nil {
				t.Fatal(err)
			}
			if err = c.Apply(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			if len(trace.sends) != 0 {
				t.Fatal("redundant writes", trace.sends)
			}
		})
	}
}
