package controller

import (
	"context"
	"fmt"
	"strings"

	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

type numberBackend interface{ SetNumber(string, int) error }

func matches(op routing.Operation, s model.Snapshot) bool {
	if op.Device != nil {
		v, ok := s.Assignments[op.Target]
		return ok && v == op.Device.Name
	}
	v, ok := operationValue(s, op.Parameter)
	return ok && v == float32(op.Value)
}
func (c *Controller) applyTopology(ctx context.Context, p routing.Plan) error {
	numeric, ok := c.Backend.(numberBackend)
	if !ok {
		return fmt.Errorf("backend does not support ASIO patch/routing writes")
	}
	fresh, e := c.Plan()
	if e != nil {
		return e
	}
	if fresh.Key() != p.Key() {
		return ErrPlanChanged
	}
	verified := 0
	ops := p.Topology.Operations
	expectedNumbers := map[string]float32{}
	expectedNames := map[string]string{}
	if p.Topology.Voice != nil {
		ops = fresh.Topology.Operations
		if len(fresh.Topology.Transition) > 0 {
			ops = fresh.Topology.Transition
		}
		for _, op := range fresh.Topology.Operations {
			if op.Device != nil {
				expectedNames[op.Target] = op.BeforeName
			} else {
				expectedNumbers[op.Parameter] = op.BeforeValue
			}
		}
	}
	check := func(s model.Snapshot, pending string) error {
		for param, v := range expectedNumbers {
			if param != pending {
				if current, ok := operationValue(s, param); !ok || current != v {
					return ErrPlanChanged
				}
			}
		}
		for target, v := range expectedNames {
			if target != pending && s.Assignments[target] != v {
				return ErrPlanChanged
			}
		}
		return nil
	}
	fail := func(e error) error { return fmt.Errorf("topology stopped (%d operations verified): %w", verified, e) }
	for _, op := range ops {
		if !op.Change {
			continue
		}
		if e = ctx.Err(); e != nil {
			return fail(e)
		}
		s, e := c.observe(op.Device == nil)
		if e != nil {
			return fail(e)
		}
		if routing.InventoryKey(s) != p.Topology.InventoryKey {
			return fail(ErrPlanChanged)
		}
		if e = check(s, ""); e != nil {
			return fail(e)
		}
		if matches(op, s) {
			continue
		}
		if strings.HasPrefix(op.Parameter, "Strip[") && strings.HasSuffix(op.Parameter, ".B1") && p.Topology.Recording != nil {
			r := c.readRecorder()
			micOff := false
			if i := c.Config.VoiceIntent(); i != nil && !i.MicActive() && op.Value == 0 {
				switch op.Parameter {
				case "Strip[0].B1", "Strip[1].B1", "Strip[2].B1", "Strip[6].B1":
					micOff = true
				}
			}
			conflict := r.Conflict()
			if i := routing.EffectiveIntent(c.Config, s); i != nil && i.Recording != nil && (i.Recording.ToVST || r.Values["Recorder.B2"] != 0) {
				conflict = r.RehearsalConflict()
			}
			if conflict != "" && !micOff {
				return fail(fmt.Errorf("%s", conflict))
			}
		}
		// Abort on unrelated/manual mutation between planning and execution. Own
		// transitions alter other parameters, not this operation's unique target.
		if op.Device != nil {
			if s.Assignments[op.Target] != op.BeforeName {
				return fail(ErrPlanChanged)
			}
		} else {
			if value, _ := operationValue(s, op.Parameter); value != op.BeforeValue {
				return fail(ErrPlanChanged)
			}
		}
		label := op.Parameter
		if op.Device != nil {
			label = op.Target
			c.event(Event{Kind: "submitting", Message: fmt.Sprintf("%s = %q (%s)", op.Target, op.Device.Name, op.Device.Driver)})
			e = c.Backend.Set(op.Target, *op.Device)
		} else {
			c.event(Event{Kind: "submitting", Message: fmt.Sprintf("%s = %d", op.Parameter, op.Value)})
			if strings.HasPrefix(op.Parameter, "Recorder.") {
				b, ok := c.Backend.(RecorderBackend)
				if !ok {
					return fail(fmt.Errorf("recorder API unavailable"))
				}
				e = b.SetRecorder(op.Parameter, op.Value)
			} else {
				e = numeric.SetNumber(op.Parameter, op.Value)
			}
		}
		if e != nil {
			return fail(e)
		}
		deadline := c.Clock.Now().Add(c.Config.Verify())
		for {
			if e = ctx.Err(); e != nil {
				return fail(e)
			}
			s, e = c.observe(true)
			// Before any subsequent write, confirm device completion against a full
			// inventory. Pending retries only need refreshed assignment parameters.
			if e == nil && op.Device != nil && matches(op, s) {
				s, e = c.observe(false)
			}
			if e != nil {
				return fail(e)
			}
			if routing.InventoryKey(s) != p.Topology.InventoryKey {
				return fail(ErrPlanChanged)
			}
			if e = check(s, label); e != nil {
				return fail(e)
			}
			if matches(op, s) {
				if p.Topology.Voice != nil {
					if op.Device != nil {
						expectedNames[op.Target] = op.Device.Name
					} else {
						expectedNumbers[op.Parameter] = float32(op.Value)
					}
				}
				verified++
				c.event(Event{Kind: "verified", Message: label})
				break
			}
			remain := deadline.Sub(c.Clock.Now())
			if remain <= 0 {
				return fail(fmt.Errorf("timed out verifying %s", label))
			}
			if e = c.Clock.Wait(ctx, min(verificationInterval(op), remain)); e != nil {
				return fail(e)
			}
		}
	}
	finalSnapshot, e := c.observe(false)
	if e != nil {
		return fail(e)
	}
	if routing.InventoryKey(finalSnapshot) != p.Topology.InventoryKey {
		return fail(ErrPlanChanged)
	}
	final, e := routing.Build(c.Config, finalSnapshot)
	if e != nil {
		return fail(e)
	}
	if final.HasChanges() {
		return fail(fmt.Errorf("routing has not converged; a fresh plan is required"))
	}
	if final.HasUnresolved() {
		return fail(fmt.Errorf("unresolved routes: %v", final.Topology.Unresolved))
	}
	return nil
}
