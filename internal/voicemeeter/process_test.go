package voicemeeter

import (
	"errors"
	"testing"
)

func TestElementProcessObservation(t *testing.T) {
	for _, tc := range []struct {
		names []string
		want  bool
	}{{[]string{"element.exe"}, true}, {[]string{"ElEmEnT.ExE", "element.exe"}, true}, {[]string{"element-helper.exe", "myelement.exe"}, false}, {nil, false}} {
		s := observeElement(func() ([]string, error) { return tc.names, nil })
		if !s.Known || s.Running != tc.want || s.Error != "" {
			t.Fatal(s)
		}
	}
	s := observeElement(func() ([]string, error) { return nil, errors.New("denied") })
	if s.Known || s.Running || s.Error == "" {
		t.Fatal("query failure treated as running", s)
	}
}
