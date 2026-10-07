package audio

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/voicemeeter"
)

// EchoTargets locates echo cancellation's channels in Voicemeeter Potato's
// insert buffers. Input inserts carry 2 channels per physical strip (5) then
// 8 per virtual strip; output inserts carry 8 channels per bus.
type EchoTargets struct {
	Mic       [2]int // Input-insert channels of the managed mic strip; -1 when unused.
	Reference [8]int // Output-insert channels of the playback bus; -1 when unused.
	Playback  string // Device assigned to the playback bus, for speaker matching.
	Reason    string // Why echo cancellation cannot run now; empty when it can.
	// Stream counts audio stream starts (engine restarts, stream changes and
	// monitor restarts); 0 without an active callback monitor.
	Stream uint32
}

// EchoTargets reports the current targets from the latest observed state.
func (i *Instance) EchoTargets() EchoTargets {
	i.mu.Lock()
	defer i.mu.Unlock()
	return echoTargets(i.state)
}

const physicalStrips = 5

func echoTargets(s control.State) EchoTargets {
	t := EchoTargets{Mic: [2]int{-1, -1}, Reference: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}}
	if cb := s.Snapshot.Callback; cb != nil && cb.Active {
		t.Stream = cb.Starting
	}
	if !s.Live || !s.Connected {
		t.Reason = "Not live"
		return t
	}
	if s.Snapshot.Edition != 3 || s.Plan == nil || s.Plan.Topology == nil {
		t.Reason = "Needs Potato"
		return t
	}
	topology := s.Plan.Topology
	voice := topology.Voice
	switch {
	case voice == nil:
		t.Reason = "No mic"
		return t
	case voice.Effective == "off":
		t.Reason = "Mic off"
		return t
	case voice.Strip < 0:
		t.Reason = "No mic"
		return t
	}
	bus, ok := hardwareBus(topology.PlaybackTarget)
	if !ok {
		t.Reason = "No playback"
		return t
	}
	first := 2 * voice.Strip
	if voice.Strip >= physicalStrips {
		first = 2*physicalStrips + 8*(voice.Strip-physicalStrips)
	}
	t.Mic = [2]int{first, first + 1}
	for c := range t.Reference {
		t.Reference[c] = 8*(bus-1) + c
	}
	t.Playback = s.Snapshot.Assignments[topology.PlaybackTarget]
	return t
}

// hardwareBus parses A1-A5 into its 1-based bus number.
func hardwareBus(target string) (int, bool) {
	number, ok := strings.CutPrefix(target, "A")
	if !ok {
		return 0, false
	}
	bus, err := strconv.Atoi(number)
	if err != nil || bus < 1 || bus > 5 {
		return 0, false
	}
	return bus, true
}

// insertRequest is read by the worker each step.
func (i *Instance) insertRequest() (*voicemeeter.InsertHook, uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.echoInsert, i.echoGeneration
}

// SetEchoInsert asks the worker to register the callback with hook, or to
// remove it when hook is nil. Installation is asynchronous. Removal waits
// until the worker confirms that the callback no longer calls the stages, or
// that the audio plugin stopped cleanly; only then may the caller free them.
func (i *Instance) SetEchoInsert(ctx context.Context, hook *voicemeeter.InsertHook) error {
	i.mu.Lock()
	i.echoGeneration++
	generation := i.echoGeneration
	i.echoInsert = nil
	if hook != nil {
		copied := *hook
		i.echoInsert = &copied
	}
	i.mu.Unlock()
	select {
	case i.actions <- control.Refresh:
	default:
	}
	if hook != nil {
		return nil
	}
	for {
		i.mu.Lock()
		released := i.state.InsertGeneration >= generation && i.state.Insert == nil
		i.mu.Unlock()
		if released {
			return nil
		}
		select {
		case <-i.done:
			if i.stopError != nil {
				return fmt.Errorf("echo insert release uncertain: %w", i.stopError)
			}
			return nil
		case <-ctx.Done():
			return fmt.Errorf("echo insert release unconfirmed: %w", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
}
