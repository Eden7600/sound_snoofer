package streamdeck

import (
	"image/color"
	"strings"
)

// UI contract: docs/ui-contract.md. Keep semantic precedence independent of icons.
var (
	neutralColor    = color.RGBA{135, 151, 163, 255}
	activeColor     = color.RGBA{58, 198, 225, 255}
	attentionColor  = color.RGBA{238, 183, 76, 255}
	criticalColor   = color.RGBA{255, 105, 120, 255}
	meterGreenColor = color.RGBA{45, 210, 140, 255}
	meterAmberColor = color.RGBA{245, 190, 60, 255}
	meterRedColor   = color.RGBA{255, 90, 100, 255}
)

// Key surfaces: an off key is keyColor with area-coloured text; an on key is
// filled and inked with inkColor.
var (
	keyColor     = color.RGBA{23, 25, 27, 255}
	inkColor     = color.RGBA{11, 12, 13, 255}
	offValue     = color.RGBA{138, 144, 150, 255}
	lightValue   = color.RGBA{242, 242, 239, 255}
	unknownColor = color.RGBA{94, 101, 108, 255}
)

// Areas name the part of the studio a key belongs to. The colour is
// identity, never state: keyAccent decides state independently of it.
var (
	micArea        = color.RGBA{56, 198, 224, 255}
	playbackArea   = color.RGBA{110, 147, 255, 255}
	recorderArea   = color.RGBA{169, 139, 255, 255}
	soundboardArea = color.RGBA{95, 211, 138, 255}
	mediaArea      = color.RGBA{229, 124, 240, 255}
	lightsArea     = color.RGBA{239, 231, 211, 255}
	otherArea      = color.RGBA{197, 203, 208, 255}
)

// keyArea is the area colour for an icon identifier.
func keyArea(icon string) color.RGBA {
	icon = strings.TrimSuffix(strings.TrimSuffix(icon, "-muted"), "-off")
	switch {
	case icon == "speaker-mute", icon == "vr-playback", icon == "monitor", icon == "discord-deafen":
		return playbackArea
	case strings.HasPrefix(icon, "record-"), strings.HasPrefix(icon, "tape-"):
		return recorderArea
	case strings.HasPrefix(icon, "soundboard-"):
		return soundboardArea
	case strings.HasPrefix(icon, "media-"), icon == "app-audio":
		return mediaArea
	case strings.HasPrefix(icon, "hue-"), strings.HasPrefix(icon, "huesync-"):
		return lightsArea
	}
	switch icon {
	case "mic-mute", "vr-mic", "mic-stack", "mode-direct", "mode-element", "tap-pre", "tap-post", "echo",
		"discord-connect", "discord-video", "screen-share", "call-leave":
		return micArea
	}
	return otherArea
}

func keyAccent(value, icon string, on, fallback bool) color.RGBA {
	switch value {
	case "UNAVAILABLE", "UNAVAIL", "UNKNOWN", "N/A":
		return neutralColor
	case "ERROR", "FAILED":
		return criticalColor
	case "PENDING", "WAIT", "NO OUTPUT":
		return attentionColor
	}
	if fallback {
		return attentionColor
	}
	if strings.HasSuffix(icon, "-muted") || value == "MUTED" || value == "RECORDING" || value == "REC" {
		return criticalColor
	}
	if on || value == "ON" || value == "LIVE" || value == "PLAYING" || value == "ACTIVE" || value == "HERE" || value == "IN USE" || value == "AUDIBLE" || value == "DIRECT" || value == "ELEMENT" || (icon == "monitor" && (value == "PRE" || value == "POST")) {
		return activeColor
	}
	return neutralColor
}
