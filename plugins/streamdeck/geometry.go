package streamdeck

import device "sound-snoofer/internal/streamdeck"

// plusXLModel is the model of every layout that predates other models.
const plusXLModel = "plus-xl"

// shape is a layout's geometry: its key grid, user dials and the reserved
// Pages key of decks without dials.
type shape struct {
	columns, keys, dials int
	pagesKey             int // Reserved key index, or -1.
}

// shapeOf is a model's layout geometry; the empty model is the + XL.
func shapeOf(model string) shape {
	m, ok := device.ModelByID(model)
	if !ok {
		m, _ = device.ModelByID(plusXLModel)
	}
	s := shape{columns: m.Columns, keys: m.Keys(), dials: max(0, m.Dials-1), pagesKey: -1}
	if m.Dials == 0 && m.TouchKeys == 0 {
		s.pagesKey = m.Keys() - 1
	}
	return s
}

// usable reports whether key n can hold a binding or a region's control.
func (s shape) usable(n int) bool { return n >= 0 && n < s.keys && n != s.pagesKey }

// modelKey is how a layout records its model: the + XL as empty.
func modelKey(model string) string {
	if model == plusXLModel {
		return ""
	}
	return model
}

// forModel returns the layout as one for model.
func (l Layout) forModel(model string) Layout {
	l.model = modelKey(model)
	return l
}

// shape is the layout's geometry.
func (l Layout) shape() shape { return shapeOf(l.model) }
