package streamdeck

import (
	"image/color"
	"strings"
)

// UI contract: docs/ui-contract.md. Keep semantic precedence independent of icons.
var (
	backgroundColor = color.RGBA{14, 20, 27, 255}
	textColor       = color.RGBA{227, 237, 243, 255}
	neutralColor    = color.RGBA{135, 151, 163, 255}
	activeColor     = color.RGBA{58, 198, 225, 255}
	attentionColor  = color.RGBA{238, 183, 76, 255}
	criticalColor   = color.RGBA{255, 105, 120, 255}
	meterQuietColor = color.RGBA{24, 38, 42, 255}
	meterGreenColor = color.RGBA{45, 210, 140, 255}
	meterAmberColor = color.RGBA{245, 190, 60, 255}
	meterRedColor   = color.RGBA{255, 90, 100, 255}
)

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
