package windowsaudio

import (
	"fmt"
	"testing"
	"time"
)

type fake struct {
	roles  [2][3]string
	writes int
}

func (f *fake) Endpoints(int) ([]Endpoint, error) { return nil, nil }
func (f *fake) Default(d, r int) (string, error)  { return f.roles[d][r], nil }
func (f *fake) Set(id string, r int) error {
	d := 0
	if id == "capture" {
		d = 1
	}
	f.roles[d][r] = id
	f.writes++
	return nil
}
func (f *fake) Close() {}
func TestProtectionRolesPreviewAndContention(t *testing.T) {
	f := &fake{}
	g := Guard{}
	now := time.Now()
	targets := [2]string{"playback", "capture"}
	r := Request{Enabled: true}
	g.Reconcile(f, r, targets, now)
	if f.writes != 0 {
		t.Fatal("preview wrote defaults")
	}
	r.Live = true
	if s := g.Reconcile(f, r, targets, now); s.Kind != Verified || f.writes != 6 {
		t.Fatal(s, f.writes)
	}
	g.Reconcile(f, r, targets, now)
	if f.writes != 6 {
		t.Fatal("rewrote matching roles")
	}
	for n := 1; n <= 3; n++ {
		f.roles[0][0] = fmt.Sprint(n)
		g.Reconcile(f, r, targets, now.Add(time.Duration(n)*time.Second))
	}
	if !g.Suspended[0] || g.Suspended[1] {
		t.Fatal("contention not isolated")
	}
	g.Reconcile(f, Request{}, targets, now)
	if g.Suspended[0] {
		t.Fatal("off did not reset")
	}
}
