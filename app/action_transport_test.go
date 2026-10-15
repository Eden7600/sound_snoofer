package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"sound-snoofer/snoofer"
)

func TestActionReaderRejectsFullTrayQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := newActionSession(ctx, cancel)
	input := bytes.NewBufferString("{\"ID\":\"first\",\"Kind\":\"retry\"}\n{\"ID\":\"second\",\"Kind\":\"retry\"}\n")
	if err := readActions(ctx, input, make(chan UIAction), session); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second"} {
		select {
		case got := <-session.replies:
			if got.ID != id || got.Status != snoofer.Rejected {
				t.Fatal("queue refusal lost identity", got)
			}
		default:
			t.Fatal("full tray queue lost a reply")
		}
	}
}

func TestActionRepliesStayWithOriginalSession(t *testing.T) {
	oldCtx, closeOld := context.WithCancel(context.Background())
	defer closeOld()
	old := newActionSession(oldCtx, closeOld)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fresh := newActionSession(ctx, cancel)
	actions := make(chan UIAction, 1)
	if err := readActions(oldCtx, bytes.NewBufferString(`{"ID":"old","Kind":"retry"}`), actions, old); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	action := <-actions
	if action.session != old {
		t.Fatal("action did not retain its source session")
	}
	closeOld()
	action.reply(snoofer.Outcome{Status: snoofer.Failed, Detail: "late"})
	if len(old.replies) != 0 || len(fresh.replies) != 0 {
		t.Fatal("late old completion reached a session")
	}
	complete := fresh.completion("new")
	complete(snoofer.Outcome{Status: snoofer.Processed})
	complete(snoofer.Outcome{Status: snoofer.Failed})
	if len(fresh.replies) != 1 || (<-fresh.replies).Status != snoofer.Processed {
		t.Fatal("completion was not once-only")
	}
}

func TestReplyLaneCapacityViolationDetaches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := newActionSession(ctx, cancel)
	for n := range actionCapacity {
		session.completion(fmt.Sprint(n))(snoofer.Outcome{Status: snoofer.Processed})
	}
	if ctx.Err() != nil || len(session.replies) != actionCapacity {
		t.Fatal("valid saturated lane was rejected")
	}
	session.completion("impossible-extra")(snoofer.Outcome{Status: snoofer.Processed})
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("overflow silently dropped a terminal result")
	}
}

type resultWriter struct {
	bytes.Buffer
	writes int
	cancel context.CancelFunc
}

func (w *resultWriter) Write(data []byte) (int, error) {
	n, err := w.Buffer.Write(data)
	w.writes++
	if w.writes == actionCapacity {
		w.cancel()
	}
	return n, err
}

func TestResultFramesCannotBeCoalescedOrStarved(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan []byte, 1)
	for n := range 100 {
		offer(updates, []byte(fmt.Sprintf(`{"Version":%d,"State":{"Notice":"latest"}}`, n+1)))
	}
	session := newActionSession(ctx, cancel)
	for n := range actionCapacity {
		session.completion(fmt.Sprint(n))(snoofer.Outcome{Status: snoofer.Processed})
	}
	w := &resultWriter{cancel: cancel}
	if err := writeFrames(ctx, w, updates, session.replies); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&w.Buffer)
	for n := range actionCapacity {
		var fields map[string]json.RawMessage
		if err := decoder.Decode(&fields); err != nil {
			t.Fatal(err)
		}
		var result ActionResult
		if err := json.Unmarshal(fields["Result"], &result); err != nil || result.ID != fmt.Sprint(n) {
			t.Fatal("terminal result lost or replaced", result, err)
		}
		if fields["State"] != nil || fields["Version"] != nil || fields["Artwork"] != nil {
			t.Fatal("result carried presentation state", fields)
		}
	}
}
