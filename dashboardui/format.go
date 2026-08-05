package dashboardui

// encodingBadgeClass returns the CSS badge class for an event encoding.
// JSON is neutral (default), CBOR is a warning (may need decoder), raw is neutral.
func encodingBadgeClass(encoding string) string {
	switch encoding {
	case "json", "":
		return badgeNeutral
	case "cbor":
		return badgeWarn
	case "raw":
		return badgeNeutral
	default:
		return badgeNeutral
	}
}
