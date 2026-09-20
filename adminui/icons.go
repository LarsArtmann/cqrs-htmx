package adminui

import (
	"github.com/larsartmann/templ-components/icons"
)

// Adminui icon key. These map adminui's string-based icon keys to the
// typed icons.Name constants from templ-components, so typos fall back to
// the Question icon instead of breaking the UI.
const (
	iconDashboard = string(icons.Chart)
	iconUsers     = string(icons.Users)
	iconTenants   = string(icons.BuildingOffice2)
	iconMembers   = string(icons.UserPlus)
	iconAudit     = string(icons.Clock)
)
