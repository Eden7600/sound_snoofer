package vrclipboard

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"sound-snoofer/internal/desktopinput"
	"sound-snoofer/internal/vroverlay"
)

const panelWidth, panelHeight = 1024, 512
const maxEventAge = 250 * time.Millisecond

type padButton struct {
	label  string
	action desktopinput.Action
	bounds image.Rectangle
}

var padButtons = [...]padButton{
	{"Copy", desktopinput.Copy, image.Rect(32, 136, 260, 244)},
	{"Cut", desktopinput.Cut, image.Rect(276, 136, 504, 244)},
	{"Paste", desktopinput.Paste, image.Rect(520, 136, 748, 244)},
	{"Select All", desktopinput.SelectAll, image.Rect(764, 136, 992, 244)},
	{"Tab", desktopinput.Tab, image.Rect(32, 260, 260, 368)},
	{"Backspace", desktopinput.Backspace, image.Rect(276, 260, 504, 368)},
	{"Enter", desktopinput.Enter, image.Rect(520, 260, 748, 368)},
	{"Use Focused\nApp", "", image.Rect(764, 260, 992, 368)},
}

type gesture struct {
	button int
	cursor uint32
	device uint32
	target desktopinput.Target
	at     time.Time
}

type intent struct {
	target desktopinput.Target
	action desktopinput.Action
}

type panel struct {
	target  desktopinput.Target
	active  bool
	since   time.Time
	hover   int
	press   *gesture
	message string
	failed  bool
}

func newPanel() *panel { return &panel{hover: -1} }

func (p *panel) activate(target desktopinput.Target, now time.Time) {
	p.target, p.active, p.since = target, true, now
	p.hover, p.press, p.message, p.failed = -1, nil, "", false
}

func (p *panel) deactivate() {
	p.active, p.press, p.hover = false, nil, -1
}

func buttonAt(x, y float32) int {
	if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) ||
		math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) || x < 0 || y < 0 {
		return -1
	}
	for n, button := range padButtons {
		if x >= float32(button.bounds.Min.X) && x < float32(button.bounds.Max.X) &&
			y >= float32(button.bounds.Min.Y) && y < float32(button.bounds.Max.Y) {
			return n
		}
	}
	return -1
}

func (p *panel) handle(event vroverlay.Event, now time.Time, latest desktopinput.Target) *intent {
	if event.Kind == vroverlay.Hidden || event.Kind == vroverlay.Quit {
		p.deactivate()
		return nil
	}
	if event.Kind == vroverlay.FocusLost {
		p.press, p.hover = nil, -1
		return nil
	}
	if !p.active {
		return nil
	}
	if event.Age < 0 || event.Age > maxEventAge || now.Add(-event.Age).Before(p.since) {
		p.press, p.hover = nil, -1
		return nil
	}
	hit := buttonAt(event.X, event.Y)
	switch event.Kind {
	case vroverlay.Move:
		if p.press != nil && (hit != p.press.button || event.Cursor != p.press.cursor || event.Device != p.press.device) {
			p.press = nil
		}
		p.hover = hit
	case vroverlay.Down:
		if event.Button != 1 {
			return nil
		}
		if p.press != nil {
			p.press = nil
			return nil
		}
		p.hover = hit
		if hit >= 0 && (padButtons[hit].action == "" || p.target.Window != 0) {
			p.press = &gesture{button: hit, cursor: event.Cursor, device: event.Device, target: p.target, at: now.Add(-event.Age)}
		}
	case vroverlay.Up:
		if event.Button != 1 {
			return nil
		}
		pressed := p.press
		p.press, p.hover = nil, hit
		if pressed == nil || hit != pressed.button || event.Cursor != pressed.cursor || event.Device != pressed.device ||
			pressed.target != p.target || now.Add(-event.Age).Before(pressed.at) {
			return nil
		}
		action := padButtons[hit].action
		if action == "" {
			p.target = latest
			p.message, p.failed = "", false
			if latest.Window == 0 {
				p.message, p.failed = "Select a desktop window first", true
			}
			return nil
		}
		return &intent{target: pressed.target, action: action}
	}
	return nil
}

var (
	padBackground = color.RGBA{0x0e, 0x14, 0x1b, 0xff}
	padText       = color.RGBA{0xe3, 0xed, 0xf3, 0xff}
	padMuted      = color.RGBA{0x87, 0x97, 0xa3, 0xff}
	padAccent     = color.RGBA{0x38, 0xc6, 0xe0, 0xff}
	padError      = color.RGBA{0xff, 0x69, 0x78, 0xff}
)

type padRenderer struct{ faces map[int]font.Face }

func newPadRenderer() (*padRenderer, error) {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	r := &padRenderer{faces: map[int]font.Face{}}
	for _, size := range []int{20, 24, 30, 40} {
		face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			r.close()
			return nil, err
		}
		r.faces[size] = face
	}
	return r, nil
}

func (r *padRenderer) close() {
	for _, face := range r.faces {
		_ = face.Close()
	}
}

func fitted(face font.Face, text string, width int) string {
	text = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, text)
	if font.MeasureString(face, text).Ceil() <= width {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if font.MeasureString(face, string(runes)+"…").Ceil() <= width {
			return string(runes) + "…"
		}
	}
	return ""
}

func (r *padRenderer) text(im *image.RGBA, text string, size, x, baseline, width int, ink color.RGBA) {
	face := r.faces[size]
	drawer := font.Drawer{Dst: im, Src: image.NewUniform(ink), Face: face, Dot: fixed.P(x, baseline)}
	drawer.DrawString(fitted(face, text, width))
}

func rounded(im *image.RGBA, bounds image.Rectangle, radius int, ink color.RGBA) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			cx := max(bounds.Min.X+radius, min(x, bounds.Max.X-radius-1))
			cy := max(bounds.Min.Y+radius, min(y, bounds.Max.Y-radius-1))
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= radius*radius {
				im.SetRGBA(x, y, ink)
			}
		}
	}
}

func (r *padRenderer) render(p *panel) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, panelWidth, panelHeight))
	draw.Draw(im, im.Bounds(), image.NewUniform(padBackground), image.Point{}, draw.Src)
	r.text(im, "Clipboard", 40, 32, 58, 960, padText)
	target := p.target.Label
	if p.target.Window == 0 {
		target = "Select a desktop window"
	}
	r.text(im, "Target: "+target, 24, 32, 104, 960, padMuted)
	for n, button := range padButtons {
		ink, fill := padText, color.RGBA{0x20, 0x2c, 0x38, 0xff}
		enabled := button.action == "" || p.target.Window != 0
		if !enabled {
			ink = padMuted
		}
		if n == p.hover && enabled {
			rounded(im, button.bounds, 14, padAccent)
			fill = color.RGBA{0x2b, 0x3c, 0x4b, 0xff}
		}
		if p.press != nil && p.press.button == n {
			fill, ink = padAccent, padBackground
		}
		rounded(im, button.bounds.Inset(2), 12, fill)
		lines := strings.Split(button.label, "\n")
		baseline := button.bounds.Min.Y + 65 - (len(lines)-1)*17
		for _, line := range lines {
			width := font.MeasureString(r.faces[30], line).Ceil()
			r.text(im, line, 30, button.bounds.Min.X+(button.bounds.Dx()-width)/2, baseline, button.bounds.Dx()-20, ink)
			baseline += 34
		}
	}
	message, ink := p.message, padMuted
	if message == "" {
		message = "Select text or a field in your desktop app first"
	}
	if p.failed {
		ink = padError
	}
	r.text(im, message, 24, 32, 422, 960, ink)
	r.text(im, "Shortcuts use the Windows clipboard", 20, 32, 474, 960, padMuted)
	return im
}

func (r *padRenderer) thumbnail() *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 128, 128))
	draw.Draw(im, im.Bounds(), image.NewUniform(padBackground), image.Point{}, draw.Src)
	rounded(im, image.Rect(35, 16, 94, 86), 7, padAccent)
	rounded(im, image.Rect(39, 20, 90, 82), 4, padBackground)
	rounded(im, image.Rect(48, 10, 81, 28), 5, padAccent)
	r.text(im, "Clipboard", 20, 19, 114, 110, padText)
	return im
}
