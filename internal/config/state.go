package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sound-snoofer/internal/ownership"
)

func stateBytes(path string) ([]byte, string, error) {
	b, e := os.ReadFile(path + ".state.json")
	if os.IsNotExist(e) {
		return nil, "missing", nil
	}
	if e != nil {
		return nil, "", e
	}
	return b, fmt.Sprintf("%x", sha256.Sum256(b)), nil
}

// State errors are retained so the interactive UI can offer an explicit reset.
// Noninteractive commands must reject StateError before accessing the mixer.
func LoadEffective(path string) (Config, error) {
	c, e := Load(path)
	if e != nil {
		return c, e
	}
	return LoadChoices(path, c), nil
}

// LoadChoices reads saved choices without requiring a second configuration file.
func LoadChoices(path string, c Config) Config {
	if c.VoiceIntent() == nil {
		return c
	}
	b, token, e := stateBytes(path)
	c.StateToken = token
	if e == nil && b != nil {
		e = uniqueKeys(json.NewDecoder(bytes.NewReader(b)))
		if e == nil {
			var i Intent
			d := json.NewDecoder(bytes.NewReader(b))
			d.DisallowUnknownFields()
			e = d.Decode(&i)
			if e == nil {
				var extra any
				if d.Decode(&extra) != io.EOF {
					e = fmt.Errorf("saved choices must contain one object")
				}
			}
			if e == nil {
				i.NormalizeRehearsal()
				e = i.Validate(c)
			}
			if e == nil {
				c.Intent = &i
			}
		}
	}
	if e != nil {
		c.StateError = "Saved choices: " + e.Error()
	}
	return c
}
func SaveIntent(path string, c Config, i *Intent, expected string) (string, error) {
	if i == nil {
		return "", fmt.Errorf("voice profile is not configured")
	}
	i = i.Clone()
	i.NormalizeRehearsal()
	if e := i.Validate(c); e != nil {
		return "", e
	}
	absolute, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	release, e := ownership.AcquireState(absolute)
	if e != nil {
		return "", e
	}
	defer release()
	_, token, e := stateBytes(path)
	if e != nil {
		return "", e
	}
	if token != expected {
		return "", fmt.Errorf("saved choices changed externally; reload before editing")
	}
	b, e := json.MarshalIndent(i, "", "  ")
	if e != nil {
		return "", e
	}
	b = append(b, '\n')
	if e = replaceBytes(absolute+".state.json", b); e != nil {
		return "", e
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
