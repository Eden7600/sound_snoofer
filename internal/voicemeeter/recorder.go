package voicemeeter

import (
	"fmt"
	"voice-snooter/internal/model"
)

func (c *Client) Recorder() (model.RecorderSnapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := model.RecorderSnapshot{Values: map[string]float32{}}
	if e := c.refresh(); e != nil {
		return r, e
	}
	edition, code := c.api.Edition()
	if e := status("recorder edition", code); e != nil {
		return r, e
	}
	if edition != 3 {
		return r, fmt.Errorf("recorder profile requires Potato")
	}
	for _, p := range model.RecorderParameters() {
		v, code := c.api.GetNumber(p)
		if e := status("read "+p, code); e != nil {
			return model.RecorderSnapshot{}, e
		}
		r.Values[p] = v
	}
	return r, nil
}
func (c *Client) SetRecorder(p string, v int) error {
	valid := (p == "Recorder.record" || p == "Recorder.stop") && v == 1
	for _, s := range model.RecorderSetup() {
		if p == s.Parameter {
			valid = v == s.Value
		}
	}
	if !valid {
		return fmt.Errorf("unsupported recorder setting %s=%d", p, v)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.refresh(); e != nil {
		return e
	}
	edition, code := c.api.Edition()
	if e := status("recorder edition", code); e != nil {
		return e
	}
	if edition != 3 {
		return fmt.Errorf("recorder profile requires Potato")
	}
	return status("set "+p, c.api.SetNumber(p, v))
}
