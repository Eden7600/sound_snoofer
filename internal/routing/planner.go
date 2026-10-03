package routing

import (
	"encoding/json"
	"fmt"
	"sort"
	"voice-snooter/internal/config"
	"voice-snooter/internal/model"
)

type Decision struct {
	Target  string        `json:"target"`
	Current string        `json:"current"`
	Desired *model.Device `json:"desired,omitempty"`
	Pattern string        `json:"pattern,omitempty"`
	Change  bool          `json:"change"`
	Reasons []string      `json:"reasons"`
}
type Plan struct {
	Edition   int        `json:"edition"`
	Decisions []Decision `json:"decisions"`
	Topology  *Topology  `json:"topology,omitempty"`
}

func Build(c config.Config, s model.Snapshot) (Plan, error) {
	p := Plan{Edition: s.Edition}
	if c.StateError != "" {
		return p, fmt.Errorf("%s", c.StateError)
	}
	if e := c.ValidateEdition(s.Edition); e != nil {
		return p, e
	}
	if c.Studio != nil {
		return buildStudio(c, s)
	}
	for _, r := range c.Routes {
		current, ok := s.Assignments[r.Target]
		if !ok {
			return Plan{}, fmt.Errorf("snapshot missing assignment for %s", r.Target)
		}
		d := Decision{Target: r.Target, Current: current, Reasons: []string{}}
		slot, _ := model.ParseSlot(r.Target)
		for _, candidate := range r.Candidates {
			if candidate.Regex == nil {
				return Plan{}, fmt.Errorf("configuration patterns must be validated first")
			}
			matches := []model.Device{}
			for _, dev := range s.Devices {
				if dev.Available && dev.Direction == slot.Direction && dev.Driver == candidate.Driver && candidate.Regex.MatchString(dev.Name) {
					matches = append(matches, dev)
				}
			}
			if len(matches) != 1 {
				d.Reasons = append(d.Reasons, fmt.Sprintf("%q matched %d eligible devices", candidate.Pattern, len(matches)))
				continue
			}
			chosen := matches[0]
			d.Desired = &chosen
			d.Pattern = candidate.Pattern
			d.Change = d.Current != chosen.Name
			d.Reasons = append(d.Reasons, "first unique available match in priority order")
			break
		}
		if d.Desired == nil {
			d.Reasons = append(d.Reasons, "unresolved: leave assignment unchanged")
		}
		p.Decisions = append(p.Decisions, d)
	}
	// Inputs first, outputs next, engine clock A1 last.
	sort.SliceStable(p.Decisions, func(i, j int) bool { return order(p.Decisions[i].Target) < order(p.Decisions[j].Target) })
	return p, nil
}
func order(target string) int {
	s, _ := model.ParseSlot(target)
	if s.Direction == "input" {
		return s.Index
	}
	if s.Index == 0 {
		return 100
	}
	return 10 + s.Index
}

// Key excludes current assignment and diagnostic text: only desired routing and
// edition stability count toward debounce. JSON avoids delimiter collisions.
func (p Plan) Key() string {
	if p.Topology != nil {
		return p.Topology.Key()
	}
	type selection struct {
		Target string
		Device *model.Device
	}
	v := struct {
		Edition  int
		Selected []selection
	}{Edition: p.Edition}
	for _, d := range p.Decisions {
		v.Selected = append(v.Selected, selection{d.Target, d.Desired})
	}
	b, _ := json.Marshal(v)
	return string(b)
}
func (p Plan) HasChanges() bool {
	if p.Topology != nil {
		for _, op := range p.Topology.Operations {
			if op.Change {
				return true
			}
		}
		return false
	}
	for _, d := range p.Decisions {
		if d.Change {
			return true
		}
	}
	return false
}
func (p Plan) HasUnresolved() bool {
	if p.Topology != nil {
		return len(p.Topology.Unresolved) > 0
	}
	for _, d := range p.Decisions {
		if d.Desired == nil {
			return true
		}
	}
	return false
}
