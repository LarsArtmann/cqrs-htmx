package dashboardui

import "context"

// encodingBadge returns the library badge HTML for an event encoding.
// JSON is neutral (default), CBOR is a warning (may need decoder), raw is
// neutral.
func encodingBadge(ctx context.Context, encoding string) string {
	return badgeHTML(ctx, encoding, encodingBadgeType(encoding))
}
