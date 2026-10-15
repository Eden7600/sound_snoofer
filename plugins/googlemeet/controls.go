package googlemeet

import (
	"context"
	"fmt"
	"time"

	"sound-snoofer/snoofer"
)

func reasonText(reason string) string {
	switch reason {
	case "not-in-call":
		return "Not in a call"
	case "reload-required":
		return "Reload this tab"
	case "unavailable":
		return "Unavailable"
	default:
		return "Unknown state"
	}
}

func (w *worker) callReason(now time.Time) string {
	if len(w.choices) > 0 {
		return "Select a call"
	}
	if len(w.browsers) == 0 {
		return "Disconnected"
	}
	supported := false
	for _, b := range w.browsers {
		supported = supported || b.Supported
	}
	if !supported {
		return "Update companion"
	}
	for _, b := range w.browsers {
		if b.Supported && b.Calls == nil {
			return "Unknown call state"
		}
	}
	for _, c := range w.calls {
		if now.Sub(c.sampled) >= freshness {
			return "Call unavailable"
		}
		if c.state.State == "unknown" {
			return "Unknown call state"
		}
	}
	return "Not in a call"
}

func (w *worker) actionControl(control, id, label, icon string, now time.Time) (snoofer.Control, command) {
	current, selected := w.current(w.selected, now)
	state := controlState(current.state, control)
	key := actionKey{w.selected, control}
	c := command{target: w.selected, selection: w.selection, control: control, revision: state.Revision,
		known: state.Value != nil, available: selected && state.Available}
	value, status := "N/A", ""
	if selected && state.Value != nil {
		value, c.want = snoofer.OnOff(*state.Value), !*state.Value
	}
	switch {
	case !selected:
		status = w.callReason(now)
	case !state.Available || state.Value == nil:
		status = reasonText(state.Reason)
	default:
		status = w.notes[key]
		for _, p := range w.pending {
			if (actionKey{p.command.target, p.command.control}) == key {
				status = "Pending"
				break
			}
		}
	}
	if !w.services.Live {
		status = "Preview"
	}
	if control == "mic" && value == "On" {
		icon = "mic-mute-muted"
	}
	if control == "camera" && value == "Off" {
		icon = "discord-video-off"
	}
	identity := controlContext{w.selected, w.selection, state.Revision}
	epoch := w.epochs[id]
	if epoch.epoch == 0 || epoch.context != identity {
		w.nextEpoch++
		epoch = controlEpoch{identity, w.nextEpoch}
		w.epochs[id] = epoch
	}
	return snoofer.Control{ID: id, Label: "Meet " + label, ShortLabel: label, Group: "Google Meet", Kind: "toggle", Icon: icon,
		Value: value, Status: status, Epoch: epoch.epoch, Operations: []string{"press"},
		Available: w.services.Live && c.available && c.known}, c
}

func (w *worker) report(now time.Time) snoofer.Control {
	bridge := w.connection.Status()
	state, value := snoofer.ConnectionDisconnected, "Disconnected"
	supported, outdated := 0, 0
	for _, b := range w.browsers {
		if b.Supported {
			supported++
		} else {
			outdated++
		}
	}
	switch {
	case bridge.Error != "":
		state, value = snoofer.ConnectionError, "Connection error"
		w.link.Fail(bridge.Error, now)
	case outdated > 0:
		state, value = snoofer.ConnectionAttention, "Update companion"
	case supported > 0:
		state, value = snoofer.ConnectionConnected, "Connected"
	}
	w.link.Observe(state, now)
	details := []snoofer.ConnectionDetail{{Label: "Browsers", Value: fmt.Sprint(len(w.browsers))}, {Label: "Joined calls", Value: fmt.Sprint(len(w.choices))}}
	if bridge.Refused != "" {
		details = append(details, snoofer.ConnectionDetail{Label: "Last refusal", Value: bridge.Refused})
	}
	return snoofer.Control{ID: "googlemeet.app-client", Label: "Google Meet", Group: "Google Meet", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: w.link.Report(fmt.Sprintf("127.0.0.1:%d/nowplaying", bridge.Port), details...)}
}

func (w *worker) publish(ctx context.Context, now time.Time) {
	w.reconcile(now)
	mute, muteCommand := w.actionControl("mic", "googlemeet.mute", "Mute", "mic-mute", now)
	camera, cameraCommand := w.actionControl("camera", "googlemeet.camera", "Camera", "discord-video", now)
	options, labels, targets := []string{}, map[string]string{}, map[string]target{}
	callValue, callStatus := w.callReason(now), ""
	if w.selected == (target{}) {
		options = append(options, "")
		labels[""] = "Select a call"
		if len(w.choices) > 0 {
			callValue = "Select a call"
		}
	} else {
		callValue, callStatus = w.selected.label(), ""
	}
	for _, choice := range w.choices {
		value := choice.value()
		options = append(options, value)
		labels[value], targets[value] = choice.label(), choice
	}
	controls := []snoofer.Control{mute, camera,
		{ID: "googlemeet.call", Label: "Meet Call", ShortLabel: "Call", Group: "Google Meet", Kind: "status", Icon: "call", Value: callValue, Status: callStatus, Available: true},
		{ID: "googlemeet.target", Label: "Meet Call", ShortLabel: "Call", Group: "Google Meet", Kind: "selection", Icon: "call", Value: w.selected.value(),
			Options: options, OptionLabels: labels, Operations: []string{"set"}, Epoch: w.catalogue, SurfaceOnly: true, Available: len(w.choices) > 0},
		w.report(now)}
	selection, catalogue := w.selection, w.catalogue
	commands := map[string]command{"googlemeet.mute": muteCommand, "googlemeet.camera": cameraCommand}
	lifetime := ctx
	_ = w.services.Controls.Publish("googlemeet", controls, func(ctx context.Context, r snoofer.Request) error {
		if err := lifetime.Err(); err != nil {
			return err
		}
		c := commands[r.ID]
		if r.ID == "googlemeet.target" {
			c = command{target: targets[r.Value], selection: selection, catalogue: catalogue}
		}
		c.request, c.accepted, c.complete = r, time.Now(), r.ClaimCompletion(lifetime)
		select {
		case w.requests <- c:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("Meet queue full")
		}
	})
}
