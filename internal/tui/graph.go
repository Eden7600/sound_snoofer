package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"sound-snoofer/internal/model"
)

// graphLines describes mixer readback, never desired operations or unsaved edits.
// Arrows describe configured sends, not measured signal or application health.
func (s screen) graphLines(width int) []string {
	width = max(12, width)
	rows := []string{"Observed sends (not signal levels)", ""}
	if !s.state.Connected || s.state.Error != "" {
		return []string{"Routing observation unavailable."}
	}
	physical, err := model.Limits(s.state.Snapshot.Edition)
	if err != nil {
		return []string{"Waiting for a supported mixer."}
	}
	buses := graphBuses(s.state.Snapshot.Edition)
	if !s.state.Live {
		rows = append(rows, "Preview: current mixer, not proposed routes.", "")
	} else if s.state.Plan != nil && s.state.Plan.HasChanges() {
		rows = append(rows, "Changes pending; showing current connections.", "")
	}
	unknown, paths := 0, 0
	elementPath := false
	addSource := func(label string, sends []string) {
		if len(sends) == 0 {
			return
		}
		paths++
		rows = append(rows, graphNode(label, width))
		for index, bus := range sends {
			branch := "  +--> "
			if index == len(sends)-1 {
				branch = "  `--> "
			}
			edge := branch + graphNode(bus, 4)
			destination := s.graphDestination(bus)
			if destination != "" {
				if ansi.StringWidth(edge)+5+ansi.StringWidth(destination)+2 <= width {
					edge += " --> " + graphNode(destination, width-ansi.StringWidth(edge)-5)
				} else {
					rows = append(rows, edge)
					edge = "       `--> " + graphNode(destination, width-12)
				}
			}
			rows = append(rows, edge)
			if bus == "B2" {
				elementPath = true
			}
		}
		rows = append(rows, "")
	}
	for strip := 0; strip < model.StripCount(s.state.Snapshot.Edition); strip++ {
		sends := []string{}
		for _, bus := range buses {
			value, ok := s.state.Snapshot.Numbers[fmt.Sprintf("Strip[%d].%s", strip, bus)]
			if !ok {
				unknown++
				continue
			}
			if value != 0 {
				sends = append(sends, bus)
			}
		}
		addSource(s.graphSource(strip, physical), sends)
		if strip == physical+1 && len(sends) > 0 {
			elementPath = true
		}
	}
	recorder := s.state.Recorder
	if recorder != nil && recorder.Error == "" {
		sends := []string{}
		for _, bus := range buses {
			if value, ok := recorder.Values["Recorder."+bus]; ok && value != 0 {
				sends = append(sends, bus)
			}
		}
		addSource("Tape: "+recorder.State(), sends)
		if mode, ok := recorder.Values["Recorder.mode.recbus"]; ok && mode == 1 {
			for index, bus := range buses {
				if arm, ok := recorder.Values[fmt.Sprintf("Recorder.ArmBus[%d]", index)]; ok && arm != 0 {
					paths++
					rows = append(rows, graphNode(bus, width), "  `--> "+graphNode("Recorder: "+recorder.State(), width-7), "")
				}
			}
		}
	}
	if paths == 0 {
		message := "All observed sends are off."
		if unknown > 0 {
			message = "No enabled sends observed."
		}
		rows = append(rows, message, "")
	}
	if s.state.Snapshot.Edition == 3 && s.state.Intent != nil && elementPath {
		rows = append(rows, "External host path (not verified):", "[B2] ..> ["+clean(s.state.Snapshot.ElementStatus())+"]", "         ..> [AUX return]", "")
	}
	if unknown > 0 {
		rows = append(rows, fmt.Sprintf("%d strip sends unknown.", unknown))
	}
	if recorder == nil || recorder.State() == "Unknown" {
		rows = append(rows, "Recorder observation unavailable.")
	}
	if s.state.Plan != nil && s.state.Plan.Topology != nil {
		topology := s.state.Plan.Topology
		if topology.Voice != nil {
			for _, reason := range []string{topology.Voice.Reason, topology.Voice.ProcessingReason} {
				if reason != "" {
					rows = append(rows, clean(reason))
				}
			}
		}
		for _, reason := range topology.Unresolved {
			rows = append(rows, "! "+clean(reason))
		}
	}
	return rows
}

func graphNode(label string, width int) string {
	return "[" + ansi.Truncate(clean(label), max(1, width-2), "~") + "]"
}

func graphBuses(edition int) []string {
	physical, _ := model.Limits(edition)
	virtual := model.StripCount(edition) - physical
	buses := make([]string, 0, physical+virtual)
	for index := 1; index <= physical; index++ {
		buses = append(buses, fmt.Sprintf("A%d", index))
	}
	for index := 1; index <= virtual; index++ {
		buses = append(buses, fmt.Sprintf("B%d", index))
	}
	return buses
}

func (s screen) graphSource(strip, physical int) string {
	if strip >= physical {
		names := []string{"VAIO / computer", "AUX", "VAIO3"}
		if s.state.Intent != nil && s.state.Snapshot.Edition == 3 {
			names[1] = "AUX / return"
		}
		return names[strip-physical]
	}
	slot := fmt.Sprintf("input:%d", strip+1)
	prefix := fmt.Sprintf("Input %d: ", strip+1)
	name, known := s.state.Snapshot.Assignments[slot]
	if name != "" {
		return prefix + name
	}
	left, leftKnown := s.state.Snapshot.Numbers[fmt.Sprintf("Patch.asio[%d]", strip*2)]
	right, rightKnown := s.state.Snapshot.Numbers[fmt.Sprintf("Patch.asio[%d]", strip*2+1)]
	if leftKnown && rightKnown && (left > 0 || right > 0) {
		return fmt.Sprintf("%sA1 ASIO %g/%g (%s)", prefix, left, right, empty(s.state.Snapshot.Assignments["A1"]))
	}
	if !known {
		return prefix + "device unknown"
	}
	if !leftKnown || !rightKnown {
		return prefix + "no device; ASIO unknown"
	}
	return prefix + "no device"
}

func (s screen) graphDestination(bus string) string {
	if strings.HasPrefix(bus, "A") {
		name, known := s.state.Snapshot.Assignments[bus]
		if !known {
			return "device unknown"
		}
		if name == "" {
			return "no device"
		}
		return name
	}
	if s.state.Snapshot.Edition == 3 && s.state.Intent != nil {
		switch bus {
		case "B1":
			if s.state.Intent.Recording != nil {
				return "Recording mix"
			}
			return "Virtual output 1"
		case "B2":
			return "Element send"
		case "B3":
			return "Discord / app mic"
		}
	}
	return "Virtual output " + strings.TrimPrefix(bus, "B")
}
