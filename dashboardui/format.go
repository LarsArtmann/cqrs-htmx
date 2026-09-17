package dashboardui

// encodingBadge returns the library badge HTML for an event encoding.
// JSON is neutral (default), CBOR is a warning (may need decoder), raw is
// neutral.
func encodingBadge(encoding string) string {
	return badgeHTML(encoding, encodingBadgeType(encoding))
}
