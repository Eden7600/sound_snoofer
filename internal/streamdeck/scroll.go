package streamdeck

import (
	"image"
	"image/color"
	"time"
)

// Text that cannot fit scrolls instead of being cut
// (openspec/changes/deck-text-refinements): it holds, moves left until its
// end shows, holds again, and returns to the start.
const (
	scrollInterval = 80 * time.Millisecond // Redraw cadence while anything scrolls.
	scrollHold     = 1200 * time.Millisecond
	scrollSpeed    = 30 // Pixels per second.
)

// scrollOffset is how far a line distance pixels too wide has moved at phase.
func scrollOffset(phase time.Duration, distance int) int {
	if distance <= 0 {
		return 0
	}
	travel := time.Duration(distance) * time.Second / scrollSpeed
	at := phase % (2*scrollHold + travel)
	switch {
	case at < scrollHold:
		return 0
	case at < scrollHold+travel:
		return int((at - scrollHold) * scrollSpeed / time.Second)
	}
	return distance
}

// drawScrolling sets one line of s in a box width pixels wide at key pixel
// (x, y), clipped to the box and offset by the scroll phase.
func (t *typeset) drawScrolling(im *image.RGBA, scale int, kind typeface, px, x, y, width int, s string, c color.RGBA, phase time.Duration) {
	offset := scrollOffset(phase, t.width(kind, px, s)-width)
	clip := image.Rect(x*scale, y*scale, (x+width)*scale, (y+px*3/2)*scale).Intersect(im.Bounds())
	t.draw(im.SubImage(clip).(*image.RGBA), scale, kind, px, x-offset, y, s, c)
}
