package audio

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
)

// reporter owns connection-report timing for the audio integrations. Only the
// publisher goroutine uses it, so it needs no locking.
type reporter struct {
	voicemeeter, callback, recorder, asio, element, defaults snoofer.ConnectionTracker
	connected                                                bool
	buffers                                                  uint32
	processor                                                string // Processor executable; empty means the default.
}

type details []snoofer.ConnectionDetail

func (d *details) add(label, value string) {
	if value != "" {
		*d = append(*d, snoofer.ConnectionDetail{Label: label, Value: value})
	}
}

func report(id, label, value string, conn *snoofer.Connection) snoofer.Control {
	return snoofer.Control{ID: "audio.app-" + id, Label: label, Group: "Audio", Kind: "connection", Value: value,
		SurfaceOnly: true, Available: true, Connection: conn}
}

func editionName(edition int) string {
	switch edition {
	case 1:
		return "Voicemeeter"
	case 2:
		return "Banana"
	case 3:
		return "Potato"
	}
	return ""
}

func (r *reporter) reports(s control.State, now time.Time) []snoofer.Control {
	return []snoofer.Control{r.voicemeeterReport(s, now), r.callbackReport(s, now), r.recorderReport(s, now),
		r.asioReport(s, now), r.elementReport(s, now), r.defaultsReport(s, now)}
}

func (r *reporter) voicemeeterReport(s control.State, now time.Time) snoofer.Control {
	t := &r.voicemeeter
	readError := ""
	if !s.Connected && s.Recorder != nil {
		readError = s.Recorder.Error // The worker stores the native read error here when disconnected.
	}
	if r.connected && !s.Connected {
		t.Fail(firstNonEmpty(readError, s.Error, "Voicemeeter disconnected"), now)
	} else if !s.Connected && s.Error != "" {
		t.Fail(s.Error, now)
	}
	r.connected = s.Connected
	state, value := snoofer.ConnectionDisconnected, "Disconnected"
	switch {
	case s.Connected && s.Stalled:
		// The API answers but the engine is not processing audio.
		state, value = snoofer.ConnectionAttention, "Engine stalled"
		t.Fail(control.StallMessage, now)
	case s.Connected:
		state, value = snoofer.ConnectionConnected, firstNonEmpty(editionName(s.Snapshot.Edition), "Connected")
	}
	t.Observe(state, now)
	if !s.ObservedAt.IsZero() {
		t.Activity(s.ObservedAt)
	}
	var d details
	d.add("Edition", editionName(s.Snapshot.Edition))
	d.add("Version", s.Remote.Version)
	switch s.Remote.Login {
	case 0:
		if s.Remote.DLLPath != "" {
			d.add("Login", "Voicemeeter running")
		}
	case 1:
		d.add("Login", "Logged in; Voicemeeter was not running")
	}
	d.add("Interval", "1s")
	d.add("Health", s.Health)
	d.add("Recovery", s.RecoveryOutcome)
	mode := "Preview"
	if s.Live {
		mode = "Live"
	}
	d.add("Mode", mode)
	d.add("Required", "Yes")
	return report("voicemeeter", "Voicemeeter", value, t.Report(firstNonEmpty(s.Remote.DLLPath, "VoicemeeterRemote64.dll"), d...))
}

func (r *reporter) callbackReport(s control.State, now time.Time) snoofer.Control {
	t := &r.callback
	cb := s.Snapshot.Callback
	state, value := snoofer.ConnectionOff, "Off"
	var d details
	switch {
	case !s.Connected:
		state, value = snoofer.ConnectionUnknown, "Unknown"
	case cb == nil:
	case cb.Error != "":
		state, value = snoofer.ConnectionError, "Error"
		t.Fail(cb.Error, now)
	case s.RecoveryPending:
		state, value = snoofer.ConnectionAttention, "Recovering"
	case s.Stalled:
		state, value = snoofer.ConnectionAttention, "Stalled"
		t.Fail(control.StallMessage, now)
	case cb.Active:
		state, value = snoofer.ConnectionConnected, "Active"
	default:
		state, value = snoofer.ConnectionConnecting, "Starting"
	}
	if cb != nil {
		if cb.Buffers != r.buffers {
			r.buffers = cb.Buffers
			t.Activity(now)
		}
		d.add("Buffers", fmt.Sprint(cb.Buffers))
		d.add("Synced", fmt.Sprint(cb.Synced))
		d.add("Starting", fmt.Sprint(cb.Starting))
		d.add("Ending", fmt.Sprint(cb.Ending))
		d.add("Changes", fmt.Sprint(cb.Changes))
		d.add("Interval", "1s")
	}
	d.add("Recovery", s.RecoveryOutcome)
	t.Observe(state, now)
	return report("callback", "Audio callback monitor", value, t.Report("snoofer-audio-monitor.dll", d...))
}

func (r *reporter) recorderReport(s control.State, now time.Time) snoofer.Control {
	t := &r.recorder
	state, value := snoofer.ConnectionOff, "Off"
	var d details
	switch {
	case s.Intent == nil || s.Intent.Recording == nil:
	case s.Recorder == nil:
		state, value = snoofer.ConnectionUnknown, "Unknown"
	case s.Recorder.Error != "":
		state, value = snoofer.ConnectionError, "Error"
		t.Fail(s.Recorder.Error, now)
	default:
		state, value = snoofer.ConnectionConnected, s.Recorder.State()
		if !s.ObservedAt.IsZero() {
			t.Activity(s.ObservedAt)
		}
		d.add("Transport", s.Recorder.State())
		d.add("Conflict", s.Recorder.Conflict())
	}
	t.Observe(state, now)
	return report("recorder", "Voicemeeter recorder", value, t.Report("Voicemeeter Recorder", d...))
}

func (r *reporter) asioReport(s control.State, now time.Time) snoofer.Control {
	t := &r.asio
	state, value, endpoint := snoofer.ConnectionUnknown, "Unknown", "Voicemeeter A1 (ASIO)"
	var d details
	if s.Plan != nil && s.Plan.Topology != nil {
		top := s.Plan.Topology
		switch {
		case top.ASIOActive:
			state, value, endpoint = snoofer.ConnectionConnected, "Active", top.ASIOName
			if !s.ObservedAt.IsZero() {
				t.Activity(s.ObservedAt)
			}
		default:
			state, value = snoofer.ConnectionDisconnected, "Not present"
		}
		d.add("Unavailable", strings.Join(top.ASIOUnavailable, ", "))
	}
	if !s.Connected {
		state, value = snoofer.ConnectionUnknown, "Unknown"
	}
	if rate, ok := s.Snapshot.Numbers["Bus[0].device.sr"]; ok && rate > 0 {
		d.add("Sample rate", fmt.Sprintf("%.0f Hz", rate))
	}
	d.add("A1 device", s.Snapshot.Assignments["A1"])
	t.Observe(state, now)
	return report("asio", "ASIO interface", value, t.Report(endpoint, d...))
}

func (r *reporter) elementReport(s control.State, now time.Time) snoofer.Control {
	t := &r.element
	e := s.Snapshot.Element
	state, value := snoofer.ConnectionUnknown, "Unknown"
	switch {
	case e == nil || !s.Connected:
	case e.Error != "":
		state, value = snoofer.ConnectionError, "Error"
		t.Fail(e.Error, now)
	case !e.Known:
	case e.Running:
		state, value = snoofer.ConnectionConnected, "Running"
		if !s.ObservedAt.IsZero() {
			t.Activity(s.ObservedAt)
		}
	default:
		state, value = snoofer.ConnectionDisconnected, "Not running"
	}
	var d details
	if s.Plan != nil && s.Plan.Topology != nil && s.Plan.Topology.Voice != nil {
		d.add("Effective mode", s.Plan.Topology.Voice.EffectiveMode)
		d.add("Processing", s.Plan.Topology.Voice.ProcessingReason)
	}
	t.Observe(state, now)
	return report("element", "Element", value, t.Report(cmp.Or(r.processor, config.DefaultProcessorProcess), d...))
}

func (r *reporter) defaultsReport(s control.State, now time.Time) snoofer.Control {
	t := &r.defaults
	result := s.DefaultsDetail
	state, value := snoofer.ConnectionUnknown, "Unknown"
	switch s.DefaultKind {
	case windowsaudio.Disabled:
		state, value = snoofer.ConnectionOff, "Off"
	case windowsaudio.Preview:
		state, value = snoofer.ConnectionOff, "Preview"
	case windowsaudio.Verified:
		state, value = snoofer.ConnectionConnected, "Verified"
	case windowsaudio.Attention:
		state, value = snoofer.ConnectionAttention, "Attention"
		t.Fail(result.Status, now)
	}
	if !result.LastCorrection.IsZero() {
		t.Activity(result.LastCorrection)
	}
	var d details
	d.add("Status", result.Status)
	d.add("Playback target", result.Playback)
	d.add("Capture target", result.Capture)
	if result.Suspended {
		d.add("Suspended", "Yes")
	}
	t.Observe(state, now)
	return report("windows-defaults", "Windows default devices", value, t.Report("Windows Core Audio (MMDevice, IPolicyConfig)", d...))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
