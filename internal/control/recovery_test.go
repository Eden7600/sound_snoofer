package control

import (
	"fmt"
	"path/filepath"
	"sound-snoofer/internal/model"
	"testing"
	"time"
)

type restarter struct {
	calls int
	fail  bool
}

func (r *restarter) RestartEngine() error {
	r.calls++
	if r.fail {
		return fmt.Errorf("unknown outcome")
	}
	return nil
}
func TestRecoveryDoesNotReplayOrClaimAudibility(t *testing.T) {
	r := newRecovery(filepath.Join(t.TempDir(), "config"))
	b := &restarter{}
	now := time.Now()
	if r.restart(b, false, now) == nil || b.calls != 0 {
		t.Fatal("preview restart")
	}
	if e := r.restart(b, true, now); e != nil {
		t.Fatal(e)
	}
	if r.restart(b, true, now) == nil || b.calls != 1 {
		t.Fatal("duplicate restart")
	}
	s := model.Snapshot{Numbers: map[string]float32{"Bus[0].device.sr": 48000}}
	r.observe(s, now.Add(2*time.Second))
	status := r.observe(s, now.Add(3*time.Second))
	if r.pending || status != "Engine responding; audio continuity unverified" {
		t.Fatal(status)
	}
	if len(newRecovery(r.path[:len(r.path)-len(".recovery.json")]).attempts) != 1 {
		t.Fatal("attempt not persisted")
	}
}
func TestRecoveryUnknownAndTimeout(t *testing.T) {
	r := newRecovery(filepath.Join(t.TempDir(), "config"))
	b := &restarter{fail: true}
	now := time.Now()
	if r.restart(b, true, now) == nil || r.pending {
		t.Fatal("ambiguous restart")
	}
	for n := 0; n < 5; n++ {
		r.observe(model.Snapshot{}, now)
	}
	if b.calls != 1 {
		t.Fatal("replayed")
	}
	b.fail = false
	r.restart(b, true, now)
	r.observe(model.Snapshot{}, now.Add(11*time.Second))
	if r.pending {
		t.Fatal("verification unbounded")
	}
}

func TestAutomaticRecoveryBudgetAndEvidence(t *testing.T) {
	now := time.Now()
	r := &recovery{}
	if r.automaticPermit(now, false, "Stopped") == nil {
		t.Fatal("silence accepted as evidence")
	}
	if r.automaticPermit(now, true, "Unknown") == nil {
		t.Fatal("unknown transport allowed")
	}
	r.attempts = []time.Time{now.Add(-30 * time.Second)}
	if r.automaticPermit(now, true, "Stopped") == nil {
		t.Fatal("cooldown ignored")
	}
	r.attempts = []time.Time{now.Add(-2 * time.Minute), now.Add(-3 * time.Minute)}
	if r.automaticPermit(now, true, "Stopped") == nil {
		t.Fatal("budget ignored")
	}
	r.attempts = []time.Time{now.Add(time.Second)}
	if r.automaticPermit(now, true, "Stopped") == nil {
		t.Fatal("backwards clock ignored")
	}
	r.attempts = []time.Time{now.Add(-11 * time.Minute)}
	if e := r.automaticPermit(now, true, "Stopped"); e != nil {
		t.Fatal(e)
	}
}
