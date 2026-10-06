package soundboard

import (
	"errors"
	"testing"
	"time"
)

type fakePlayer struct {
	id      int
	playing string
	active  bool
	stops   int
	closed  bool
	pollErr error
}

func (f *fakePlayer) Play(path string, _ bool) error { f.playing, f.active = path, true; return nil }
func (f *fakePlayer) Stop() error                    { f.active = false; f.stops++; return nil }
func (f *fakePlayer) Poll() (bool, error)            { return f.active, f.pollErr }
func (f *fakePlayer) Close() error                   { f.closed = true; return nil }

func newFakePool() (*voicePool, *[]*fakePlayer) {
	var made []*fakePlayer
	pool := newVoicePool(func() (clipPlayer, error) {
		p := &fakePlayer{id: len(made)}
		made = append(made, p)
		return p, nil
	})
	return pool, &made
}

func TestReplaceWithoutOverlap(t *testing.T) {
	pool, made := newFakePool()
	if err := pool.play("a", "a.wav", time.Time{}, false); err != nil {
		t.Fatal(err)
	}
	if err := pool.play("b", "b.wav", time.Time{}, false); err != nil {
		t.Fatal(err)
	}
	if len(pool.voices) != 1 || !pool.playing("b") || pool.playing("a") {
		t.Fatalf("voices %+v", pool.voices)
	}
	if len(*made) != 1 {
		t.Fatalf("single-voice mode opened %d players", len(*made))
	}
}

func TestOverlapCapsAtEightAndCutsOldest(t *testing.T) {
	pool, made := newFakePool()
	for n := range 9 {
		if err := pool.play(string(rune('a'+n)), "clip.wav", time.Time{}, true); err != nil {
			t.Fatal(err)
		}
	}
	if len(pool.voices) != maxVoices || pool.playing("a") || !pool.playing("i") {
		t.Fatalf("voices %+v", pool.voices)
	}
	if len(*made) != maxVoices || (*made)[0].stops != 1 {
		t.Fatalf("expected 8 players with the oldest reused, got %d", len(*made))
	}
}

func TestPollFreesFinishedVoices(t *testing.T) {
	pool, made := newFakePool()
	_ = pool.play("a", "a.wav", time.Time{}, true)
	_ = pool.play("b", "b.wav", time.Time{}, true)
	(*made)[0].active = false // Clip a finished.
	if err := pool.poll(); err != nil {
		t.Fatal(err)
	}
	if pool.playing("a") || !pool.playing("b") || len(pool.idle) != 1 {
		t.Fatalf("voices %+v idle %d", pool.voices, len(pool.idle))
	}
	_ = pool.play("c", "c.wav", time.Time{}, true)
	if len(*made) != 2 {
		t.Fatal("finished player not reused")
	}
	(*made)[1].pollErr = errors.New("native poll failed")
	if err := pool.poll(); err == nil || pool.playing("b") {
		t.Fatal("poll error not reported or failing voice kept")
	}
}

func TestStopAllAndClose(t *testing.T) {
	pool, made := newFakePool()
	_ = pool.play("a", "a.wav", time.Time{}, true)
	_ = pool.play("b", "b.wav", time.Time{}, true)
	if got := pool.paths(); len(got) != 2 {
		t.Fatalf("paths %v", got)
	}
	if err := pool.stopAll(); err != nil || pool.active() {
		t.Fatal("stop all left voices")
	}
	if err := pool.close(); err != nil {
		t.Fatal(err)
	}
	for _, p := range *made {
		if !p.closed {
			t.Fatal("player not closed")
		}
	}
	if !pool.loaded() {
		t.Fatal("loaded state lost")
	}
}

func TestOpenFailureAddsNoVoice(t *testing.T) {
	pool := newVoicePool(func() (clipPlayer, error) { return nil, errors.New("companion missing") })
	if err := pool.play("a", "a.wav", time.Time{}, true); err == nil || pool.active() || pool.loaded() {
		t.Fatal("failed open created a voice")
	}
}

func TestTimersFollowVoices(t *testing.T) {
	pool, made := newFakePool()
	end := time.Unix(100, 0)
	_ = pool.play("a", "a.wav", end, true)
	_ = pool.play("b", "b.wav", time.Time{}, true) // Unknown length: no timer.
	_ = pool.play("c", "c.wav", end.Add(time.Second), true)
	if got := pool.timers(); len(got) != 2 || got[0].Control != "a" || got[1].Control != "c" || !got[1].Ends.Equal(end.Add(time.Second)) {
		t.Fatalf("timers %+v", got)
	}
	(*made)[0].active = false // a finished.
	if err := pool.poll(); err != nil {
		t.Fatal(err)
	}
	if got := pool.timers(); len(got) != 1 || got[0].Control != "c" {
		t.Fatalf("finished clip kept its timer %+v", got)
	}
	if err := pool.stopAll(); err != nil || pool.timers() != nil {
		t.Fatal("Stop kept timers", err)
	}
}
