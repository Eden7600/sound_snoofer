package config

import (
	"strings"
	"testing"
)

const valid = `{"version":1,"routes":[{"target":"input:1","candidates":[{"driver":"wdm","pattern":"(?i)volt.*2"}]}]}`

func TestDecode(t *testing.T) {
	c, e := Decode([]byte(valid))
	if e != nil {
		t.Fatal(e)
	}
	if c.PollMS != 1000 || c.DebounceMS != 1000 || c.VerifyMS != 5000 || !c.Routes[0].Candidates[0].Regex.MatchString("INPUT 1/2 (Volt 2)") {
		t.Fatal(c)
	}
	cases := map[string]string{
		"bad regex":          strings.Replace(valid, "(?i)volt.*2", "[", 1),
		"unknown field":      strings.Replace(valid, `"version":1`, `"version":1,"typo":true`, 1),
		"unknown nested":     strings.Replace(valid, `"driver":"wdm"`, `"driver":"wdm","typo":true`, 1),
		"duplicate JSON key": strings.Replace(valid, `"version":1`, `"version":1,"version":1`, 1),
		"driver":             strings.Replace(valid, "wdm", "asio", 1),
		"empty regex":        strings.Replace(valid, "(?i)volt.*2", "", 1),
		"target":             strings.Replace(valid, "input:1", "B1", 1),
		"alias":              strings.Replace(valid, "input:1", "input:01", 1),
		"poll":               strings.Replace(valid, `"version":1`, `"version":1,"poll_ms":0`, 1),
		"debounce":           strings.Replace(valid, `"version":1`, `"version":1,"debounce_ms":-1`, 1),
		"timeout":            strings.Replace(valid, `"version":1`, `"version":1,"verify_ms":60001`, 1),
		"trailing":           valid + `{}`, "null": `null`, "empty routes": `{"version":1,"routes":[]}`,
		"duplicate target": `{"version":1,"routes":[{"target":"A1","candidates":[{"driver":"wdm","pattern":"x"}]},{"target":"A1","candidates":[{"driver":"wdm","pattern":"y"}]}]}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, e := Decode([]byte(input)); e == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}
func TestEdition(t *testing.T) {
	c, e := Decode([]byte(strings.Replace(valid, "input:1", "A4", 1)))
	if e != nil {
		t.Fatal(e)
	}
	if c.ValidateEdition(2) == nil {
		t.Fatal("Banana accepted A4")
	}
	if e = c.ValidateEdition(3); e != nil {
		t.Fatal(e)
	}
	if c.ValidateEdition(1) == nil {
		t.Fatal("accepted Standard")
	}
}
