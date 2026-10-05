package streamdeck

import "sound-snoofer/snoofer"

// EditorView is read-only presentation of the plugin-owned draft.
type EditorView struct {
	Keys     []EditorSlot
	Dials    []EditorSlot
	Selected int
	Dirty    bool
	Home     bool
}
type EditorSlot struct {
	Control, Label, Source string
}

func editorView(draft Layout, page string, selected int, dirty bool, controls []snoofer.Control) EditorView {
	view := EditorView{Selected: selected, Dirty: dirty, Home: draft.Home == page}
	base := draft.Pages[draft.index(page)]
	effective := draft.expanded(controls).effective(page)
	slot := func(binding, manual, shared Binding) EditorSlot {
		source := ""
		if shared.Control != "" {
			source = "Shared"
		} else if binding.Control != "" && manual.Control == "" {
			source = "Auto"
		}
		return EditorSlot{Control: binding.Control, Label: binding.Label, Source: source}
	}
	for n, b := range effective.Keys {
		view.Keys = append(view.Keys, slot(b, base.Keys[n], draft.SharedKeys[n]))
	}
	for n, b := range effective.Dials {
		view.Dials = append(view.Dials, slot(b, base.Dials[n], draft.SharedDials[n]))
	}
	return view
}
