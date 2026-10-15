package browserbridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type wireCommand struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	MeetCommand
}

func captured(l *MeetLease, browser string, tab int64) MeetCommand {
	return MeetCommand{Generation: l.Generation(browser), Tab: tab, Instance: "42", Call: "3", Control: "mic", Revision: 8, Value: false}
}

func reply(t *testing.T, ws *websocket.Conn, command wireCommand, change func(*MeetResult)) {
	t.Helper()
	c := command.MeetCommand
	result := MeetResult{ID: command.ID, Tab: c.Tab, Instance: c.Instance, Call: c.Call, Control: c.Control, Revision: c.Revision, Value: c.Value, Status: "observed", Reason: ""}
	if change != nil {
		change(&result)
	}
	writeJSON(t, ws, struct {
		Type string `json:"type"`
		MeetResult
	}{"meet-result", result})
}

func assertNoResult(t *testing.T, ch <-chan MeetResult) {
	t.Helper()
	select {
	case result := <-ch:
		t.Fatal("unexpected result", result)
	default:
	}
}

func TestMeetResultMatchesExactConnectionTargetAndIntent(t *testing.T) {
	b := newTestBridge(t)
	meet := meetLease(t, b)
	media := mediaLease(t, b)
	a := connect(t, b, "A", 1)
	other := connect(t, b, "B", 1)
	var caps capabilities
	readJSON(t, a, &caps)
	readJSON(t, other, &caps)
	receive(t, meet.Updates())
	receive(t, meet.Updates())
	receive(t, media.Updates())
	receive(t, media.Updates())
	results, err := meet.Send(context.Background(), "A", captured(meet, "A", 7))
	if err != nil {
		t.Fatal(err)
	}
	var command wireCommand
	readJSON(t, a, &command)
	if command.Type != "meet-command" || command.Value || command.ID == "" {
		t.Fatal(command)
	}
	for _, change := range []func(*MeetResult){
		func(r *MeetResult) { r.ID = "999" }, func(r *MeetResult) { r.Tab++ }, func(r *MeetResult) { r.Instance = "43" },
		func(r *MeetResult) { r.Call = "4" }, func(r *MeetResult) { r.Control = "camera" }, func(r *MeetResult) { r.Revision++ }, func(r *MeetResult) { r.Value = true },
	} {
		reply(t, a, command, change)
	}
	reply(t, other, command, nil)
	// A subsequent full-state report is a reader-order barrier, without timing
	// assumptions or relying on a negative read timeout that poisons the socket.
	writeJSON(t, a, map[string]any{"type": "meet-state", "calls": []MeetCall{}})
	receive(t, meet.Updates())
	writeJSON(t, other, map[string]any{"type": "meet-state", "calls": []MeetCall{}})
	receive(t, meet.Updates())
	assertNoResult(t, results)
	// Media may be backlogged without preventing a result from reaching its slot.
	b.mu.Lock()
	for range 16 {
		b.mediaLocked(b.conns["A"], nil)
	}
	b.mu.Unlock()
	reply(t, a, command, nil)
	r := receive(t, results)
	if r.Status != "observed" || r.Value || r.ID != command.ID {
		t.Fatal(r)
	}
	if _, open := <-results; open {
		t.Fatal("result channel did not settle once")
	}
	reply(t, a, command, nil)
	if meet.Generation("A") == 0 {
		t.Fatal("result bypass retired connection")
	}
}

func TestMeetPendingReservationsAndStopPreserveMedia(t *testing.T) {
	b := newTestBridge(t)
	meet := meetLease(t, b)
	media := mediaLease(t, b)
	ws := connect(t, b, "Browser", 1)
	var caps capabilities
	readJSON(t, ws, &caps)
	receive(t, meet.Updates())
	receive(t, media.Updates())
	command := captured(meet, "Browser", 7)
	results, err := meet.Send(context.Background(), "Browser", command)
	if err != nil {
		t.Fatal(err)
	}
	var sent wireCommand
	readJSON(t, ws, &sent)
	if _, err := meet.Send(context.Background(), "Browser", command); !errors.Is(err, ErrBusy) {
		t.Fatal("duplicate control accepted", err)
	}
	other := command
	other.Control = "camera"
	camera, err := meet.Send(context.Background(), "Browser", other)
	if err != nil {
		t.Fatal("independent control rejected", err)
	}
	readJSON(t, ws, &sent)
	closeLease(t, meet.lease)
	for _, ch := range []<-chan MeetResult{results, camera} {
		if result := receive(t, ch); result.Status != "unknown" || result.Reason != "stopped" {
			t.Fatal(result)
		}
	}
	readJSON(t, ws, &caps)
	if caps.MeetEnabled || !caps.MediaEnabled || media.Generation("Browser") != command.Generation {
		t.Fatal("Meet stop affected Media", caps)
	}
	if err := media.Send("Browser", MediaCommand{Type: "command", Generation: command.Generation, ID: "7:0", Instance: "1", Op: "next"}); err != nil {
		t.Fatal(err)
	}
	var mediaCommand MediaCommand
	readJSON(t, ws, &mediaCommand)
	if mediaCommand.Op != "next" {
		t.Fatal(mediaCommand)
	}
}

func TestMeetPendingLimitAndNoReplayAfterReplacement(t *testing.T) {
	b := newTestBridge(t)
	meet := meetLease(t, b)
	ws := connect(t, b, "Browser", 1)
	var caps capabilities
	readJSON(t, ws, &caps)
	receive(t, meet.Updates())
	old := meet.Generation("Browser")
	var results []<-chan MeetResult
	for n := int64(0); n < 8; n++ {
		ch, err := meet.Send(context.Background(), "Browser", captured(meet, "Browser", n))
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, ch)
		var command wireCommand
		readJSON(t, ws, &command)
	}
	if _, err := meet.Send(context.Background(), "Browser", captured(meet, "Browser", 9)); !errors.Is(err, ErrBusy) {
		t.Fatal("pending limit not enforced", err)
	}
	ws.Close()
	eventually(t, func() bool { return meet.Generation("Browser") == 0 })
	for _, ch := range results {
		if r := receive(t, ch); r.Status != "unknown" || r.Reason != "disconnected" {
			t.Fatal(r)
		}
	}
	replacement := connect(t, b, "Browser", 1)
	readJSON(t, replacement, &caps)
	command := captured(meet, "Browser", 1)
	command.Generation = old
	if _, err := meet.Send(context.Background(), "Browser", command); !errors.Is(err, ErrConnectionChanged) {
		t.Fatal("stale generation accepted", err)
	}
	fresh := captured(meet, "Browser", 10)
	ch, err := meet.Send(context.Background(), "Browser", fresh)
	if err != nil {
		t.Fatal(err)
	}
	var only wireCommand
	readJSON(t, replacement, &only)
	if only.Tab != 10 || only.ID != "9" {
		t.Fatal("old operation replayed", only)
	}
	reply(t, replacement, only, nil)
	if r := receive(t, ch); r.Status != "observed" {
		t.Fatal(r)
	}
}

func TestMeetTimeoutAndCanceledWriteDoNotClaimObservation(t *testing.T) {
	for _, beforeWrite := range []bool{false, true} {
		t.Run(map[bool]string{true: "before-write", false: "after-write"}[beforeWrite], func(t *testing.T) {
			b := newTestBridge(t)
			meet := meetLease(t, b)
			ws := connect(t, b, "Browser", 1)
			var caps capabilities
			readJSON(t, ws, &caps)
			receive(t, meet.Updates())
			b.mu.Lock()
			c := b.conns["Browser"]
			b.mu.Unlock()
			if beforeWrite {
				c.gate <- struct{}{}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ch, err := meet.Send(ctx, "Browser", captured(meet, "Browser", 7))
			if err != nil {
				t.Fatal(err)
			}
			if !beforeWrite {
				var command wireCommand
				readJSON(t, ws, &command)
			}
			r := receive(t, ch)
			if beforeWrite {
				<-c.gate
				if r.Status != "rejected" || r.Reason != "busy" {
					t.Fatal(r)
				}
			} else if r.Status != "unknown" || r.Reason != "not-confirmed" {
				t.Fatal(r)
			}
		})
	}
}

func TestCanceledTargetCannotWriteAfterGateRelease(t *testing.T) {
	b := newTestBridge(t)
	meet := meetLease(t, b)
	ws := connect(t, b, "Browser", 1)
	var caps capabilities
	readJSON(t, ws, &caps)
	receive(t, meet.Updates())
	b.mu.Lock()
	c := b.conns["Browser"]
	b.mu.Unlock()
	c.gate <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := meet.Send(ctx, "Browser", captured(meet, "Browser", 7))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	<-c.gate
	if r := receive(t, ch); r.Status != "rejected" || r.Reason != "disabled" {
		t.Fatal(r)
	}
	// A fresh operation is the first command ever seen by this socket.
	fresh, err := meet.Send(context.Background(), "Browser", captured(meet, "Browser", 8))
	if err != nil {
		t.Fatal(err)
	}
	var command wireCommand
	readJSON(t, ws, &command)
	if command.Tab != 8 {
		t.Fatal("canceled operation wrote", command)
	}
	reply(t, ws, command, nil)
	receive(t, fresh)
}

func TestDisabledMediaReportsCannotRetireMeet(t *testing.T) {
	b := newTestBridge(t)
	meet := meetLease(t, b)
	ws := connect(t, b, "Browser", 1)
	var caps capabilities
	readJSON(t, ws, &caps)
	initial := receive(t, meet.Updates())
	writeJSON(t, ws, map[string]any{"type": "sessions", "sessions": []MediaSession{{ID: "invalid"}}})
	writeJSON(t, ws, map[string]any{"type": "meet-state", "calls": []MeetCall{}})
	observed := receive(t, meet.Updates())
	if !observed.Connected || observed.Calls == nil || observed.Generation != initial.Generation {
		t.Fatal("disabled Media affected Meet", observed)
	}
}
