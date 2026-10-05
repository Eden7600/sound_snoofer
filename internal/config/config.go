package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sound-snoofer/internal/model"
	"time"
)

type Candidate struct {
	Driver  string         `json:"driver"`
	Pattern string         `json:"pattern"`
	Regex   *regexp.Regexp `json:"-"`
}
type Route struct {
	Target     string      `json:"target"`
	Candidates []Candidate `json:"candidates"`
}
type Config struct {
	StreamDeck *StreamDeck `json:"stream_deck,omitempty"`
	VR         *VR         `json:"vr,omitempty"`
	Intent     *Intent     `json:"-"`
	StateError string      `json:"-"`
	StateToken string      `json:"-"`
	Version    int         `json:"version"`
	PollMS     int         `json:"poll_ms"`
	DebounceMS int         `json:"debounce_ms"`
	VerifyMS   int         `json:"verify_ms"`
	Routes     []Route     `json:"routes"`
	Studio     *Studio     `json:"studio,omitempty"`
}

func Load(path string) (Config, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return Config{}, e
	}
	return Decode(b)
}
func Decode(b []byte) (Config, error) {
	c := Config{PollMS: 1000, DebounceMS: 1000, VerifyMS: 5000}
	// Reject duplicate keys as well as unknown fields; ambiguous config is never applied.
	if err := uniqueKeys(json.NewDecoder(bytes.NewReader(b))); err != nil {
		return c, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, fmt.Errorf("configuration: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("configuration must contain exactly one JSON object")
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

func uniqueKeys(d *json.Decoder) error {
	t, e := d.Token()
	if e != nil {
		return e
	}
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				s := k.(string)
				if seen[s] {
					return fmt.Errorf("duplicate JSON key %q", s)
				}
				seen[s] = true
				if e = uniqueKeys(d); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = uniqueKeys(d); e != nil {
					return e
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		_, e = d.Token()
		return e
	}
	return nil
}

func (c *Config) Validate() error {
	if c.StreamDeck != nil {
		if err := c.StreamDeck.Validate(); err != nil {
			return err
		}
	}
	if c.VR != nil {
		if err := c.VR.Validate(); err != nil {
			return err
		}
		if c.Studio == nil || c.Studio.Voice == nil {
			return fmt.Errorf("VR requires studio voice profile")
		}
	}
	if c.Version != 1 {
		return fmt.Errorf("configuration version must be 1")
	}
	if c.PollMS < 100 || c.PollMS > 60000 || c.DebounceMS < 100 || c.DebounceMS > 60000 || c.VerifyMS < 100 || c.VerifyMS > 60000 {
		return fmt.Errorf("poll_ms, debounce_ms and verify_ms must each be between 100 and 60000")
	}
	if c.Studio != nil {
		if len(c.Routes) != 0 {
			return fmt.Errorf("studio and fixed routes cannot be combined")
		}
		return c.Studio.Validate()
	}
	if len(c.Routes) == 0 {
		return fmt.Errorf("at least one managed route is required")
	}
	seen := map[string]bool{}
	for i := range c.Routes {
		r := &c.Routes[i]
		if _, e := model.ParseSlot(r.Target); e != nil {
			return e
		}
		if seen[r.Target] {
			return fmt.Errorf("duplicate target %q", r.Target)
		}
		seen[r.Target] = true
		if len(r.Candidates) == 0 {
			return fmt.Errorf("%s: candidates must not be empty", r.Target)
		}
		for j := range r.Candidates {
			p := &r.Candidates[j]
			if p.Driver != "wdm" {
				return fmt.Errorf("%s candidate %d: only wdm assignments are supported", r.Target, j+1)
			}
			if p.Pattern == "" {
				return fmt.Errorf("%s candidate %d: pattern must not be empty", r.Target, j+1)
			}
			re, e := regexp.Compile(p.Pattern)
			if e != nil {
				return fmt.Errorf("%s candidate %d regex: %w", r.Target, j+1, e)
			}
			p.Regex = re
		}
	}
	return nil
}
func (c Config) ValidateEdition(edition int) error {
	if c.Studio != nil && c.Studio.Voice != nil && edition != 3 {
		return fmt.Errorf("voice profile requires Potato")
	}
	n, e := model.Limits(edition)
	if e != nil {
		return e
	}
	if c.Studio != nil && edition == 2 {
		for _, source := range c.Studio.PlaybackSources {
			if source == "virtual:3" {
				return fmt.Errorf("virtual:3 requires Potato")
			}
		}
	}
	for _, r := range c.Routes {
		s, e := model.ParseSlot(r.Target)
		if e != nil {
			return e
		}
		if s.Index >= n {
			return fmt.Errorf("target %s is unavailable in edition %d", r.Target, edition)
		}
	}
	return nil
}
func (c Config) Poll() time.Duration     { return time.Duration(c.PollMS) * time.Millisecond }
func (c Config) Debounce() time.Duration { return time.Duration(c.DebounceMS) * time.Millisecond }
func (c Config) Verify() time.Duration   { return time.Duration(c.VerifyMS) * time.Millisecond }
