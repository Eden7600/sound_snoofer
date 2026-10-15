package discord

import (
	"context"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func attachMeetingFixture(t *testing.T, w *worker, ctx context.Context, mic *fakeMic) *snoofer.Meetings {
	t.Helper()
	m := snoofer.NewMeetings(w.services.Controls, w.services.Live)
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	w.services.Meetings = m
	adapter, err := m.Attach(ctx, "discord")
	if err != nil {
		t.Fatal(err)
	}
	w.meetings = adapter
	var source *snoofer.MeetingAudio
	if mic != nil {
		source, err = m.AttachAudio(ctx, func(value bool, revision uint64, complete snoofer.Completion) error {
			return mic.SetMicMuteWithResult(value, "meetings", revision, func(outcome snoofer.Outcome) {
				pref := mic.MicMute()
				source.Report(snoofer.MeetingMicPreference{Known: pref.Known, Muted: pref.Muted, Applied: pref.Applied, Revision: pref.Revision})
				complete(outcome)
			})
		})
		if err != nil {
			t.Fatal(err)
		}
		pref := mic.MicMute()
		source.Report(snoofer.MeetingMicPreference{Known: pref.Known, Muted: pref.Muted, Applied: pref.Applied, Revision: pref.Revision})
	}
	t.Cleanup(func() {
		adapter.Close()
		if source != nil {
			source.Close()
		}
		stop, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := m.Close(stop); err != nil {
			t.Error(err)
		}
	})
	return m
}
