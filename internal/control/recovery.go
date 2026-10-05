package control

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type recoveryJournal struct {
	Attempts  []time.Time
	Uncertain bool
}

type recovery struct {
	callback                    callbackHealth
	blocked                     bool
	verifyCallbacks             bool
	verifyBuffers, verifySynced uint32
	verifiedAt                  time.Time
	clearSamples                int
	path                        string
	attempts                    []time.Time
	err                         error
	pending                     bool
	started                     time.Time
	samples                     int
	status                      string
}

func newRecovery(path string) *recovery {
	r := &recovery{path: path + ".recovery.json"}
	b, e := os.ReadFile(r.path)
	if e == nil {
		var journal recoveryJournal
		r.err = json.Unmarshal(b, &journal)
		if r.err != nil {
			// Existing releases stored only the attempt timestamps.
			r.err = json.Unmarshal(b, &r.attempts)
		} else {
			r.attempts, r.blocked = journal.Attempts, journal.Uncertain
		}
	} else if !os.IsNotExist(e) {
		r.err = e
	}
	return r
}
func (r *recovery) restart(b interface{ RestartEngine() error }, live bool, now time.Time) error {
	if !live {
		return fmt.Errorf("preview: engine restart not sent")
	}
	if r.pending {
		return fmt.Errorf("audio recovery already in progress")
	}
	if r.err != nil {
		return fmt.Errorf("recovery journal: %w", r.err)
	}
	// Manual actions may bypass automatic cooldown, but every attempt is durably recorded before submission.
	recent := []time.Time{}
	for _, t := range r.attempts {
		if now.Sub(t) < 10*time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 64 {
		return fmt.Errorf("too many restart requests; wait ten minutes")
	}
	recent = append(recent, now)
	if err := config.WriteJournal(r.path, recoveryJournal{Attempts: recent, Uncertain: true}); err != nil {
		return err
	}
	r.attempts = recent
	r.blocked = true
	r.verifyBuffers, r.verifySynced = r.callback.buffers, r.callback.synced
	r.clearSamples = 0
	if err := b.RestartEngine(); err != nil {
		r.status = "Restart outcome unknown; manual attention required"
		return err
	}
	r.blocked = false
	if err := config.WriteJournal(r.path, recoveryJournal{Attempts: r.attempts}); err != nil {
		r.blocked = true
		r.err = err
		r.status = "Restart submitted; recovery journal failed; manual attention required"
		return err
	}
	r.verifyCallbacks = r.callback.enabled
	r.verifyBuffers, r.verifySynced = r.callback.buffers, r.callback.synced
	r.verifiedAt = time.Time{}
	r.pending = true
	r.started = now
	r.samples = 0
	r.status = "Restarting audio engine"
	return nil
}
func (r *recovery) observe(s model.Snapshot, now time.Time) string {
	if r.blocked && r.err == nil {
		cb := s.Callback
		if cb != nil && cb.Active && cb.Error == "" && cb.Buffers != r.verifyBuffers && cb.Synced != r.verifySynced {
			r.clearSamples++
			r.verifyBuffers, r.verifySynced = cb.Buffers, cb.Synced
			if r.clearSamples >= 2 {
				if err := config.WriteJournal(r.path, recoveryJournal{Attempts: r.attempts}); err != nil {
					r.err = err
				} else {
					r.blocked = false
				}
			}
		} else {
			r.clearSamples = 0
		}
	}
	if r.pending {
		if now.Sub(r.started) > 10*time.Second {
			r.pending = false
			r.status = "Restart unverified; inspect audio"
			return r.status
		}
		if now.Sub(r.started) < time.Second {
			return r.status
		}
		if r.verifyCallbacks {
			cb := s.Callback
			if cb == nil || !cb.Active || cb.Error != "" {
				r.samples = 0
			} else if cb.Buffers != r.verifyBuffers && cb.Synced != r.verifySynced &&
				(r.verifiedAt.IsZero() || now.Sub(r.verifiedAt) >= 500*time.Millisecond) {
				r.samples++
				r.verifiedAt = now
				r.verifyBuffers, r.verifySynced = cb.Buffers, cb.Synced
			}
		} else if sr, ok := s.Numbers["Bus[0].device.sr"]; ok && sr > 0 {
			r.samples++
		} else {
			r.samples = 0
		}
		if r.samples >= 2 {
			r.pending = false
			r.status = "Engine responding; audio continuity unverified"
			if r.verifyCallbacks {
				r.status = "Audio processing resumed; audible output unverified"
			}
		}
		return r.status
	}
	if s.Assignments == nil {
		return "Engine health unknown"
	}
	if s.Assignments["A1"] != "" {
		if sr, ok := s.Numbers["Bus[0].device.sr"]; !ok {
			return "Engine health unknown"
		} else if sr <= 0 {
			return "A1 stream unavailable; restart available"
		}
	}
	return "No engine fault observed (stall detection unverified)"
}

// automaticPermit is deliberately separate from signal levels. A caller must
// supply a backend-validated fault signature before requesting automatic repair.
func (r *recovery) automaticPermit(now time.Time, validated bool, recorder string) error {
	if r.err != nil {
		return fmt.Errorf("recovery journal: %w", r.err)
	}
	if r.blocked {
		return fmt.Errorf("previous restart outcome uncertain; manual retry required")
	}
	if !validated {
		return fmt.Errorf("no validated engine-stall evidence")
	}
	if r.pending {
		return fmt.Errorf("recovery already in flight")
	}
	if recorder != "Stopped" {
		return fmt.Errorf("automatic recovery deferred during active or unknown transport")
	}
	recent := 0
	for _, at := range r.attempts {
		elapsed := now.Sub(at)
		if elapsed < 0 {
			return fmt.Errorf("clock moved backwards; automatic recovery suspended")
		}
		if elapsed < time.Minute {
			return fmt.Errorf("automatic recovery cooldown")
		}
		if elapsed < 10*time.Minute {
			recent++
		}
	}
	if recent >= 2 {
		return fmt.Errorf("automatic recovery budget exhausted")
	}
	return nil
}
