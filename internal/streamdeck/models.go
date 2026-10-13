package streamdeck

import (
	"slices"
	"strconv"
	"strings"
)

// turn is how an image is turned before it is sent to a deck.
type turn int

const (
	turnCounterclockwise turn = iota // 90° counterclockwise: the + XL, and the zero value.
	turnHalf                         // 180°.
	upright
)

// display is the strip or screen a model has besides its keys, and how it
// is written.
type display int

const (
	noDisplay   display = iota
	plusXLStrip         // One panel per dial, turned counterclockwise, report 0x0B.
	plusStrip           // One panel per dial, upright, report 0x0C with a region header.
	neoScreen           // The info screen, turned 180°, report 0x0B.
)

// Model is the hardware of a supported Stream Deck. Layouts are persisted
// by ID; Name is display text only.
type Model struct {
	ID, Name      string
	Products      []uint16 // USB product IDs (vendor 0FD9).
	Columns, Rows int
	KeySize       int // Native key image pixels, square.
	Dials         int // Physical dials; the last one pages.
	TouchKeys     int // Neo touch buttons, reported after the keys.
	keyTurn       turn
	display       display
}

// Keys is the number of display keys.
func (m Model) Keys() int { return m.Columns * m.Rows }

// Neo info screen size.
const (
	screenWidth  = 248
	screenHeight = 58
)

// models are the supported JPEG-protocol decks
// (openspec/changes/multi-model-stream-deck). Original v2 shares the MK.2's
// geometry and protocol, so it shares its layout.
var models = []Model{
	{ID: "plus-xl", Name: "Stream Deck + XL", Products: []uint16{0x00c6}, Columns: 9, Rows: 4, KeySize: 112, Dials: 6, keyTurn: turnCounterclockwise, display: plusXLStrip},
	{ID: "plus", Name: "Stream Deck +", Products: []uint16{0x0084}, Columns: 4, Rows: 2, KeySize: 120, Dials: 4, keyTurn: upright, display: plusStrip},
	{ID: "xl", Name: "Stream Deck XL", Products: []uint16{0x006c, 0x008f}, Columns: 8, Rows: 4, KeySize: 96, keyTurn: turnHalf},
	{ID: "mk2", Name: "Stream Deck MK.2", Products: []uint16{0x006d, 0x0080, 0x00a5}, Columns: 5, Rows: 3, KeySize: 72, keyTurn: turnHalf},
	{ID: "neo", Name: "Stream Deck Neo", Products: []uint16{0x009a}, Columns: 4, Rows: 2, KeySize: 96, TouchKeys: 2, keyTurn: turnHalf, display: neoScreen},
}

// plusXL is the model every layout predating other models belongs to.
var plusXL = models[0]

// Models lists the supported models.
func Models() []Model {
	out := slices.Clone(models)
	for n := range out {
		out[n].Products = slices.Clone(out[n].Products)
	}
	return out
}

// ModelByID finds a supported model by its persisted ID.
func ModelByID(id string) (Model, bool) {
	for _, m := range models {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}

// modelForProduct finds the model with a USB product ID.
func modelForProduct(product uint16) (Model, bool) {
	for _, m := range models {
		if slices.Contains(m.Products, product) {
			return m, true
		}
	}
	return Model{}, false
}

// productForPath reads the product ID of an Elgato HID interface path,
// which names its IDs as "vid_0fd9&pid_xxxx".
func productForPath(path string) (uint16, bool) {
	lower := strings.ToLower(path)
	_, rest, ok := strings.Cut(lower, "vid_0fd9&pid_")
	if !ok || len(rest) < 4 {
		return 0, false
	}
	product, err := strconv.ParseUint(rest[:4], 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(product), true
}

// modelForPath finds the model of an Elgato HID interface path.
func modelForPath(path string) (Model, bool) {
	product, ok := productForPath(path)
	if !ok {
		return Model{}, false
	}
	return modelForProduct(product)
}
