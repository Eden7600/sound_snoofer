package control

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

// callbackHealth qualifies progress, not sample amplitude. Time gaps reset grace.
type callbackHealth struct {
	identity                                   string
	last, grace, progressed, firstFault        time.Time
	buffers, synced, starting, ending, changes uint32
	samples                                    int
	enabled                                    bool
}

func callbackTarget(cfg config.Config, s model.Snapshot) (string, string) {
	if cfg.Studio == nil ||
		s.Edition != 3 || !strings.EqualFold(s.Assignments["A1"], "Universal Audio Volt") {
		return "", "Automatic recovery supports managed Volt A1 on Potato"
	}
	dev, err := routing.SelectASIO(cfg.Studio, s)
	if err != nil {
		return "", "A1 presence uncertain: " + err.Error()
	}
	if dev == nil {
		return "", "Volt disconnected; waiting for hardware before recovery"
	}
	if dev.Name != s.Assignments["A1"] {
		return "", "A1 assignment differs from managed Volt"
	}
	// Include physical identity, all assignments and matcher changes in transition grace.
	ids := []string{}
	for _, d := range s.Devices {
		if d.Available && d.Driver == "wdm" && d.Direction == "input" {
			for _, a := range cfg.Studio.ASIO {
				if a.ASIORegex != nil && a.PresenceRegex != nil && a.ASIORegex.MatchString(dev.Name) && a.PresenceRegex.MatchString(d.Name) {
					ids = append(ids, d.ID, d.Name)
				}
			}
		}
	}
	key, _ := json.Marshal([]any{s.Assignments, ids, cfg.Studio.ASIO})
	return string(key), ""
}

func (h *callbackHealth) update(cfg config.Config, s model.Snapshot, now time.Time) (bool, string) {
	cb := s.Callback
	if cb == nil || !cb.Active || cb.Error != "" {
		*h = callbackHealth{}
		if cb != nil && cb.Error != "" {
			return false, "Callback monitor unavailable: " + cb.Error
		}
		return false, "Callback monitoring inactive"
	}
	key, reason := callbackTarget(cfg, s)
	if reason != "" {
		*h = callbackHealth{enabled: true}
		return false, reason
	}
	reset := !h.enabled || key != h.identity || h.last.IsZero() || now.Before(h.last) || now.Sub(h.last) > 2*time.Second ||
		cb.Starting != h.starting || cb.Ending != h.ending || cb.Changes != h.changes
	progress := cb.Buffers != h.buffers
	syncProgress := cb.Synced != h.synced
	if reset {
		h.grace = now.Add(5 * time.Second)
		h.progressed = now
		h.samples = 0
		h.firstFault = time.Time{}
	}
	h.enabled, h.identity, h.last = true, key, now
	h.buffers, h.synced, h.starting, h.ending, h.changes = cb.Buffers, cb.Synced, cb.Starting, cb.Ending, cb.Changes
	if progress {
		h.progressed = now
		h.samples = 0
		h.firstFault = time.Time{}
	}
	if cb.Ending >= cb.Starting && cb.Ending != 0 {
		return false, "Callback stream ended; monitoring unverified"
	}
	if now.Before(h.grace) {
		return false, "Waiting for audio transition to settle"
	}
	if progress && syncProgress {
		return false, "Audio processing active; audible output unverified"
	}
	if now.Sub(h.progressed) < 2*time.Second {
		return false, "Waiting for callback progress"
	}
	if h.firstFault.IsZero() {
		h.firstFault = now
	}
	h.samples++
	return h.samples >= 3 && now.Sub(h.firstFault) >= time.Second, "Audio processing stalled: no callback buffers"
}

// automaticRestart rechecks inventory, identity, routing and transport at submission.
// It runs on the existing owner; no callbacks perform API calls or trigger writes.
func (r *recovery) automaticRestart(b *observed, cfg config.Config, live bool, now time.Time) error {
	if !live {
		return fmt.Errorf("preview: automatic recovery disabled")
	}
	if err := r.automaticPermit(now, true, "Stopped"); err != nil {
		return err
	}
	s, err := b.Snapshot()
	if err != nil {
		r.callback = callbackHealth{}
		return err
	}
	fault, reason := r.callback.update(cfg, s, now)
	if !fault {
		return fmt.Errorf("%s", reason)
	}
	p, err := routing.Build(cfg, s)
	if err != nil {
		return err
	}
	if p.HasChanges() || p.HasUnresolved() {
		return fmt.Errorf("waiting for routing to settle")
	}
	recorder, err := b.Recorder()
	if err != nil {
		return fmt.Errorf("recorder state unknown: %w", err)
	}
	if err = r.automaticPermit(now, true, recorder.State()); err != nil {
		return err
	}
	// Recorder reads can take time: ensure callback progress has not resumed meanwhile.
	s, err = b.ParameterSnapshot()
	if err != nil {
		r.callback = callbackHealth{}
		return err
	}
	if fault, reason = r.callback.update(cfg, s, time.Now()); !fault {
		return fmt.Errorf("%s", reason)
	}
	return r.restart(b, live, time.Now())
}
