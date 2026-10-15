package storage

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	type entry struct {
		Name string `json:"name"`
	}
	type document struct {
		Name  string  `json:"name"`
		Count int     `json:"count"`
		Child *entry  `json:"child"`
		Items []entry `json:"items"`
	}
	defaults := document{Name: "default", Count: 7}
	cases := []struct {
		name    string
		input   string
		want    document
		invalid bool
	}{
		{name: "valid", input: `{"name":"saved","count":2}`, want: document{Name: "saved", Count: 2}},
		{name: "defaults", input: `{}`, want: defaults},
		{name: "null preserves defaults", input: `null`, want: defaults},
		{name: "sibling names", input: `{"child":{"name":"first"},"items":[{"name":"second"},{"name":"third"}]}`, want: document{Name: "default", Count: 7, Child: &entry{Name: "first"}, Items: []entry{{Name: "second"}, {Name: "third"}}}},
		{name: "duplicate top level", input: `{"name":"a","name":"b"}`, invalid: true},
		{name: "duplicate nested", input: `{"child":{"name":"a","name":"b"}}`, invalid: true},
		{name: "duplicate in array", input: `{"items":[{"name":"a","name":"b"}]}`, invalid: true},
		{name: "unknown top level", input: `{"extra":1}`, invalid: true},
		{name: "unknown nested", input: `{"child":{"extra":1}}`, invalid: true},
		{name: "malformed", input: `{"name":}`, invalid: true},
		{name: "empty", input: ``, invalid: true},
		{name: "type mismatch", input: `{"count":"two"}`, invalid: true},
		{name: "second object", input: `{} {}`, invalid: true},
		{name: "trailing scalar", input: `{} 1`, invalid: true},
		{name: "trailing junk", input: `{} junk`, invalid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := defaults
			err := Decode([]byte(tc.input), &got)
			if tc.invalid {
				if err == nil {
					t.Fatal("accepted invalid JSON")
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Decode = %#v, %v; want %#v", got, err, tc.want)
			}
		})
	}
	var got document
	var mismatch *json.UnmarshalTypeError
	if err := Decode([]byte(`{"count":"two"}`), &got); !errors.As(err, &mismatch) {
		t.Fatalf("type error lost: %v", err)
	}
	var syntax *json.SyntaxError
	if err := Decode([]byte(`{"name":}`), &got); !errors.As(err, &syntax) {
		t.Fatalf("syntax error lost: %v", err)
	}
}
