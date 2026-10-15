package appaudio

import (
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestStatusAttentionUsesBackendAndEditErrors(t *testing.T) {
	for _, live := range []bool{false, true} {
		for _, field := range []string{"none", "backend", "edit"} {
			registry := snoofer.NewControls()
			w := &worker{services: snoofer.Services{Controls: registry, Live: live}}
			if field == "backend" {
				w.backendErr = "Running"
			}
			if field == "edit" {
				w.editErr = "Off"
			}
			check := func(want snoofer.AttentionLevel) {
				t.Helper()
				w.publish(nil, time.Unix(1, 0))
				for _, item := range registry.Snapshot() {
					if item.ID == "appaudio.status" {
						if item.Attention != want {
							t.Fatalf("live=%v %s: %q, want %q", live, field, item.Attention, want)
						}
						return
					}
				}
				t.Fatal("missing status")
			}
			want := snoofer.AttentionLevel("")
			if field != "none" {
				want = snoofer.AttentionCritical
			}
			check(want)
			w.backendErr, w.editErr = "", ""
			check("")
		}
	}
}
