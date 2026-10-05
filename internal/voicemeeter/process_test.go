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
		s := observeProcess(tc.names, nil, "element.exe")
		if !s.Known || s.Running != tc.want || s.Error != "" {
			t.Fatal(s)
		}
	}
	s := observeProcess(nil, errors.New("denied"), "element.exe")
	if s.Known || s.Running || s.Error == "" {
		t.Fatal("query failure treated as running", s)
	}
}
