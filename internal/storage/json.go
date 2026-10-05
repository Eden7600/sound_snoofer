// Package storage provides strict JSON decoding and atomic file replacement.
package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Decode rejects duplicate and unknown fields and multiple documents.
func Decode(data []byte, value any) error {
	if err := UniqueKeys(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return fmt.Errorf("expected exactly one JSON object")
	}
	return nil
}

// UniqueKeys rejects ambiguous JSON objects, including nested plugin settings.
func UniqueKeys(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("invalid JSON key")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON key %q", name)
			}
			seen[name] = true
			if err := UniqueKeys(d); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := UniqueKeys(d); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
	_, err = d.Token()
	return err
}
