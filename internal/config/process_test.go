package config

import (
	"strings"
	"testing"
)

func TestProcessNames(t *testing.T) {
	c, e := Decode(DefaultBytes())
	if e != nil {
		t.Fatal(e)
	}
	if c.ProcessorProcess() != DefaultProcessorProcess || (ProfilePolicy{}).ProcessName() != DefaultVRProcess {
		t.Fatal("omitted process names must keep the defaults")
	}
	custom := strings.Replace(string(DefaultBytes()), `"monitor": "off"`, `"monitor": "off", "processor_process": "processor.exe"`, 1)
	if c, e = Decode([]byte(custom)); e != nil || c.ProcessorProcess() != "processor.exe" {
		t.Fatal(c.ProcessorProcess(), e)
	}
	for _, bad := range []string{` element.exe`, `C:\apps\element.exe`, `bin/element.exe`} {
		b := strings.Replace(string(DefaultBytes()), `"monitor": "off"`, `"monitor": "off", "processor_process": "`+bad+`"`, 1)
		if _, e := Decode([]byte(b)); e == nil {
			t.Fatalf("accepted processor_process %q", bad)
		}
	}
	p := ProfilePolicy{Devices: VR{Input: 4}, Microphones: []string{"normal"}, Playback: []Candidate{{Driver: "normal"}}, Choices: ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}, Process: "vr/server.exe"}
	if p.Validate() == nil {
		t.Fatal("accepted VR process path")
	}
	p.Process = "vrmonitor.exe"
	if e := p.Validate(); e != nil || p.ProcessName() != "vrmonitor.exe" {
		t.Fatal(p.ProcessName(), e)
	}
}
