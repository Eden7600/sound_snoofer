package browserbridge

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const validCalls = `{"type":"meet-state","calls":[{"tab":7,"instance":"42","sample":"29","call":"3","state":"joined","mic":{"value":true,"revision":"8","available":true,"reason":""},"camera":{"value":false,"revision":"5","available":true,"reason":""}}]}`
const validResult = `{"type":"meet-result","id":"106","tab":7,"instance":"42","call":"3","control":"mic","revision":"8","value":false,"status":"observed","reason":""}`

func TestMeetDecoderPreservesFalseAndUnknown(t *testing.T) {
	calls, err := decodeCalls([]byte(validCalls))
	if err != nil || len(calls) != 1 || calls[0].Camera.Value == nil || *calls[0].Camera.Value {
		t.Fatalf("false camera lost: %+v, %v", calls, err)
	}
	unknown := strings.Replace(validCalls, `"value":false,"revision":"5","available":true,"reason":""`, `"value":null,"revision":"5","available":false,"reason":"unknown"`, 1)
	calls, err = decodeCalls([]byte(unknown))
	if err != nil || calls[0].Camera.Value != nil || calls[0].Camera.Available {
		t.Fatalf("unknown camera lost: %+v, %v", calls, err)
	}
	result, err := decodeResult([]byte(validResult))
	if err != nil || result.Value || result.Status != "observed" {
		t.Fatalf("false result lost: %+v, %v", result, err)
	}
	if _, err := decodeCalls([]byte(`{"calls":[]}`)); err != nil {
		t.Fatal("empty full list rejected", err)
	}
}

func TestMeetDecoderRejectsMissingNullAndMalformedFields(t *testing.T) {
	for _, mutation := range []struct{ from, to string }{
		{`"tab":7,`, ""}, {`"tab":7`, `"tab":null`}, {`"tab":7`, `"tab":1.2`},
		{`"tab":7`, `"tab":9007199254740992`}, {`"value":false,`, ""},
		{`"value":false`, `"value":0`}, {`"available":true`, `"available":null`},
		{`"reason":""`, `"reason":null`}, {`"call":"3"`, `"call":null`},
		{`"state":"joined"`, `"state":"prejoin"`}, {`"sample":"29",`, ""},
		{`"revision":"8",`, ""}, {`"value":true`, `"value":null`},
	} {
		t.Run(mutation.to+mutation.from, func(t *testing.T) {
			if _, err := decodeCalls([]byte(strings.Replace(validCalls, mutation.from, mutation.to, 1))); err == nil {
				t.Fatal("invalid state accepted")
			}
		})
	}
	for _, raw := range []string{`{}`, `{"calls":null}`, `{"calls":{}}`, `{"calls":[null]}`, validCalls + strings.Repeat(" ", 64*1024)} {
		if _, err := decodeCalls([]byte(raw)); err == nil {
			t.Fatalf("invalid state accepted: %.80s", raw)
		}
	}
	for _, mutation := range []struct{ from, to string }{
		{`"value":false,`, ""}, {`"value":false`, `"value":null`},
		{`"tab":7`, `"tab":null`}, {`"reason":""`, `"reason":null`},
		{`"status":"observed"`, `"status":"sent"`}, {`"reason":""`, `"reason":"stale"`},
	} {
		if _, err := decodeResult([]byte(strings.Replace(validResult, mutation.from, mutation.to, 1))); err == nil {
			t.Fatalf("invalid result accepted: %+v", mutation)
		}
	}
}

func TestMeetDecoderRequiresCanonicalUint64Counters(t *testing.T) {
	for _, value := range []string{"0", "01", "+1", "-1", "1.0", " 1", "18446744073709551616", ""} {
		for _, field := range []string{`"sample":"29"`, `"revision":"8"`, `"instance":"42"`, `"call":"3"`} {
			key := strings.SplitN(field, ":", 2)[0]
			if _, err := decodeCalls([]byte(strings.Replace(validCalls, field, key+`:"`+value+`"`, 1))); err == nil {
				t.Fatalf("state accepted noncanonical %s=%q", key, value)
			}
		}
		for _, field := range []string{`"revision":"8"`, `"id":"106"`} {
			key := strings.SplitN(field, ":", 2)[0]
			if _, err := decodeResult([]byte(strings.Replace(validResult, field, key+`:"`+value+`"`, 1))); err == nil {
				t.Fatalf("result accepted noncanonical %s=%q", key, value)
			}
		}
	}
	maximum := strings.ReplaceAll(strings.Replace(validCalls, `"sample":"29"`, `"sample":"18446744073709551615"`, 1), `"revision":"8"`, `"revision":"18446744073709551615"`)
	if _, err := decodeCalls([]byte(maximum)); err != nil {
		t.Fatal("valid uint64 maximum rejected", err)
	}
}

func TestMeetDecoderBoundsAndDuplicateTabs(t *testing.T) {
	var state struct {
		Calls []json.RawMessage `json:"calls"`
	}
	if err := json.Unmarshal([]byte(validCalls), &state); err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{2, 65} {
		calls := make([]json.RawMessage, count)
		for n := range calls {
			calls[n] = state.Calls[0]
		}
		raw, _ := json.Marshal(struct {
			Calls []json.RawMessage `json:"calls"`
		}{calls})
		if _, err := decodeCalls(raw); err == nil {
			t.Fatalf("accepted %d duplicate tabs", count)
		}
	}
	if _, err := decodeResult([]byte(validResult + strings.Repeat(" ", 2048))); err == nil {
		t.Fatal("oversized result accepted")
	}
}

func TestLegacyPortExtraction(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		port    int
		invalid bool
	}{
		{"", 47815, false}, {`{}`, 47815, false}, {`{"port":0,"unknown":{}}`, 47815, false},
		{`{"port":1024}`, 1024, false}, {`{"port":65535}`, 65535, false},
		{`{"port":null}`, 47815, true}, {`{"port":"47815"}`, 47815, true},
		{`{"port":47815.1}`, 47815, true}, {`{"port":1}`, 1, true}, {`{"port":65536}`, 65536, true},
	} {
		b := New(context.Background(), json.RawMessage(tc.raw))
		if status := b.Status(); status.Port != tc.port || (status.Error != "") != tc.invalid {
			t.Errorf("%s: %+v", tc.raw, status)
		}
		if b.run != nil {
			t.Fatal("constructor acquired resources")
		}
	}
}
