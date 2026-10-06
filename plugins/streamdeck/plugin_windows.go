//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Plugin returns the optional physical control surface.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "streamdeck", Validate: validateSettings, Label: "Stream Deck", Defaults: snoofer.MarshalSettings(Settings{Layout: DefaultLayout()}), Start: start}
}
func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
	return startWithSurface(ctx, s, raw, device.StartSurface)
}
func startWithSurface(ctx context.Context, s snoofer.Services, raw json.RawMessage, surface func(context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{})) (snoofer.Instance, error) {
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return nil, err
	}
	if err := settings.Layout.Validate(s.Controls.Snapshot()); err != nil {
		return nil, err
	}
	for serial, l := range settings.Serials {
		if serial == "" {
			return nil, fmt.Errorf("serial must be nonempty")
		}
		if err := l.Validate(s.Controls.Snapshot()); err != nil {
			return nil, err
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	frames, events, deviceDone := surface(runCtx)
	go func() {
		defer close(i.done)
		defer s.Controls.Remove("streamdeck")
		commands := make(chan snoofer.Request, 8)
		draft := settings.Layout.clone()
		page := settings.Layout.Home
		editPage := draft.Home
		slot := 0
		shared := false
		serial := ""
		editSerial := "Default"
		status := "" // Layout editor feedback only; device state lives in hardware.
		var hardware deviceLink
		generation := uint64(1)
		dirty := false
		editorEpoch := uint64(1)
		// Dial meters refresh faster than static tiles; the cadence follows
		// whether the shown page has any meter.
		ticker := time.NewTicker(idleRefresh)
		defer ticker.Stop()
		refresh := idleRefresh
		var vus [Dials]vuMeter
		active := func() Layout {
			if l, ok := settings.Serials[serial]; ok {
				return l
			}
			return settings.Layout
		}
		editorLayout := func() Layout {
			if editSerial != "Default" {
				return settings.Serials[editSerial]
			}
			return settings.Layout
		}
		var shown map[string]snoofer.Control
		var displayed Layout
		var displayedPage Page
		lastBindings := ""
		publish := func() {
			all := s.Controls.Snapshot()
			shown = map[string]snoofer.Control{}
			for _, c := range all {
				shown[c.ID] = c
			}
			l := active().expanded(all)
			p := l.effective(page)
			displayed, displayedPage = l, p
			page = p.ID
			var signature strings.Builder
			for _, b := range p.Keys {
				fmt.Fprintf(&signature, "%s:%d;", b.Control, shown[b.Control].Revision)
			}
			for _, b := range p.Dials {
				fmt.Fprintf(&signature, "%s:%d;", b.Control, shown[b.Control].Revision)
			}
			if signature.String() != lastBindings {
				generation++
				lastBindings = signature.String()
			}
			frame := device.Frame{Generation: generation}
			tile := func(b Binding) device.Tile {
				c, ok := shown[b.Control]
				return bindingTile(b, c, ok, time.Now())
			}
			for n, b := range p.Keys {
				frame.Keys[n] = tile(b)
			}
			now := time.Now()
			metered := false
			for n, b := range p.Dials {
				frame.Dials[n] = vus[n].apply(b.Control, withPosition(tile(b)), now)
				metered = metered || frame.Dials[n].Meter
			}
			want := idleRefresh
			if metered {
				want = meterRefresh
			}
			if want != refresh {
				refresh = want
				ticker.Reset(refresh)
			}
			names := l.pageNames(p.ID)
			frame.Dials[5] = device.Tile{Label: names[0], Value: names[1], Icon: names[2]}
			select {
			case <-frames:
			default:
			}
			select {
			case frames <- frame:
			default:
			}
			selected := draft.index(editPage)
			if selected < 0 {
				selected = 0
				editPage = draft.Pages[0].ID
			}
			b := draft.Pages[selected].Keys[0]
			if slot < Keys {
				b = draft.Pages[selected].Keys[slot]
				if shared {
					b = draft.SharedKeys[slot]
				}
			} else {
				b = draft.Pages[selected].Dials[slot-Keys]
				if shared {
					b = draft.SharedDials[slot-Keys]
				}
			}
			ids := []string{""}
			for _, c := range all {
				if strings.HasPrefix(c.ID, "streamdeck.") {
					continue
				}
				if (slot < Keys && (slices.Contains(c.Operations, "press") || slices.Contains(c.Operations, "set"))) || (slot >= Keys && slices.Contains(c.Operations, "adjust")) {
					ids = append(ids, c.ID)
				}
			}
			if b.Control != "" && !slices.Contains(ids, b.Control) {
				ids = append(ids, b.Control)
			}
			pages := []string{}
			for _, p := range draft.Pages {
				pages = append(pages, p.ID)
			}
			slots := []string{}
			for n := 1; n <= Keys; n++ {
				slots = append(slots, fmt.Sprintf("Key %d", n))
			}
			for n := 1; n <= Dials; n++ {
				slots = append(slots, fmt.Sprintf("Dial %d", n))
			}
			profiles := []string{"Default"}
			for id := range settings.Serials {
				profiles = append(profiles, id)
			}
			slices.Sort(profiles[1:])
			list := []snoofer.Control{}
			add := func(id, label, kind, value string, options []string, ops ...string) {
				list = append(list, snoofer.Control{ID: "streamdeck." + id, Epoch: editorEpoch, Label: label, Kind: kind, Group: "Stream Deck configurator", Value: value, Options: options, Operations: ops, Available: true})
			}
			add("profile", "Device layout", "selection", editSerial, profiles, "set")
			add("page", "Page", "selection", editPage, pages, "set")
			add("name", "Rename page", "text", draft.Pages[selected].Name, nil, "set")
			add("auto-controls", "Auto controls prefix", "text", draft.Pages[selected].AutoControls, nil, "set")
			add("add", "New page (name)", "text", "", nil, "set")
			add("delete", "Delete page", "command", "", nil, "press")
			add("earlier", "Move page earlier", "command", "", nil, "press")
			add("later", "Move page later", "command", "", nil, "press")
			add("home", "Make this Home", "command", draft.Home, nil, "press")
			add("slot", "Position", "selection", slots[slot], slots, "set")
			value := "Off"
			if shared {
				value = "On"
			}
			add("shared", "Shared across pages", "toggle", value, nil, "press")
			add("binding", "Binding (empty clears)", "selection", b.Control, ids, "set")
			preview := draft.expanded(all).effective(editPage)
			effective := preview.Keys[0]
			if slot < Keys {
				effective = preview.Keys[slot]
			} else {
				effective = preview.Dials[slot-Keys]
			}
			reason := effective.Label
			if c, ok := shown[effective.Control]; ok {
				reason = c.Label + " " + c.Value
			} else if effective.Control != "" {
				reason += " — Unavailable"
			}
			add("preview", "Effective position", "status", reason, nil)
			viewData, viewErr := json.Marshal(editorView(draft, editPage, slot, dirty, all))
			if viewErr != nil {
				status = viewErr.Error()
			} else {
				list[len(list)-1].ViewData = viewData
			}
			note := status
			if dirty {
				note = "Unsaved — " + note
			}
			add("status", "Layout status", "status", note, nil)
			add("save", "Save layout", "command", "", nil, "press")
			add("cancel", "Cancel edits", "command", "", nil, "press")
			for n := range list {
				if list[n].ID == "streamdeck.page" {
					list[n].OptionLabels = map[string]string{}
					for _, p := range draft.Pages {
						list[n].OptionLabels[p.ID] = p.Name
					}
				}
				if list[n].ID == "streamdeck.binding" {
					list[n].OptionLabels = map[string]string{"": "Clear binding"}
					for _, c := range all {
						list[n].OptionLabels[c.ID] = c.Label + " (" + c.ID + ")"
					}
				}
			}
			list = append(list, hardware.report(time.Now()))
			_ = s.Controls.Publish("streamdeck", list, func(ctx context.Context, r snoofer.Request) error {
				select {
				case commands <- r:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				default:
					return fmt.Errorf("layout editor busy")
				}
			})
		}
		publish()
		for {
			select {
			case <-runCtx.Done():
				<-deviceDone
				return
			case <-ticker.C:
				publish()
			case event, ok := <-events:
				if runCtx.Err() != nil {
					<-deviceDone
					return
				}
				if !ok {
					return
				}
				if event.Error != "" {
					hardware.failed(event.Error, time.Now())
					generation++
					publish()
					continue
				}
				if event.Connected {
					hardware.connectedTo(event.Serial, time.Now())
					serial = event.Serial
					page = active().Home
					generation++
					publish()
					continue
				}
				if event.Generation != generation {
					continue
				}
				hardware.link.Activity(time.Now())
				if event.Encoder == 5 {
					if event.Press {
						page = active().Home
					} else {
						page = displayed.next(page, event.Delta)
					}
					generation++
					publish()
					continue
				}
				// Dispatch the binding actually displayed, even if the catalogue changed.
				p := displayedPage
				var b Binding
				if event.Encoder >= 0 {
					if event.Encoder >= Dials {
						continue
					}
					b = p.Dials[event.Encoder]
				} else {
					if event.Key < 0 || event.Key >= Keys {
						continue
					}
					b = p.Keys[event.Key]
				}
				c, ok := shown[b.Control]
				if !ok || !c.Available || c.Hidden { // Hidden bindings are blank and inert.
					continue
				}
				r := snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
				if event.Delta != 0 {
					r.Operation = "adjust"
					r.Delta = event.Delta
				} else if !slices.Contains(c.Operations, "press") && len(c.Options) > 0 {
					r.Operation = "set"
					n := slices.Index(c.Options, c.Value)
					r.Value = c.Options[(n+1)%len(c.Options)]
				}
				if err := s.Controls.Dispatch(runCtx, r); err != nil {
					status = err.Error()
				}
			case r := <-commands:
				if runCtx.Err() != nil {
					<-deviceDone
					return
				}
				current := false
				for _, c := range s.Controls.Snapshot() {
					if c.ID == r.ID && c.Revision == r.Revision {
						current = true
						break
					}
				}
				if !current {
					status = "Editor changed; try again"
					continue
				}
				editorEpoch++
				n := draft.index(editPage)
				if n < 0 {
					continue
				}
				err := error(nil)
				switch strings.TrimPrefix(r.ID, "streamdeck.") {
				case "profile":
					if dirty {
						err = fmt.Errorf("save or cancel before switching layouts")
					} else {
						editSerial = r.Value
						draft = editorLayout().clone()
						editPage = draft.Home
					}
				case "page":
					editPage = r.Value
				case "auto-controls":
					draft.Pages[draft.index(editPage)].AutoControls = strings.TrimSpace(r.Value)
					dirty = true
				case "name":
					if strings.TrimSpace(r.Value) == "" {
						err = fmt.Errorf("name is required")
					} else {
						draft.Pages[n].Name = r.Value
						dirty = true
					}
				case "add":
					if strings.TrimSpace(r.Value) == "" {
						err = fmt.Errorf("name is required")
						break
					}
					id := fmt.Sprintf("page-%d", time.Now().UnixNano())
					draft.Pages = append(draft.Pages, Page{ID: id, Name: r.Value})
					editPage = id
					dirty = true
				case "delete":
					err = draft.remove(editPage)
					if err == nil {
						editPage = draft.Home
						dirty = true
					}
				case "earlier":
					if n > 0 {
						draft.Pages[n], draft.Pages[n-1] = draft.Pages[n-1], draft.Pages[n]
						dirty = true
					}
				case "later":
					if n+1 < len(draft.Pages) {
						draft.Pages[n], draft.Pages[n+1] = draft.Pages[n+1], draft.Pages[n]
						dirty = true
					}
				case "home":
					draft.Home = editPage
					dirty = true
				case "slot":
					parts := strings.Fields(r.Value)
					if len(parts) == 2 {
						number, e := strconv.Atoi(parts[1])
						if e == nil {
							slot = number - 1
							if parts[0] == "Dial" {
								slot += Keys
							}
						}
					}
				case "shared":
					shared = !shared
				case "binding":
					next := draft.clone()
					binding := Binding{Control: r.Value}
					if c, ok := shown[r.Value]; ok {
						binding.Label = c.Label
					} else {
						binding.Label = r.Value
					}
					if slot < Keys {
						if shared {
							next.SharedKeys[slot] = binding
						} else {
							next.Pages[n].Keys[slot] = binding
						}
					} else {
						if shared {
							next.SharedDials[slot-Keys] = binding
						} else {
							next.Pages[n].Dials[slot-Keys] = binding
						}
					}
					err = next.Validate(s.Controls.Snapshot())
					if err == nil {
						draft = next
						dirty = true
					}
				case "cancel":
					draft = editorLayout().clone()
					editPage = draft.Home
					dirty = false
				case "save":
					err = draft.Validate(s.Controls.Snapshot())
					if err != nil {
						break
					}
					next := settings
					next.Serials = map[string]Layout{}
					for k, v := range settings.Serials {
						next.Serials[k] = v
					}
					if editSerial == "Default" {
						next.Layout = draft.clone()
					} else {
						next.Serials[editSerial] = draft.clone()
					}
					data, e := json.Marshal(next)
					if e != nil {
						err = e
						break
					}
					if s.SaveSettings == nil {
						err = fmt.Errorf("settings persistence unavailable")
						break
					}
					err = s.SaveSettings("streamdeck", raw, data)
					if err == nil {
						settings = next
						raw = data
						dirty = false
						generation++
						status = "Saved"
					}
				}
				if err != nil {
					status = err.Error()
				}
				publish()
			}
		}
	}()
	return i, nil
}
