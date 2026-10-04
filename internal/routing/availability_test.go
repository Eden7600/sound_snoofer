package routing

import (
	"reflect"
	"testing"

	"sound-snoofer/internal/model"
)

func TestConnectedMicrophoneOptions(t *testing.T) {
	c, s := voiceFixture(t)
	if got := MicrophoneOptions(c, s); !reflect.DeepEqual(got, []string{"desk", "lav", "webcam", "off"}) {
		t.Fatal(got)
	}
	s.Devices[1].Available = false
	if got := MicrophoneOptions(c, s); !reflect.DeepEqual(got, []string{"webcam", "off"}) {
		t.Fatal("driver mistaken for hardware", got)
	}
	s.Devices[1].Available = true
	s.Devices = append(s.Devices, s.Devices[1], s.Devices[2])
	if got := MicrophoneOptions(c, s); !reflect.DeepEqual(got, []string{"off"}) {
		t.Fatal("ambiguous devices offered", got)
	}
}

func TestPreferredPlaybackDisconnectReconnect(t *testing.T) {
	c, s := voiceFixture(t)
	s.Devices = append(s.Devices, model.Device{Name: "AirPods", Driver: "wdm", Direction: "output", Available: true})
	c.Intent = c.VoiceIntent()
	c.Intent.PlaybackDevice = "speakers"
	apply := func(want string) {
		t.Helper()
		p, err := Build(c, s)
		if err != nil {
			t.Fatal(err)
		}
		applyPlan(&s, p)
		if s.Assignments["A2"] != want || s.Assignments["A1"] != "Volt ASIO" {
			t.Fatal(s.Assignments)
		}
	}
	apply("speakers")
	s.Devices[3].Available = false
	apply("AirPods")
	if c.Intent.PlaybackDevice != "speakers" {
		t.Fatal("preference erased")
	}
	s.Devices[3].Available = true
	apply("speakers")
	c.Intent.PlaybackDevice = ""
	apply("AirPods")
	s.Devices = append(s.Devices, s.Devices[3])
	if got := PlaybackOptions(c, s); !reflect.DeepEqual(got, []string{"", "AirPods"}) {
		t.Fatal("ambiguous choice offered", got)
	}
}
