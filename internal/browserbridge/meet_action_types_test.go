package browserbridge

import (
	"context"
	"strings"
	"testing"
)

func withCallAction(control, value string) string {
	return strings.Replace(validCalls, `"state":"joined",`, `"state":"joined","`+control+`":`+value+`,`, 1)
}

func TestMeetOptionalCallActionsValidateWithoutChangingOldHelpers(t *testing.T) {
	old, err := decodeCalls([]byte(validCalls))
	if err != nil || old[0].Share != nil || old[0].Leave != nil {
		t.Fatal("old helper action capabilities changed", old, err)
	}
	for _, control := range []string{"share", "leave"} {
		valid := `{"identity":"2","value":false,"revision":"9","available":true,"reason":""}`
		calls, err := decodeCalls([]byte(withCallAction(control, valid)))
		if err != nil {
			t.Fatal(control, err)
		}
		state := calls[0].Share
		if control == "leave" {
			state = calls[0].Leave
		}
		if state == nil || state.Value == nil || *state.Value || state.Revision != 9 || state.Identity != 2 {
			t.Fatal("known false lost", control, state)
		}
		for _, value := range []string{
			`null`, `{}`, `false`,
			strings.Replace(valid, `"value":false,`, "", 1),
			strings.Replace(valid, `"value":false`, `"value":null`, 1),
			strings.Replace(valid, `"revision":"9"`, `"revision":"0"`, 1),
			strings.Replace(valid, `"identity":"2"`, `"identity":"18446744073709551616"`, 1),
			strings.Replace(valid, `"available":true`, `"available":null`, 1),
		} {
			if _, err := decodeCalls([]byte(withCallAction(control, value))); err == nil {
				t.Fatal("malformed action accepted", control, value)
			}
		}
		prejoin := strings.Replace(strings.Replace(withCallAction(control, valid), `"state":"joined"`, `"state":"prejoin"`, 1), `"call":"3"`, `"call":""`, 1)
		prejoin = strings.ReplaceAll(prejoin, `"available":true,"reason":""`, `"available":false,"reason":"not-in-call"`)
		prejoin = strings.Replace(prejoin, `"revision":"9","available":false,"reason":"not-in-call"`, `"revision":"9","available":true,"reason":""`, 1)
		if _, err := decodeCalls([]byte(prejoin)); err == nil {
			t.Fatal("prejoin action accepted", control)
		}
	}
}

func TestMeetShareSubmissionAndLeaveCommandBoundaries(t *testing.T) {
	for _, control := range []string{"mic", "camera", "share", "leave", "deafen"} {
		for _, value := range []bool{false, true} {
			c := MeetCommand{Generation: 1, Tab: 7, Instance: "42", Call: "3", Control: control, Revision: 8, Value: value}
			want := control != "deafen" && (control != "leave" || value)
			if validCommand(c) != want {
				t.Fatal("command validity", c)
			}
			raw := strings.Replace(validResult, `"control":"mic"`, `"control":"`+control+`"`, 1)
			if value {
				raw = strings.Replace(raw, `"value":false`, `"value":true`, 1)
			}
			raw = strings.Replace(strings.Replace(raw, `"status":"observed"`, `"status":"submitted"`, 1), `"reason":""`, `"reason":"choose-source"`, 1)
			if _, err := decodeResult([]byte(raw)); (err == nil) != (control == "share" && value) {
				t.Fatal("invalid submission outcome", raw, err)
			}
		}
	}
}

func TestMeetCallActionsRetainExactResultCorrelation(t *testing.T) {
	b := newTestBridge(t)
	meet := meetLease(t, b)
	ws := connect(t, b, "Browser", 1)
	var caps capabilities
	readJSON(t, ws, &caps)
	receive(t, meet.Updates())
	for _, control := range []string{"share", "leave"} {
		command := captured(meet, "Browser", 7)
		command.Control, command.Value = control, true
		results, err := meet.Send(context.Background(), "Browser", command)
		if err != nil {
			t.Fatal(err)
		}
		var sent wireCommand
		readJSON(t, ws, &sent)
		reply(t, ws, sent, func(r *MeetResult) { r.Revision++ })
		writeJSON(t, ws, map[string]any{"type": "meet-state", "calls": []MeetCall{}})
		receive(t, meet.Updates())
		assertNoResult(t, results)
		reply(t, ws, sent, func(r *MeetResult) {
			if control == "share" {
				r.Status, r.Reason = "submitted", "choose-source"
			}
		})
		result := receive(t, results)
		want := "observed"
		if control == "share" {
			want = "submitted"
		}
		if result.Status != want || result.Control != control || result.Revision != command.Revision {
			t.Fatal(result)
		}
	}
}
