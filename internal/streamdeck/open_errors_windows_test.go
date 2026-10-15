//go:build windows

package streamdeck

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOpenErrorsArePerPathAndRetire(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type scan struct {
		paths    []string
		failures map[string]string
		err      error
	}
	scans := make(chan scan)
	var current scan // The discovery goroutine alone reads and writes this.
	surface := startSurface(ctx, func() ([]string, error) {
		select {
		case current = <-scans:
			return current.paths, current.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, func(path string) (hidDeck, error) {
		if message := current.failures[path]; message != "" {
			return nil, errors.New(message)
		}
		serial := path
		if path == "anonymous" {
			serial = ""
		}
		return newFakeDeck(serial, "plus"), nil
	})
	defer func() {
		cancel()
		select {
		case <-surface.Done():
		case <-time.After(time.Second):
			t.Error("surface did not stop")
		}
	}()
	advance := func(s scan) {
		t.Helper()
		select {
		case scans <- s:
		case <-time.After(3 * time.Second):
			t.Fatal("discovery did not scan again")
		}
	}
	// Compare every event in each batch, including unexpected duplicates/retirements.
	want := func(expected map[string]Event) {
		t.Helper()
		for len(expected) > 0 {
			e := next(t, surface.Events(), func(Event) bool { return true })
			kind := "error"
			if e.Connected {
				kind = "connected"
			}
			if e.Retired {
				kind = "retired"
			}
			key := e.Device + ":" + kind
			w, ok := expected[key]
			if !ok || e.Error != w.Error || e.Serial != w.Serial {
				t.Fatalf("unexpected event %+v; remaining %+v", e, expected)
			}
			delete(expected, key)
		}
	}
	failed := scan{paths: []string{"working", "other", "anonymous"}, failures: map[string]string{"other": "access denied", "anonymous": "busy"}}
	advance(failed)
	want(map[string]Event{
		"working:connected":    {Serial: "working"},
		"path:other:error":     {Error: "access denied"},
		"path:anonymous:error": {Error: "busy"},
	})
	advance(failed)                                      // Identical failures are not repeated.
	advance(scan{err: errors.New("enumeration failed")}) // Not evidence that paths disappeared.
	advance(scan{paths: []string{"working", "other"}, failures: map[string]string{"other": "still busy"}})
	want(map[string]Event{
		"path:anonymous:retired": {},
		"path:other:error":       {Error: "still busy"},
	})
	advance(scan{paths: []string{"working", "other", "anonymous"}, failures: map[string]string{"anonymous": "busy"}})
	want(map[string]Event{
		"path:other:retired":   {},
		"other:connected":      {Serial: "other"},
		"path:anonymous:error": {Error: "busy"}, // Same error returns after disappearance.
	})
	advance(scan{paths: []string{"working", "other", "anonymous"}})
	// Without a serial, retirement must precede recreation of the same path identity.
	if e := next(t, surface.Events(), func(Event) bool { return true }); e.Device != "path:anonymous" || !e.Retired {
		t.Fatal("anonymous path was not retired before recovery", e)
	}
	want(map[string]Event{"path:anonymous:connected": {}})
}
