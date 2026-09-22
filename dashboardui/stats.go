package dashboardui

import (
	"github.com/larsartmann/templ-components/display"
)

// healthKindToTone maps an internal health kind to a StatCard tone.
func healthKindToTone(kind string) display.StatTone {
	switch kind {
	case statusGood:
		return display.StatToneGreen
	case statusWarn:
		return display.StatToneYellow
	case statusBad:
		return display.StatToneRed
	default:
		return display.StatToneBlue
	}
}
