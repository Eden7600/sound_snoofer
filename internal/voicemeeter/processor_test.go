package voicemeeter

import "testing"

func TestConfiguredProcessorProcess(t *testing.T) {
	c, e := connect(&fakeAPI{edition: 3})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.processList = func() ([]string, error) { return []string{"element.exe", "Processor.EXE"}, nil }
	c.SetProcessorProcess("processor.exe")
	s, e := c.Snapshot()
	if e != nil || !s.Element.Running {
		t.Fatal("configured processor not observed", e)
	}
	c.processList = func() ([]string, error) { return []string{"element.exe"}, nil }
	if s, e = c.Snapshot(); e != nil || s.Element.Running {
		t.Fatal("default processor observed after configuring another", e)
	}
}
