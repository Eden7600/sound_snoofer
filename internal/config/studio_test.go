package config

import "testing"

func TestStudioValidation(t *testing.T) {
	valid := `{"version":1,"studio":{"asio_pattern":"volt","presence_pattern":"volt","playback":[{"driver":"wdm","pattern":"speakers"}],"fallback_mic":[{"driver":"wdm","pattern":"webcam"}],"playback_sources":["virtual:3"]}}`
	c, e := Decode([]byte(valid))
	if e != nil {
		t.Fatal(e)
	}
	if c.ValidateEdition(2) == nil {
		t.Fatal("Banana accepted virtual:3")
	}
	if e = c.ValidateEdition(3); e != nil {
		t.Fatal(e)
	}
	c.Studio.PlaybackSources = []string{"virtual:1", "virtual:1"}
	if c.Validate() == nil {
		t.Fatal("duplicate source")
	}
	c.Studio.PlaybackSources = []string{"hardware:99"}
	if c.Validate() == nil {
		t.Fatal("invalid source")
	}
}
