//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"sound-snoofer/internal/presence"
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

// watchPresence observes the session lock and monitors; tests replace it.
var watchPresence = presence.Watch

// surface is the deck hardware: it shows frames per deck and reports input,
// connections and failures until its context ends.
type surface interface {
	Show(deck string, f device.Frame)
	Events() <-chan device.Event
	Done() <-chan struct{}
}

// deckState is a connected deck's page, drawing and input. The plugin
// goroutine owns it.
type deckState struct {
	id, serial    string
	model         device.Model
	page          string
	displayed     Layout                     // Expanded layout as drawn.
	displayedPage Page                       // Page as drawn.
	shown         map[string]snoofer.Control // Controls as drawn, with this deck's go-to and scroll keys.
	generation    uint64                     // Input carrying another generation is stale.
	bindings      string                     // Drawn bindings and revisions.
	vus           [Dials]vuMeter
	held          *heldKey
}

// heldKey is a key down on a control that offers hold, or on a page key. A
// release before holdAfter sends press (a page key steps step pages);
// reaching holdAfter sends hold (a page key goes Home). The request uses
// the control as shown when the key went down.
type heldKey struct {
	key   int
	at    time.Time
	req   snoofer.Request
	step  int // Page keys only.
	fired bool
}

// pageKeys are the keys that page a deck without dials, with a tap's step:
// the Pages key, or the Neo's touch buttons (left back, right forward).
func pageKeys(m device.Model) map[int]int {
	if sh := shapeOf(m.ID); sh.pagesKey >= 0 {
		return map[int]int{sh.pagesKey: 1}
	}
	if m.TouchKeys == 2 {
		return map[int]int{m.Keys(): -1, m.Keys() + 1: 1}
	}
	return nil
}

// Editor targets: a model's layout or a remembered deck's own layout.
const (
	modelTarget  = "model:"
	serialTarget = "serial:"
)

func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
	return startWithSurface(ctx, s, raw, func(ctx context.Context) surface { return device.StartSurface(ctx) })
}
func startWithSurface(ctx context.Context, s snoofer.Services, raw json.RawMessage, startSurface func(context.Context) surface) (snoofer.Instance, error) {
	settings, err := decodeSettings(raw)
	if err != nil {
		return nil, err
	}
	if err := settings.validate(s.Controls.Snapshot()); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	hardwareSurface := startSurface(runCtx)
	events, deviceDone := hardwareSurface.Events(), hardwareSurface.Done()
	go func() {
		defer close(i.done)
		defer s.Controls.Remove("streamdeck")
		commands := make(chan snoofer.Request, 8)
		// Editor state: target is the layout being edited; until the user
		// picks one (chosen) it follows the first deck to connect.
		target := modelTarget + plusXLModel
		chosen := false
		draft := settings.Layout.clone()
		editPage := draft.Home
		slot := 0
		shared := false
		editDeck := "" // Serial of the remembered deck the deck commands apply to.
		status := ""   // Layout editor feedback only; device state lives in links.
		dirty := false
		editorEpoch := uint64(1)
		decks := map[string]*deckState{}  // Connected decks by device ID.
		links := map[string]*deviceLink{} // Connection reports by device ID, kept across reconnects.
		var discovery deviceLink          // Reported while no deck is known.
		focus := ""                       // The deck used most recently; GUI page requests act on it.
		display := ""                     // Display state for the reports.
		// away: the session is locked or the monitors are off, so every deck
		// is dark and ignores input. The watcher stops before the plugin is done.
		away := false
		presenceCtx, stopPresence := context.WithCancel(runCtx)
		presenceStates, presenceDone := watchPresence(presenceCtx)
		defer func() {
			stopPresence()
			<-presenceDone
		}()
		// Dial meters and animated keys refresh faster than static tiles; the
		// cadence follows whether any shown page has either.
		ticker := time.NewTicker(idleRefresh)
		defer ticker.Stop()
		refresh := idleRefresh
		// resetFocus sends reset to every control that offers it.
		resetFocus := func() {
			for _, c := range s.Controls.Snapshot() {
				if c.Available && slices.Contains(c.Operations, "reset") {
					if err := s.Controls.Dispatch(runCtx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "reset"}); err != nil {
						status = err.Error()
					}
				}
			}
		}
		dispatch := func(r snoofer.Request) {
			if err := s.Controls.Dispatch(runCtx, r); err != nil {
				status = err.Error()
			}
		}
		layoutOf := func(d *deckState) Layout {
			return settings.layoutFor(d.model.ID, d.serial)
		}
		editorLayout := func() Layout {
			if serial, ok := strings.CutPrefix(target, serialTarget); ok {
				return settings.Serials[serial]
			}
			return settings.modelLayout(strings.TrimPrefix(target, modelTarget))
		}
		// edit switches the editor to a target; callers check dirty first.
		edit := func(next string) {
			target = next
			draft = editorLayout().clone()
			editPage = draft.Home
			slot = 0
		}
		link := func(id string) *deviceLink {
			l, ok := links[id]
			if !ok {
				l = &deviceLink{}
				links[id] = l
			}
			return l
		}
		focused := func() *deckState {
			if d, ok := decks[focus]; ok {
				return d
			}
			ids := slices.Sorted(maps.Keys(decks))
			if len(ids) == 0 {
				return nil
			}
			return decks[ids[0]]
		}
		// save persists next; the editor draft is unaffected.
		save := func(next Settings) error {
			data, err := json.Marshal(next)
			if err != nil {
				return err
			}
			if s.SaveSettings == nil {
				return fmt.Errorf("settings persistence unavailable")
			}
			if err := s.SaveSettings("streamdeck", raw, data); err != nil {
				return err
			}
			settings, raw = next, data
			for _, d := range decks {
				d.generation++
			}
			return nil
		}
		// remember records a deck by serial the first time it connects. A
		// failed save is reported and retried on its next connection.
		remember := func(serial, model string) {
			if serial == "" {
				return
			}
			if known, ok := settings.Devices[serial]; ok && known.Model == model {
				return
			}
			if own, ok := settings.Serials[serial]; ok && own.model != modelKey(model) {
				status = fmt.Sprintf("Deck %s has a layout for another model; it is not remembered", serial)
				return
			}
			next := settings.clone()
			next.Devices[serial] = Deck{Model: model, Name: next.Devices[serial].Name}
			if err := save(next); err != nil {
				status = "Remember deck: " + err.Error()
			}
		}
		// show moves deck d to page id and, when mirrored, every other deck
		// whose layout has its base page; a deck already on it keeps its set.
		show := func(d *deckState, id string) {
			d.page = id
			d.generation++
			if settings.Paging != pagingMirrored {
				return
			}
			base, _, _ := strings.Cut(id, "~auto~")
			for _, other := range decks {
				current, _, _ := strings.Cut(other.page, "~auto~")
				if other == d || current == base || layoutOf(other).index(base) < 0 {
					continue
				}
				other.page = base
				other.generation++
			}
		}
		// goHome shows Home on d, or on every deck when mirrored.
		goHome := func(d *deckState) {
			for _, each := range decks {
				if each == d || settings.Paging == pagingMirrored {
					each.page = layoutOf(each).Home
					each.generation++
				}
			}
		}
		// drawDeck builds d's frame from the current controls and reports
		// whether it needs the meter cadence.
		drawDeck := func(d *deckState, all []snoofer.Control, base map[string]snoofer.Control, now time.Time) (device.Frame, bool) {
			own := layoutOf(d)
			sh := own.shape()
			l := own.expanded(all)
			p := l.effective(d.page)
			d.displayed, d.displayedPage, d.page = l, p, p.ID
			// This deck's go-to and scroll keys describe the page being drawn;
			// presses on them are handled locally.
			d.shown = maps.Clone(base)
			for _, g := range append(append(gotoControls(own, p.ID), scrollControls(l, p.ID)...), resetFocusControl()) {
				d.shown[g.ID] = g
			}
			var signature strings.Builder
			for _, b := range p.Keys[:sh.keys] {
				fmt.Fprintf(&signature, "%s:%d;", b.Control, d.shown[b.Control].Revision)
			}
			for _, b := range p.Dials[:sh.dials] {
				fmt.Fprintf(&signature, "%s:%d;", b.Control, d.shown[b.Control].Revision)
			}
			if signature.String() != d.bindings {
				d.generation++
				d.bindings = signature.String()
			}
			frame := device.Frame{Generation: d.generation}
			tile := func(b Binding) device.Tile {
				c, ok := d.shown[b.Control]
				return bindingTile(b, c, ok, now)
			}
			// Meters and animated artwork refresh at the meter cadence.
			metered := false
			for n, b := range p.Keys[:sh.keys] {
				frame.Keys[n] = tile(b)
				metered = metered || len(d.shown[b.Control].Animation) > 1
			}
			for n, b := range p.Dials[:sh.dials] {
				frame.Dials[n] = d.vus[n].apply(b.Control, withPosition(tile(b)), now)
				metered = metered || frame.Dials[n].Meter
			}
			for n, timers := range timerPanels(p, sh.dials, d.shown, now) {
				frame.Dials[n].Timers = timers
			}
			names := l.pageNames(p.ID)
			pages := device.Tile{Label: names[0], Value: names[1], Icon: names[2]}
			switch {
			case d.model.Dials > 0:
				frame.Dials[d.model.Dials-1] = pages
			case sh.pagesKey >= 0:
				frame.Keys[sh.pagesKey] = device.Tile{Label: names[1], Icon: "deck-page"}
			default:
				frame.Screen = pages
			}
			frame.Brightness = settings.backlight()
			if away {
				frame = device.Frame{Generation: d.generation, Dark: true}
			}
			return frame, metered
		}
		publish := func() {
			all := s.Controls.Snapshot()
			base := map[string]snoofer.Control{}
			for _, c := range all {
				base[c.ID] = c
			}
			now := time.Now()
			metered := false
			for _, d := range decks {
				frame, animated := drawDeck(d, all, base, now)
				metered = metered || animated
				hardwareSurface.Show(d.id, frame)
			}
			want := idleRefresh
			if metered {
				want = meterRefresh
			}
			if want != refresh {
				refresh = want
				ticker.Reset(refresh)
			}

			sh := draft.shape()
			selected := draft.index(editPage)
			if selected < 0 {
				selected = 0
				editPage = draft.Pages[0].ID
			}
			if slot >= sh.keys+sh.dials || slot == sh.pagesKey {
				slot = 0
			}
			var b Binding
			if slot < sh.keys {
				b = draft.Pages[selected].Keys[slot]
				if shared {
					b = draft.SharedKeys[slot]
				}
			} else {
				b = draft.Pages[selected].Dials[slot-sh.keys]
				if shared {
					b = draft.SharedDials[slot-sh.keys]
				}
			}
			ids := []string{""}
			for _, c := range all {
				// Go-to and scroll keys are the only Stream Deck controls offered as bindings.
				if strings.HasPrefix(c.ID, "streamdeck.") && !strings.HasPrefix(c.ID, gotoPrefix) && !strings.HasPrefix(c.ID, scrollPrefix) && c.ID != resetFocusID {
					continue
				}
				if bindingCompatible(c, slot >= sh.keys) {
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
			for n := 1; n <= sh.keys; n++ {
				if n-1 != sh.pagesKey {
					slots = append(slots, fmt.Sprintf("Key %d", n))
				}
			}
			for n := 1; n <= sh.dials; n++ {
				slots = append(slots, fmt.Sprintf("Dial %d", n))
			}
			slotName := fmt.Sprintf("Key %d", slot+1)
			if slot >= sh.keys {
				slotName = fmt.Sprintf("Dial %d", slot-sh.keys+1)
			}
			targets, targetLabels := []string{}, map[string]string{}
			for _, m := range device.Models() {
				targets = append(targets, modelTarget+m.ID)
				targetLabels[modelTarget+m.ID] = m.Name
			}
			deckLabel := func(serial string) string {
				known := settings.Devices[serial]
				name := known.Name
				if name == "" {
					name = serial
				}
				model := plusXLModel // A deck's own layout without a record is the + XL's.
				if known.Model != "" {
					model = known.Model
				}
				m, _ := device.ModelByID(model)
				return name + " (" + m.Name + ")"
			}
			for _, serial := range slices.Sorted(maps.Keys(settings.Serials)) {
				targets = append(targets, serialTarget+serial)
				targetLabels[serialTarget+serial] = deckLabel(serial)
			}
			remembered := slices.Sorted(maps.Keys(settings.Devices))
			if _, ok := settings.Devices[editDeck]; !ok {
				editDeck = ""
				if d := focused(); d != nil && settings.Devices[d.serial].Model != "" {
					editDeck = d.serial
				} else if len(remembered) > 0 {
					editDeck = remembered[0]
				}
			}
			deckLabels := map[string]string{}
			for _, serial := range remembered {
				deckLabels[serial] = deckLabel(serial)
			}
			list := []snoofer.Control{}
			add := func(id, label, kind, value string, options []string, ops ...string) {
				list = append(list, snoofer.Control{ID: "streamdeck." + id, Epoch: editorEpoch, Label: label, Kind: kind, Group: "Stream Deck Configurator", Value: value, Options: options, Operations: ops, Available: true})
			}
			add("profile", "Device Layout", "selection", target, targets, "set")
			add("page", "Page", "selection", editPage, pages, "set")
			add("name", "Rename Page", "text", draft.Pages[selected].Name, nil, "set")
			add("region-add", "Add Region", "text", "", nil, "set")
			add("region-source", "Region Source", "text", "", nil, "set")
			add("region-clip", "Region Overflow", "text", "", nil, "set")
			add("region-remove", "Remove Region", "text", "", nil, "set")
			add("add", "New Page (Name)", "text", "", nil, "set")
			add("delete", "Delete Page", "command", "", nil, "press")
			add("earlier", "Move Page Earlier", "command", "", nil, "press")
			add("later", "Move Page Later", "command", "", nil, "press")
			add("home", "Make This Home", "command", draft.Home, nil, "press")
			add("slot", "Position", "selection", slotName, slots, "set")
			value := "Off"
			if shared {
				value = "On"
			}
			add("shared", "Shared Across Pages", "toggle", value, nil, "press")
			add("binding", "Binding (Empty Clears)", "selection", b.Control, ids, "set")
			preview := draft.expanded(all).effective(editPage)
			var effective Binding
			if slot < sh.keys {
				effective = preview.Keys[slot]
			} else {
				effective = preview.Dials[slot-sh.keys]
			}
			reason := effective.Label
			if c, ok := base[effective.Control]; ok {
				reason = c.Label + " " + c.Value
			} else if effective.Control != "" {
				reason += " — Unavailable"
			}
			add("preview", "Effective Position", "status", reason, nil)
			deckList := []EditorDeck{}
			for _, serial := range remembered {
				deck := EditorDeck{Serial: serial, Name: settings.Devices[serial].Name, Model: settings.Devices[serial].Model}
				_, deck.Own = settings.Serials[serial]
				for _, d := range decks {
					deck.Connected = deck.Connected || d.serial == serial
				}
				deckList = append(deckList, deck)
			}
			viewData, viewErr := json.Marshal(editorView(draft, editPage, slot, dirty, all, deckList))
			if viewErr != nil {
				status = viewErr.Error()
			} else {
				list[len(list)-1].ViewData = viewData
			}
			note := status
			if dirty {
				note = "Unsaved — " + note
			}
			add("status", "Layout Status", "status", note, nil)
			add("save", "Save Layout", "command", "", nil, "press")
			add("cancel", "Cancel Edits", "command", "", nil, "press")
			own := "Off"
			if _, ok := settings.Serials[editDeck]; ok {
				own = "On"
			}
			add("deck", "Deck", "selection", editDeck, remembered, "set")
			add("deck-name", "Deck Name", "text", settings.Devices[editDeck].Name, nil, "set")
			add("deck-own", "Own Layout", "toggle", own, nil, "press")
			add("deck-forget", "Forget Deck", "command", "", nil, "press")
			paging := settings.Paging
			if paging == "" {
				paging = pagingIndependent
			}
			add("paging", "Paging", "selection", paging, []string{pagingIndependent, pagingMirrored}, "set")
			for n := range list {
				switch list[n].ID {
				case "streamdeck.profile":
					list[n].OptionLabels = targetLabels
				case "streamdeck.deck":
					list[n].OptionLabels = deckLabels
				case "streamdeck.deck-name", "streamdeck.deck-own", "streamdeck.deck-forget":
					list[n].Available = editDeck != ""
				case "streamdeck.paging":
					list[n].OptionLabels = map[string]string{pagingIndependent: "Independent", pagingMirrored: "Mirrored"}
				case "streamdeck.page":
					list[n].OptionLabels = map[string]string{}
					for _, p := range draft.Pages {
						list[n].OptionLabels[p.ID] = p.Name
					}
				case "streamdeck.binding":
					list[n].OptionLabels = map[string]string{"": "Clear Binding"}
					for _, c := range all {
						list[n].OptionLabels[c.ID] = c.Label + " (" + c.ID + ")"
					}
				}
			}
			// The go-to and scroll keys describe the focused deck.
			if d := focused(); d != nil {
				list = append(list, gotoControls(layoutOf(d), d.displayedPage.ID)...)
				list = append(list, scrollControls(d.displayed, d.displayedPage.ID)...)
			} else {
				list = append(list, gotoControls(settings.Layout, settings.Layout.Home)...)
				list = append(list, scrollControls(settings.Layout.expanded(all), settings.Layout.Home)...)
			}
			list = append(list, resetFocusControl())
			known := map[string]bool{}
			for id := range links {
				known[id] = true
			}
			for serial := range settings.Devices {
				known[serial] = true
			}
			if len(known) == 0 {
				list = append(list, discovery.report(discoveryReportID, "", display, now))
			}
			for _, id := range slices.Sorted(maps.Keys(known)) {
				l := link(id)
				if !l.connected && l.serial == "" && !strings.HasPrefix(id, "path:") {
					// Remembered but not seen this session.
					l.serial, l.model = id, settings.Devices[id].Model
				}
				list = append(list, l.report(reportID(id), settings.Devices[l.serial].Name, display, now))
			}
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
				for _, d := range decks {
					h := d.held
					if h == nil || h.fired || time.Since(h.at) < holdAfter {
						continue
					}
					h.fired = true
					if h.step != 0 {
						goHome(d)
						continue
					}
					r := h.req
					r.Operation = "hold"
					dispatch(r)
				}
				publish()
			case st := <-presenceStates:
				if st.Away() != away {
					away = st.Away()
					// A key down across the change never fires.
					for _, d := range decks {
						d.held = nil
						d.generation++
					}
				}
				display = displayState(st)
				publish()
			case event, ok := <-events:
				if runCtx.Err() != nil {
					<-deviceDone
					return
				}
				if !ok {
					return
				}
				now := time.Now()
				if event.Error != "" {
					if event.Device == "" {
						discovery.failed(event.Error, now)
					} else {
						link(event.Device).failed(event.Error, now)
						delete(decks, event.Device)
					}
					publish()
					continue
				}
				if event.Connected {
					m, ok := device.ModelByID(event.Model)
					if !ok {
						continue
					}
					link(event.Device).connectedTo(event.Serial, m.ID, event.Product, now)
					d := &deckState{id: event.Device, serial: event.Serial, model: m, generation: 1}
					decks[d.id] = d
					d.page = layoutOf(d).Home
					if _, ok := decks[focus]; !ok {
						focus = d.id
					}
					remember(d.serial, m.ID)
					if !chosen && !dirty {
						edit(modelTarget + m.ID)
						if _, own := settings.Serials[d.serial]; own {
							edit(serialTarget + d.serial)
						}
					}
					publish()
					continue
				}
				d, ok := decks[event.Device]
				if !ok {
					continue
				}
				// A dark deck is inert: nothing is dispatched, navigated or held.
				if away {
					continue
				}
				// Releases match the held key by position, even if the layout
				// generation changed while it was down.
				if event.Release {
					if h := d.held; h != nil && h.key == event.Key {
						d.held = nil
						switch {
						case h.fired:
						case h.step != 0:
							show(d, d.displayed.next(d.page, h.step))
							publish()
						default:
							dispatch(h.req)
						}
					}
					continue
				}
				if event.Generation != d.generation {
					continue
				}
				link(d.id).link.Activity(now)
				focus = d.id
				if event.Encoder >= 0 && event.Encoder == d.model.Dials-1 {
					if event.Press {
						goHome(d)
					} else {
						show(d, d.displayed.next(d.page, event.Delta))
					}
					publish()
					continue
				}
				if step, ok := pageKeys(d.model)[event.Key]; ok && event.Encoder < 0 {
					d.held = &heldKey{key: event.Key, at: now, step: step}
					continue
				}
				// Dispatch the binding actually displayed, even if the catalogue changed.
				sh := d.displayed.shape()
				p := d.displayedPage
				var b Binding
				if event.Encoder >= 0 {
					if event.Encoder >= sh.dials {
						continue
					}
					b = p.Dials[event.Encoder]
				} else {
					if !sh.usable(event.Key) {
						continue
					}
					b = p.Keys[event.Key]
				}
				c, ok := d.shown[b.Control]
				if !ok || !c.Available || c.Hidden || c.Inactive { // Hidden bindings are blank and inactive ones grey; both are inert.
					continue
				}
				if page, ok := strings.CutPrefix(c.ID, gotoPrefix); ok {
					show(d, page)
					publish()
					continue
				}
				if c.ID == scrollPrefix+"up" || c.ID == scrollPrefix+"down" {
					// Overflow sets scroll this deck only.
					d.page = d.displayed.scroll(d.page, scrollDelta(c.ID))
					d.generation++
					publish()
					continue
				}
				if c.ID == resetFocusID {
					resetFocus()
					continue
				}
				if c.Kind == "status" {
					continue // Display-only keys never create actions or pending holds.
				}
				r := snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
				if event.Encoder < 0 && slices.Contains(c.Operations, "hold") {
					// Tap or hold is known only later; other keys act at once.
					d.held = &heldKey{key: event.Key, at: now, req: r}
					continue
				}
				if event.Delta != 0 {
					r.Operation = "adjust"
					r.Delta = event.Delta
				} else if !slices.Contains(c.Operations, "press") && len(c.Options) > 0 {
					r.Operation = "set"
					n := slices.Index(c.Options, c.Value)
					r.Value = c.Options[(n+1)%len(c.Options)]
				}
				dispatch(r)
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
				if page, ok := strings.CutPrefix(r.ID, gotoPrefix); ok {
					// Switches the focused deck only; the layout and Home are unchanged.
					if d := focused(); d != nil && layoutOf(d).index(page) >= 0 {
						show(d, page)
						publish()
					}
					continue
				}
				if r.ID == scrollPrefix+"up" || r.ID == scrollPrefix+"down" {
					if d := focused(); d != nil {
						d.page = d.displayed.scroll(d.page, scrollDelta(r.ID))
						d.generation++
						publish()
					}
					continue
				}
				if r.ID == resetFocusID {
					resetFocus()
					continue
				}
				editorEpoch++
				n := draft.index(editPage)
				if n < 0 {
					continue
				}
				sh := draft.shape()
				err := error(nil)
				switch strings.TrimPrefix(r.ID, "streamdeck.") {
				case "profile":
					serial, isSerial := strings.CutPrefix(r.Value, serialTarget)
					model, isModelTarget := strings.CutPrefix(r.Value, modelTarget)
					_, isModel := device.ModelByID(model)
					_, own := settings.Serials[serial]
					switch {
					case dirty:
						err = fmt.Errorf("save or cancel before switching layouts")
					case !(isSerial && own) && !(isModelTarget && isModel):
						err = fmt.Errorf("unknown layout")
					default:
						chosen = true
						edit(r.Value)
					}
				case "page":
					editPage = r.Value
				case "region-add", "region-source", "region-clip", "region-remove":
					next, e := draft.editRegion(n, strings.TrimPrefix(strings.TrimPrefix(r.ID, "streamdeck."), "region-"), r.Value)
					if e == nil {
						e = next.Validate(s.Controls.Snapshot())
					}
					if e != nil {
						err = e
						break
					}
					draft = next
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
					if draft.Home != editPage {
						draft.Home = editPage
						dirty = true
					}
				case "slot":
					parts := strings.Fields(r.Value)
					if len(parts) == 2 {
						number, e := strconv.Atoi(parts[1])
						switch {
						case e != nil:
						case parts[0] == "Key" && number >= 1 && number <= sh.keys && number-1 != sh.pagesKey:
							slot = number - 1
						case parts[0] == "Dial" && number >= 1 && number <= sh.dials:
							slot = sh.keys + number - 1
						}
					}
				case "shared":
					shared = !shared
				case "binding":
					next := draft.clone()
					binding := Binding{Control: r.Value, Label: r.Value}
					for _, c := range s.Controls.Snapshot() {
						if c.ID == r.Value {
							binding.Label = c.Label
						}
					}
					if slot < sh.keys {
						if shared {
							next.SharedKeys[slot] = binding
						} else {
							next.Pages[n].Keys[slot] = binding
						}
					} else {
						if shared {
							next.SharedDials[slot-sh.keys] = binding
						} else {
							next.Pages[n].Dials[slot-sh.keys] = binding
						}
					}
					err = next.Validate(s.Controls.Snapshot())
					if err == nil {
						draft = next
						dirty = true
					}
				case "cancel":
					edit(target)
					dirty = false
				case "save":
					err = draft.Validate(s.Controls.Snapshot())
					if err != nil {
						break
					}
					next := settings.clone()
					if serial, ok := strings.CutPrefix(target, serialTarget); ok {
						next.Serials[serial] = draft.clone()
					} else if model := strings.TrimPrefix(target, modelTarget); model == plusXLModel {
						next.Layout = draft.clone()
					} else {
						next.Models[model] = draft.clone()
					}
					err = save(next)
					if err == nil {
						dirty = false
						status = "Saved"
					}
				case "deck":
					if _, ok := settings.Devices[r.Value]; !ok {
						err = fmt.Errorf("unknown deck")
					} else {
						editDeck = r.Value
					}
				case "deck-name":
					if editDeck == "" {
						err = fmt.Errorf("no deck selected")
						break
					}
					next := settings.clone()
					known := next.Devices[editDeck]
					known.Name = strings.TrimSpace(r.Value)
					next.Devices[editDeck] = known
					err = save(next)
				case "deck-own", "deck-forget":
					forget := r.ID == "streamdeck.deck-forget"
					editing := target == serialTarget+editDeck
					_, own := settings.Serials[editDeck]
					model := settings.Devices[editDeck].Model
					switch {
					case editDeck == "":
						err = fmt.Errorf("no deck selected")
					case dirty && editing:
						err = fmt.Errorf("save or cancel before changing this deck's layout")
					default:
						next := settings.clone()
						switch {
						case forget:
							delete(next.Devices, editDeck)
							delete(next.Serials, editDeck)
						case own:
							delete(next.Serials, editDeck)
						default:
							next.Serials[editDeck] = settings.modelLayout(model).clone().forModel(model)
						}
						if err = save(next); err != nil {
							break
						}
						if !forget && !own {
							chosen = true
							edit(serialTarget + editDeck)
						} else if editing {
							edit(modelTarget + model)
						}
					}
				case "paging":
					if r.Value != pagingIndependent && r.Value != pagingMirrored {
						err = fmt.Errorf("paging must be independent or mirrored")
						break
					}
					next := settings.clone()
					next.Paging = r.Value
					err = save(next)
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
