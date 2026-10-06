package mediasessions

import "testing"

func TestParseSnapshot(t *testing.T) {
	sessions, err := parseSnapshot([]byte(`[{"id":"Brave","app":"Brave","title":"Grace","artist":"Calum Graham","album":"","status":"playing","positionMs":3279,"durationMs":405258,"updatedMs":1759780000000,"rate":1.000000,"canPlay":false,"canPause":true,"canNext":true,"canPrev":true,"canSeek":true,"current":true,"artKey":"00ff"},` +
		`{"id":"Spotify.exe#1","app":"Spotify.exe","title":"Line \"two\"\n","artist":"","album":"","status":"paused","positionMs":0,"durationMs":0,"updatedMs":0,"rate":1,"canPlay":true,"canPause":false,"canNext":false,"canPrev":false,"canSeek":false,"current":false,"artKey":""}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 || sessions[0].Title != "Grace" || !sessions[0].Current || sessions[0].DurationMs != 405258 || sessions[1].Title != "Line \"two\"\n" || sessions[1].CanSeek {
		t.Fatalf("%+v", sessions)
	}
	if _, err := parseSnapshot([]byte(`[{`)); err == nil {
		t.Fatal("truncated snapshot accepted")
	}
}
