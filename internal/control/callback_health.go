package control

import (
	"encoding/json"
	"fmt"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

// StallMessage is the health text for a qualified callback stall.
const StallMessage = "Audio engine stalled: no callback buffers"

// NoStallMessage is the health text while callback buffers advance. It claims
// processing progress only, which is what the monitor observes.
const NoStallMessage = "No stall detected"

// callbackHealth qualifies progress, not sample amplitude. Time gaps reset grace.
type callbackHealth struct {
	identity                                   string
	last, grace, progressed, firstFault        time.Time
	buffers, synced, starting, ending, changes uint32
	samples                                    int
	enabled                                    bool
}

// callbackTarget identifies the A1 device whose engine clock the callback
// monitor watches. An ASIO A1 needs the configured physical presence evidence
// (an installed ASIO driver alone never counts); any other A1 device needs an
// available non-ASIO output endpoint with the assigned name. The key changes
// with assignments and matching endpoints, so hotplug restarts grace.
func callbackTarget(cfg config.Config, s model.Snapshot) (string, string) {
	if s.Edition == 0 {
		return "", "Voicemeeter edition unknown"
	}
	a1 := s.Assignments["A1"]
	if a1 == "" {
		return "", "No device assigned to A1"
	}
	ids := []string{}
	if asioA1(cfg, s, a1) {
		if cfg.Studio == nil {
			return "", "A1 ASIO presence unknown: no configured interface"
		}
		dev, err := routing.SelectASIO(cfg.Studio, s)
		if err != nil {
			return "", "A1 presence uncertain: " + err.Error()
		}
		if dev == nil || dev.Name != a1 {
			return "", "A1 interface " + a1 + " not present; waiting for hardware"
		}
		for _, d := range s.Devices {
			if d.Available && d.Driver == "wdm" && d.Direction == "input" {
				for _, a := range cfg.Studio.ASIO {
					if a.ASIORegex != nil && a.PresenceRegex != nil && a.ASIORegex.MatchString(dev.Name) && a.PresenceRegex.MatchString(d.Name) {
						ids = append(ids, d.ID, d.Name)
					}
				}
			}
		}
	} else {
		for _, d := range s.Devices {
			if d.Available && d.Driver != "asio" && d.Direction == "output" && d.Name == a1 {
				ids = append(ids, d.Driver, d.ID, d.Name)
			}
		}
		if len(ids) == 0 {
			return "", "A1 device " + a1 + " not present; waiting for hardware"
		}
	}
	var asio any
	if cfg.Studio != nil {
		asio = cfg.Studio.ASIO
	}
	key, _ := json.Marshal([]any{s.Assignments, ids, asio})
	return string(key), ""
}

// asioA1 reports whether the A1 assignment names an ASIO interface.
func asioA1(cfg config.Config, s model.Snapshot, a1 string) bool {
	for _, d := range s.Devices {
		if d.Driver == "asio" && d.Name == a1 {
			return true
		}
	}
	if cfg.Studio != nil {
		for _, a := range cfg.Studio.ASIO {
			if a.ASIORegex != nil && a.ASIORegex.MatchString(a1) {
				return true
			}
		}
	}
	return false
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
		return false, NoStallMessage
	}
	if now.Sub(h.progressed) < 2*time.Second {
		return false, "Waiting for callback progress"
	}
	if h.firstFault.IsZero() {
		h.firstFault = now
	}
	h.samples++
	return h.samples >= 3 && now.Sub(h.firstFault) >= time.Second, StallMessage
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
	// A stalled engine may never confirm routing writes, so pending changes defer
	// dispatch only for a bounded time. Unresolved items have their own guards.
	if target := pendingRoutingTarget(p); target != "" && now.Sub(r.callback.firstFault) < routingSettleLimit {
		return fmt.Errorf("waiting for routing to settle: %s", target)
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

// routingSettleLimit bounds how long a qualified stall waits for routing changes.
const routingSettleLimit = 10 * time.Second

// pendingRoutingTarget names the first planned change, or "" when routing is settled.
func pendingRoutingTarget(p routing.Plan) string {
	if p.Topology != nil {
		for _, op := range p.Topology.Operations {
			if !op.Change {
				continue
			}
			if op.Parameter != "" {
				return op.Parameter
			}
			return op.Target
		}
		return ""
	}
	for _, d := range p.Decisions {
		if d.Change {
			return d.Target
		}
	}
	return ""
}

// callbackRestartInterval bounds how often the live owner restarts the monitor.
const callbackRestartInterval = 5 * time.Second

// callbackStream decides when the live monitor must be restarted. The Remote
// API requires restarting audio after the engine ends or changes the stream;
// otherwise the registration stays silent and a later stall goes unseen.
type callbackStream struct {
	changes     uint32
	known, lost bool
	restarted   time.Time
}

// restartDue records the latest status and reports whether to restart now.
// A stream loss stays pending through the rate limit.
func (c *callbackStream) restartDue(cb *model.CallbackStatus, now time.Time) bool {
	if cb == nil || !cb.Active || cb.Error != "" {
		*c = callbackStream{restarted: c.restarted}
		return false
	}
	ended := cb.Ending != 0 && cb.Ending >= cb.Starting
	changed := c.known && cb.Changes != c.changes
	c.lost = c.lost || ended || changed
	c.changes, c.known = cb.Changes, true
	if !c.lost {
		return false
	}
	since := now.Sub(c.restarted)
	if !c.restarted.IsZero() && since >= 0 && since < callbackRestartInterval {
		return false
	}
	c.lost = false
	c.restarted = now
	return true
}
