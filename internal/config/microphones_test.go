package config

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestLegacyMicrophones(t *testing.T) {
	c, err := Decode(DefaultBytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Studio.MicrophoneIDs(); !slices.Equal(got, []string{"desk", "lav", "webcam"}) {
		t.Fatal(got)
	}
	webcam, n, _ := c.Studio.Microphone("webcam")
	if n != 2 || !webcam.IsDevice() || webcam.Name != "Webcam" || !webcam.Devices[0].Regex.MatchString("Microphone (2- Insta360 Link 2)") {
		t.Fatal(webcam)
	}
	in := c.Studio.ASIO[0].Inputs
	if in.Left("desk") != 1 || in.Right("desk") != 1 || in.Left("lav") != 2 || in.Left("webcam") != 0 {
		t.Fatal("legacy [desk, lav] inputs", in)
	}
	var legacy MicInputs
	json.Unmarshal([]byte(`[0, 2]`), &legacy)
	if _, ok := legacy["desk"]; ok || legacy.Left("lav") != 2 {
		t.Fatal("zero channel means absent", legacy)
	}
}

const genericStudio = `{"version":1,"profiles":{"microphones":["yeti","pair"]},"studio":{
 "asio":[{"asio_pattern":"Focusrite","presence_pattern":"Analogue","inputs":{"pair":[3,4]}}],
 "playback":[{"driver":"wdm","pattern":"speakers"}],
 "microphones":[{"id":"yeti","name":"Yeti","devices":[{"driver":"wdm","pattern":"Yeti"}]},{"id":"pair","name":"Stereo pair"}],
 "playback_sources":["virtual:1"],"voice":{}}}`

func TestGenericMicrophones(t *testing.T) {
	c, err := Decode([]byte(genericStudio))
	if err != nil {
		t.Fatal(err)
	}
	if c.Studio.Voice.Source != "auto" {
		t.Fatal("source defaults to Auto without a desk microphone", c.Studio.Voice.Source)
	}
	if in := c.Studio.ASIO[0].Inputs; in.Left("pair") != 3 || in.Right("pair") != 4 {
		t.Fatal(in)
	}
	i := c.VoiceIntent()
	i.Source = "desk"
	if i.Validate(c) == nil {
		t.Fatal("saved choice of an unconfigured microphone accepted")
	}
	i.Source = "yeti"
	if err := i.Validate(c); err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{`"id":"yeti"`, `"id":"vr-mic"`},
		{`"id":"pair","name":"Stereo pair"`, `"id":"yeti","name":"Stereo pair"`},
		{`"name":"Yeti"`, `"name":""`},
		{`{"pair":[3,4]}`, `{"yeti":[1]}`},
		{`{"pair":[3,4]}`, `{"pair":[3,4,5]}`},
		{`{"pair":[3,4]}`, `{"pair":[0]}`},
		{`["yeti","pair"]`, `["yeti","desk"]`},
		{`"devices":[{"driver":"wdm","pattern":"Yeti"}]`, `"devices":[{"driver":"asio","pattern":"Yeti"}]`},
		{`"playback_sources"`, `"fallback_mic":[{"driver":"wdm","pattern":"cam"}],"playback_sources"`},
		{`"voice":{}`, `"voice":{"source":"lav"}`},
	} {
		if _, err := Decode([]byte(strings.Replace(genericStudio, change[0], change[1], 1))); err == nil {
			t.Fatalf("accepted %s", change[1])
		}
	}
	six := `[{"id":"a","name":"A"},{"id":"b","name":"B"},{"id":"c","name":"C"},{"id":"d","name":"D"},{"id":"e","name":"E"},{"id":"f","name":"F"}]`
	b := strings.Replace(genericStudio, `[{"id":"yeti","name":"Yeti","devices":[{"driver":"wdm","pattern":"Yeti"}]},{"id":"pair","name":"Stereo pair"}]`, six, 1)
	b = strings.Replace(b, `["yeti","pair"]`, `["a"]`, 1)
	b = strings.Replace(b, `{"pair":[3,4]}`, `{}`, 1)
	if _, err := Decode([]byte(b)); err == nil {
		t.Fatal("accepted six microphones")
	}
}
