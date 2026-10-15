package config

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodePreservesJSONErrorContext(t *testing.T) {
	_, err := Decode([]byte(`{"version":"one"}`))
	var mismatch *json.UnmarshalTypeError
	if !errors.As(err, &mismatch) || !strings.HasPrefix(err.Error(), "configuration: ") {
		t.Fatalf("type error lost its cause or context: %v", err)
	}
	_, err = Decode([]byte(`{"version":}`))
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) || !strings.HasPrefix(err.Error(), "configuration: ") {
		t.Fatalf("syntax error lost its cause or context: %v", err)
	}
}

func TestLoadChoicesRetainsIntentOnFailure(t *testing.T) {
	c, err := Decode([]byte(voiceJSON))
	if err != nil {
		t.Fatal(err)
	}
	c.Intent = c.VoiceIntent()
	path := filepath.Join(t.TempDir(), "audio")
	for _, input := range []string{
		`{"source":"lav",`,
		`{"source":"lav","unknown":true}`,
		`{"source":"lav","source":"desk"}`,
		`{"source":"lav"} {}`,
		`{"version":2,"source":"lav"}`,
	} {
		if err := os.WriteFile(path+".state.json", []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		loaded := LoadChoices(path, c)
		if !strings.HasPrefix(loaded.StateError, "Saved choices: ") || loaded.Intent != c.Intent {
			t.Fatalf("failed choices replaced intent or lost error: %#v", loaded)
		}
		if want := fmt.Sprintf("%x", sha256.Sum256([]byte(input))); loaded.StateToken != want {
			t.Fatalf("failed choices token = %q; want %q", loaded.StateToken, want)
		}
	}
}
