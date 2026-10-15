package appaudio

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
)

func TestQueuedFocusCannotRetargetAnotherApp(t *testing.T) {
	backend := studio()
	w := &worker{services: snoofer.Services{Controls: snoofer.NewControls(), Live: true}, timing: defaultTiming,
		open:  func() (windowsaudio.SessionBackend, error) { return backend, nil },
		peaks: map[string]float64{}, heard: map[string]time.Time{}, pending: map[string]request{}, ignored: map[string]bool{}}
	now := time.Now()
	w.step(now)
	w.focus = key("Google Chrome")
	queue := make(chan command, 8)
	w.publish(context.Background(), queue, now)
	var c snoofer.Control
	for _, control := range w.services.Controls.Snapshot() {
		if control.ID == "appaudio.focus" {
			c = control
		}
	}
	if err := w.services.Controls.Dispatch(context.Background(), snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "adjust", Delta: 1}); err != nil {
		t.Fatal(err)
	}
	queued := <-queue
	w.focus = key("Discord")
	w.process(context.Background(), queued, now)
	if backend.writes != 0 {
		t.Fatal("queued focus input wrote to replacement app", backend.writes)
	}
}

func outcomeWorker(t *testing.T, backend windowsaudio.SessionBackend) *worker {
	t.Helper()
	settings := Settings{}
	raw, _ := json.Marshal(settings)
	w := &worker{services: snoofer.Services{Controls: snoofer.NewControls(), Live: true, SaveSettings: func(string, json.RawMessage, json.RawMessage) error { return nil }},
		settings: settings, raw: raw, timing: defaultTiming, open: func() (windowsaudio.SessionBackend, error) { return backend, nil },
		peaks: map[string]float64{}, heard: map[string]time.Time{}, pending: map[string]request{}, ignored: map[string]bool{}}
	w.step(time.Now())
	return w
}

func queued(t *testing.T, w *worker, ctx context.Context, id, op, value string, delta int) (command, chan snoofer.Outcome) {
	t.Helper()
	queue := make(chan command, 1)
	w.publish(ctx, queue, time.Now())
	var request snoofer.Request
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID == id {
			request = snoofer.Request{ID: id, Revision: c.Revision, Operation: op, Value: value, Delta: delta}
		}
	}
	results := make(chan snoofer.Outcome, 2)
	if err := w.services.Controls.DispatchWithResult(context.Background(), request, func(result snoofer.Outcome) { results <- result }); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-results:
		t.Fatal("enqueue claimed completion", result)
	default:
	}
	return <-queue, results
}

func TestAppTargetMembershipAndLifetimeCannotRetarget(t *testing.T) {
	for _, change := range []string{"membership", "reopen", "backend", "focus-aba"} {
		t.Run(change, func(t *testing.T) {
			backend := studio()
			w := outcomeWorker(t, backend)
			w.focus = key("Google Chrome")
			id := controlID("Google Chrome")
			if change == "focus-aba" {
				id = "appaudio.focus"
			}
			c, results := queued(t, w, context.Background(), id, "adjust", "", 1)
			switch change {
			case "membership":
				backend.sessions = slices.DeleteFunc(backend.sessions, func(s windowsaudio.Session) bool { return s.Key == "c2" })
				w.listAt = time.Time{}
				w.step(time.Now())
			case "reopen":
				before := backend.sessions
				backend.sessions = nil
				w.listAt = time.Time{}
				w.step(time.Now())
				w.refreshContexts(time.Now())
				backend.sessions = before
				w.listAt = time.Time{}
				w.step(time.Now())
			case "backend":
				w.closeBackend()
				w.step(time.Now())
			case "focus-aba":
				w.focus = key("Discord")
				w.refreshContexts(time.Now())
				w.focus = key("Google Chrome")
			}
			w.process(context.Background(), c, time.Now())
			if result := <-results; result.Status != snoofer.Rejected {
				t.Fatal(result)
			}
			if backend.writes != 0 {
				t.Fatal("changed target was written", backend.writes)
			}
		})
	}
}

type partialSessions struct {
	*fakeSessions
	volumeCalls []string
}

func (b *partialSessions) SetVolume(key string, value float64) error {
	b.volumeCalls = append(b.volumeCalls, key)
	if key == "c2" {
		return errors.New("second session refused volume")
	}
	return b.fakeSessions.SetVolume(key, value)
}

func TestAppFiniteWritesReportPartialFailure(t *testing.T) {
	backend := &partialSessions{fakeSessions: studio()}
	w := outcomeWorker(t, backend)
	c, results := queued(t, w, context.Background(), controlID("Google Chrome"), "set", "25", 0)
	w.process(context.Background(), c, time.Now())
	result := <-results
	if result.Status != snoofer.Failed || !strings.Contains(result.Detail, "second session") {
		t.Fatal(result)
	}
	if !slices.Equal(backend.volumeCalls, []string{"c1", "c2", "c3"}) {
		t.Fatal("finite writes lost partial error", backend.volumeCalls)
	}
	if pending := w.pending[key("Google Chrome")]; pending.failed == "" || !pending.volume {
		t.Fatal("native status lost", pending)
	}
}

func TestAppProcessedDoesNotClaimObservedAndDeltasAccumulate(t *testing.T) {
	backend := studio()
	backend.stubborn[`C:\Chrome\chrome.exe`] = true
	w := outcomeWorker(t, backend)
	first, results := queued(t, w, context.Background(), controlID("Google Chrome"), "adjust", "", 1)
	second, secondResults := queued(t, w, context.Background(), controlID("Google Chrome"), "adjust", "", 2)
	w.process(context.Background(), first, time.Now())
	w.process(context.Background(), second, time.Now())
	if a, b := <-results, <-secondResults; a.Status != snoofer.Processed || b.Status != snoofer.Processed {
		t.Fatal(a, b)
	}
	if pending := w.pending[key("Google Chrome")]; pending.wantVolume != .86 {
		t.Fatal("same-target detents did not accumulate", pending)
	}
	if _, status := w.appValue(w.find(key("Google Chrome"))); status != "Pending" {
		t.Fatal("processed setter claimed observation", status)
	}
	w.observe(time.Now().Add(w.timing.observe + time.Second))
	if !w.ignored[key("Google Chrome")] || backend.writes != 6 {
		t.Fatal("missing readback was replayed or hidden", backend.writes)
	}
}

func TestAppMuteCapturesDesiredBoolean(t *testing.T) {
	backend := studio()
	w := outcomeWorker(t, backend)
	c, results := queued(t, w, context.Background(), controlID("Discord"), "press", "", 0)
	// A newer pending mute must not invert this captured click again.
	w.pending[key("Discord")] = request{mute: true, wantMute: true}
	w.process(context.Background(), c, time.Now())
	if result := <-results; result.Status != snoofer.Processed {
		t.Fatal(result)
	}
	if !w.pending[key("Discord")].wantMute {
		t.Fatal("queued mute re-resolved its desired value")
	}
}

func TestAppSettingsCaptureAndErrors(t *testing.T) {
	for _, test := range []string{"stale", "membership", "missing", "validation", "save"} {
		t.Run(test, func(t *testing.T) {
			backend := studio()
			w := outcomeWorker(t, backend)
			saves := 0
			w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error {
				saves++
				if test == "save" {
					return errors.New("disk denied")
				}
				return nil
			}
			value := `{"op":"rename","app":"Google Chrome","value":"Browser"}`
			if test == "validation" {
				value = "bad json"
			}
			if test == "missing" {
				value = `{"op":"rename","app":"New App","value":"Renamed"}`
			}
			c, results := queued(t, w, context.Background(), "appaudio.edit", "set", value, 0)
			switch test {
			case "stale":
				w.settingsGeneration++
			case "membership":
				backend.sessions = slices.DeleteFunc(backend.sessions, func(s windowsaudio.Session) bool { return s.Key == "c2" })
				w.listAt = time.Time{}
				w.step(time.Now())
			case "missing":
				w.apps = append(w.apps, &app{Name: "New App", Sessions: []windowsaudio.Session{{Key: "new", Path: "new.exe"}}})
			}
			w.process(context.Background(), c, time.Now())
			result := <-results
			if test == "save" {
				if result.Status != snoofer.Failed || result.Detail != "disk denied" || saves != 1 {
					t.Fatal(result, saves)
				}
			} else if result.Status != snoofer.Rejected || saves != 0 {
				t.Fatal("refused edit reached save", result, saves)
			}
		})
	}
}

func TestAppQueueRefusalStopAndLateCompletion(t *testing.T) {
	backend := studio()
	w := outcomeWorker(t, backend)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue := make(chan command, 1)
	w.publish(ctx, queue, time.Now())
	var r snoofer.Request
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID == controlID("Discord") {
			r = snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
		}
	}
	results := make(chan snoofer.Outcome, 3)
	complete := func(o snoofer.Outcome) { results <- o }
	if err := w.services.Controls.DispatchWithResult(context.Background(), r, complete); err != nil {
		t.Fatal(err)
	}
	if err := w.services.Controls.DispatchWithResult(context.Background(), r, complete); err == nil {
		t.Fatal("full queue accepted")
	}
	if result := <-results; result.Status != snoofer.Rejected {
		t.Fatal(result)
	}
	cancel()
	select {
	case result := <-results:
		if result.Status != snoofer.Unknown {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("provider stop left claim pending")
	}
	w.process(ctx, <-queue, time.Now())
	select {
	case result := <-results:
		t.Fatal("late completion delivered twice", result)
	default:
	}
	if backend.writes != 0 {
		t.Fatal("queued input wrote after stop")
	}
}
