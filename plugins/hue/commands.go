package hue

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

// actionTarget is command identity, independently of changing observed values.
type actionTarget struct {
	bridge, sync                                int
	settings, room                              uint64
	group, scene, resources, endpoint, bridgeID string
	syncing, chosen                             bool
}

type actionContext struct {
	target actionTarget
	epoch  uint64
	effect string
}
type command struct {
	request  snoofer.Request
	target   actionTarget
	want     bool
	effect   string
	sensors  []motionSensor
	complete snoofer.Completion
}

// resourceIdentity includes graph membership and each resource's current lifetime.
func (w *worker) resourceIdentity(ids ...string) string {
	seen := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		r := w.model.resources[id]
		if r.Type == "room" || r.Type == "zone" || r.Type == "device" {
			for _, ref := range r.Children {
				visit(ref.RID)
			}
			for _, ref := range r.Services {
				visit(ref.RID)
			}
		}
	}
	for _, id := range ids {
		visit(id)
	}
	keys := make([]string, 0, len(seen))
	for id := range seen {
		keys = append(keys, id)
	}
	slices.Sort(keys)
	var out strings.Builder
	for _, id := range keys {
		fmt.Fprintf(&out, "%q:%d;", id, w.model.lifetimes[id])
	}
	return out.String()
}

func (w *worker) capture(id string) command {
	c := command{request: snoofer.Request{ID: id}}
	switch {
	case id == "hue.pair":
		address := w.settings.Address
		bridgeID := ""
		if w.connected || w.status == "Not paired" {
			bridgeID = w.bridgeInfo.BridgeID
			if address == "" {
				if w.connected {
					address = w.lastAddress
				} else {
					address = w.bridgeAddress
				}
			}
		}
		c.target = actionTarget{bridge: w.generation, settings: w.settingsEpoch, endpoint: address, bridgeID: bridgeID}
	case id == "hue.group" || id == "hue.rooms" || id == "hue.sort":
		c.target.settings = w.settingsEpoch
		if id != "hue.sort" {
			c.target.bridge = w.generation
			var ids []string
			for _, g := range w.model.groups() {
				ids = append(ids, g.ID)
			}
			c.target.resources = w.resourceIdentity(ids...)
		}
	case strings.HasPrefix(id, "hue.sync"):
		c.target.sync = w.sync.generation
		c.want = !w.syncing()
		if id != "hue.sync" {
			c.target.syncing = w.syncing()
		}
	case strings.HasPrefix(id, "hue.scene-") || strings.HasPrefix(id, "hue.room-scene-"):
		c.target.bridge = w.generation
		c.target.scene = w.sceneForControl(id)
		if strings.HasPrefix(id, "hue.room-scene-") {
			c.target.scene = w.sceneForSlot(id)
			c.target.room, c.target.group = w.roomEpoch, w.settings.Group
		}
		scene := w.model.resources[c.target.scene]
		group := ""
		if scene.Group != nil {
			group = scene.Group.RID
		}
		c.target.resources = w.resourceIdentity(c.target.scene, group)
		c.target.chosen = w.settings.chosen(group)
		c.target.syncing = w.syncing()
	default:
		c.target = actionTarget{bridge: w.generation, room: w.roomEpoch, group: w.settings.Group}
		c.target.resources = w.resourceIdentity(w.settings.Group)
		switch id {
		case "hue.brightness":
			c.target.syncing = w.syncing()
			if c.target.syncing {
				c.target.sync = w.sync.generation
			}
			on := w.view().On
			if w.group.context == c.target {
				on, _ = w.effectiveOn(w.view())
			}
			c.want = !on
		case "hue.motion":
			c.sensors = w.roomSensors()
			c.want = motionValue(c.sensors) != "On"
		default:
			if d, ok := findGroupDial(id); ok {
				c.target.syncing = w.syncing()
				c.want = !summarize(d.lights(w.roomLightsNow())).AnyOn
				for _, dial := range groupDials {
					if staged, ok := w.stages[dial.id]; ok {
						c.effect += fmt.Sprintf("%s:%g;", dial.id, staged.target)
					}
				}
			}
		}
	}
	if c.target.syncing {
		c.target.sync = w.sync.generation
	}
	return c
}

func rejected(detail string) snoofer.Outcome {
	return snoofer.Outcome{Status: snoofer.Rejected, Detail: detail}
}
func submitted(detail string) snoofer.Outcome {
	return snoofer.Outcome{Status: snoofer.Submitted, Detail: detail}
}
func processed(detail string) snoofer.Outcome {
	return snoofer.Outcome{Status: snoofer.Processed, Detail: detail}
}
func operationOutcome(err error) snoofer.Outcome {
	if err == nil {
		return processed("Hue request processed")
	}
	status := snoofer.Failed
	var network net.Error
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.As(err, &network) {
		status = snoofer.Unknown
	}
	return snoofer.Outcome{Status: status, Detail: err.Error()}
}

func (w *worker) handle(ctx context.Context, c command) {
	r := c.request
	finish := func(o snoofer.Outcome) {
		if c.complete != nil {
			c.complete(o)
		}
	}
	if ctx.Err() != nil {
		finish(operationOutcome(ctx.Err()))
		return
	}
	if !w.services.Live {
		finish(rejected("Hue unavailable in preview"))
		return
	}
	if current := w.capture(r.ID); c.target != current.target || (r.Operation == "press" && c.effect != current.effect) {
		finish(rejected("Hue target changed; try again"))
		return
	}
	var err error
	switch {
	case r.ID == "hue.pair":
		if w.pairing {
			finish(rejected("Hue pairing already in progress"))
			return
		}
		w.startPairing(ctx, c)
		return
	case r.ID == "hue.group":
		found := false
		for _, g := range w.chosenGroups() {
			found = found || g.ID == r.Value
		}
		if !w.connected || !found {
			finish(rejected("Hue room unavailable"))
			return
		}
		err = w.selectGroup(r.Value)
	case r.ID == "hue.rooms":
		if !w.connected {
			finish(rejected("Hue bridge disconnected"))
			return
		}
		err = w.setRooms(r.Value)
		if errors.Is(err, errNoRooms) {
			finish(rejected(err.Error()))
			return
		}
	case r.ID == "hue.sort":
		if !slices.Contains(sceneSorts, r.Value) {
			finish(rejected("Invalid scene sort"))
			return
		}
		err = w.setSceneSort(r.Value)
	case r.ID == "hue.brightness":
		err = w.adjustBrightness(r, c.want)
		if err != nil {
			finish(rejected(err.Error()))
		} else {
			finish(submitted("Brightness queued"))
		}
		return
	case r.ID == "hue.motion":
		w.toggleMotion(ctx, c)
		return
	case r.ID == "hue.decor" || r.ID == "hue.mixed" || r.ID == "hue.saturation":
		d, _ := findGroupDial(r.ID)
		err = w.adjustGroupDial(d, r, c.want)
		if err != nil {
			finish(rejected(err.Error()))
		} else {
			finish(submitted("Light changes queued"))
		}
		return
	case strings.HasPrefix(r.ID, "hue.sync"):
		finish(w.handleSync(r, c.want))
		return
	case c.target.scene != "":
		w.recallScene(ctx, c)
		return
	default:
		finish(rejected("Hue control unavailable"))
		return
	}
	if err != nil {
		finish(operationOutcome(err))
	} else {
		finish(processed("Saved"))
	}
}

func (w *worker) publish(ctx context.Context) {
	if w.group.target != "" && w.group.context != w.capture("hue.brightness").target {
		w.group = groupRequest{}
	}
	for id, staged := range w.stages {
		if staged.context != w.capture(id).target {
			delete(w.stages, id)
		}
	}
	controls := w.controls()
	commands := make(map[string]command, len(controls))
	if w.contexts == nil {
		w.contexts = map[string]actionContext{}
	}
	current := make(map[string]actionContext, len(controls))
	for i := range controls {
		control := &controls[i]
		if len(control.Operations) == 0 {
			continue
		}
		c := w.capture(control.ID)
		identity := w.contexts[control.ID]
		if identity.epoch == 0 || identity.target != c.target || identity.effect != c.effect {
			w.nextContext++
			identity = actionContext{target: c.target, epoch: w.nextContext, effect: c.effect}
		}
		current[control.ID] = identity
		control.Epoch = identity.epoch
		commands[control.ID] = c
	}
	w.contexts = current
	_ = w.services.Controls.Publish("hue", controls, func(dispatchCtx context.Context, r snoofer.Request) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		c := commands[r.ID]
		c.request = r
		c.complete = r.ClaimCompletion(ctx)
		select {
		case w.requests <- c:
			return nil
		case <-dispatchCtx.Done():
			return dispatchCtx.Err()
		default:
			return errors.New("hue queue full")
		}
	})
}
