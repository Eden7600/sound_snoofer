package controller

import (
	"context"
	"fmt"
	"testing"

	"sound-snoofer/internal/config"
)

func TestVROffUnderRecorderConflict(t *testing.T) {
	for _, input := range []int{4, 5} {
		for _, state := range []string{"Stopped", "Recording", "Paused", "Playing", "Unknown", "Conflict"} {
			t.Run(fmt.Sprintf("%d/%s", input, state), func(t *testing.T) {
				c, b := recorderController(t)
				c.Config.VR = &config.VR{Input: input, Headsets: []config.Headset{{ID: "vr", Microphone: "^VR Mic$"}}}
				if e := c.Config.Validate(); e != nil {
					t.Fatal(e)
				}
				b.s.Assignments[fmt.Sprintf("input:%d", input)] = "VR Mic"
				parameter := fmt.Sprintf("Strip[%d].B1", input-1)
				b.s.Numbers[parameter] = 1
				c.Config.Intent.Recording.ComputerEnabled = true
				b.s.Numbers["Strip[5].B1"] = 1
				for _, p := range []string{"Recorder.record", "Recorder.stop", "Recorder.pause", "Recorder.play"} {
					b.r.Values[p] = 0
				}
				switch state {
				case "Stopped":
					b.r.Values["Recorder.stop"] = 1
				case "Recording":
					b.r.Values["Recorder.record"] = 1
				case "Paused":
					b.r.Values["Recorder.pause"] = 1
				case "Playing":
					b.r.Values["Recorder.play"] = 1
				case "Unknown":
					b.r.Error = "unavailable"
				case "Conflict":
					b.r.Values["Recorder.record"] = 1
					b.r.Values["Recorder.mode.MultiTrack"] = 1
				}
				c.Config.Intent.Source = "off"
				p, e := c.Plan()
				if e != nil {
					t.Fatal(e)
				}
				// Unrelated guarded operations may stop the pass, but mic Off must complete first.
				_ = c.Apply(context.Background(), p)
				if b.s.Numbers[parameter] != 0 {
					t.Fatal("VR mic capture remained connected")
				}
				for _, w := range b.writesRecorder {
					if w == "Recorder.record" || w == "Recorder.stop" || w == "Recorder.play" || w == "Recorder.replay" {
						t.Fatal("transport changed", w)
					}
				}
				if b.s.Assignments["A1"] != "Volt ASIO" || b.s.Assignments["A2"] != "AirPods" || b.s.Numbers["Strip[5].B1"] != 1 {
					t.Fatal("playback/capture changed")
				}
			})
		}
	}
}

func TestRepeatedReadErrorRemainsVisibleWithoutRepeatedLog(t *testing.T) {
	c, b := voiceController(t)
	b.err = fmt.Errorf("persistent failure")
	events := 0
	c.Emit = func(e Event) {
		if e.Kind == "error" {
			events++
		}
	}
	for n := 0; n < 3; n++ {
		c.Step(context.Background(), false)
		if c.Error == "" {
			t.Fatal("deduplication erased persistent error")
		}
	}
	if events != 1 {
		t.Fatal("error log not deduplicated", events)
	}
	b.err = nil
	c.Step(context.Background(), false)
	if c.Error != "" {
		t.Fatal("resolved error retained", c.Error)
	}
}
