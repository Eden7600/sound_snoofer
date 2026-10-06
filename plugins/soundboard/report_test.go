package soundboard

import (
	"errors"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestPlaybackReport(t *testing.T) {
	var link snoofer.ConnectionTracker
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	renderer := "DirectSound: Voicemeeter VAIO3 Input (VB-Audio Voicemeeter VAIO)"
	r := playbackReport(&link, true, nil, false, renderer, "", now)
	if r.Connection.State != snoofer.ConnectionReady || r.Connection.Endpoint != renderer || r.Kind != "connection" || len(r.Operations) != 0 {
		t.Fatalf("ready %+v %+v", r, r.Connection)
	}
	link.Fail("soundboard native error 0x80040218", now)
	r = playbackReport(&link, true, errors.New("soundboard routing not verified"), false, renderer, "", now.Add(time.Second))
	if r.Connection.State != snoofer.ConnectionAttention || r.Connection.Details[1].Value != "soundboard routing not verified" || r.Connection.LastError == "" {
		t.Fatalf("not ready %+v", r.Connection)
	}
	link.Activity(now.Add(2 * time.Second))
	r = playbackReport(&link, true, nil, true, renderer, "airhorn", now.Add(2*time.Second))
	if r.Value != "Loaded" || r.Connection.Details[2].Value != "airhorn" || r.Connection.LastActivity.IsZero() {
		t.Fatalf("loaded %+v", r.Connection)
	}
	if r = playbackReport(&link, false, nil, true, renderer, "", now); r.Connection.State != snoofer.ConnectionOff {
		t.Fatalf("preview %+v", r.Connection)
	}
}
