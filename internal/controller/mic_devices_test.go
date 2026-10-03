package controller

import (
	"context"
	"testing"
)

func TestMicDeviceReleaseVerificationAndRecovery(t *testing.T) {
	for _, failure := range []string{"none", "setter", "readback"} {
		t.Run(failure, func(t *testing.T) {
			c, b := voiceController(t)
			c.Config.Intent = c.Config.VoiceIntent()
			c.Config.Intent.Source = "off"
			if failure == "setter" {
				b.setErr = "input:3"
			}
			if failure == "readback" {
				b.never = true
			}
			p, err := c.Plan()
			if err != nil {
				t.Fatal(err)
			}
			err = c.Apply(context.Background(), p)
			if failure != "none" {
				if err == nil {
					t.Fatal("unverified disconnection accepted")
				}
				if b.s.Assignments["input:3"] != "webcam" {
					t.Fatal("unexpected device state")
				}
				b.setErr = ""
				b.never = false
				p, err = c.Plan()
				if err != nil {
					t.Fatal(err)
				}
				err = c.Apply(context.Background(), p)
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.s.Assignments["input:3"] != "" || b.s.Assignments["A1"] != "Volt ASIO" || b.s.Assignments["A2"] != "AirPods" {
				t.Fatal("incorrect device assignments", b.s.Assignments)
			}
			for _, w := range b.writes {
				if w.target == "A1" || w.target == "A2" {
					t.Fatal("Off rewrote an output", w)
				}
			}
			p, err = c.Plan()
			if err != nil || p.HasChanges() {
				t.Fatal("Off failed to converge", err)
			}
		})
	}
}
