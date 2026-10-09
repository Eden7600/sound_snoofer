package streamdeck

import (
	"image"
	"image/color"
	"time"
)

// Text that cannot fit scrolls instead of being cut
// (openspec/changes/deck-text-refinements): a continuous loop of the text, a
// gap, and the text again, moving left without pausing.
const (
	scrollInterval = 80 * time.Millisecond // Redraw cadence while anything scrolls.
	scrollSpeed    = 30                    // Pixels per second.
	scrollGap      = 24                    // Pixels between the end of the text and its next copy.
)

// scrollOffset is how far a loop of period pixels has moved at phase.
func scrollOffset(phase time.Duration, period int) int {
	if period <= 0 {
		return 0
	}
	return int(phase*scrollSpeed/time.Second) % period
}

// drawScrolling sets one line of s in a box width pixels wide at key pixel
// (x, y), clipped to the box, looping continuously by the scroll phase.
func (t *typeset) drawScrolling(im *image.RGBA, scale int, kind typeface, px, x, y, width int, s string, c color.RGBA, phase time.Duration) {
	period := t.width(kind, px, s) + scrollGap
	offset := scrollOffset(phase, period)
	clip := image.Rect(x*scale, y*scale, (x+width)*scale, (y+px*3/2)*scale).Intersect(im.Bounds())
	box := im.SubImage(clip).(*image.RGBA)
	// The copy that is leaving, then the one following it in.
	t.draw(box, scale, kind, px, x-offset, y, s, c)
	t.draw(box, scale, kind, px, x-offset+period, y, s, c)
}
