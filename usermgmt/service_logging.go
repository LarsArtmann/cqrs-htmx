package usermgmt

import "log/slog"

// logAuthEvent emits a structured auth-related log line with consistent key
// names ("event", "user_id", plus caller-supplied attributes). Shared by the
// logAuth methods on *Service and *OAuth2Service to keep log shape identical
// across all auth flows.
func logAuthEvent(logger *slog.Logger, event string, userID UserID, attrs ...any) {
	args := make([]any, 0, 4+len(attrs))
	args = append(args, "event", event, "user_id", userID)
	args = append(args, attrs...)
	logger.Info("usermgmt: "+event, args...)
}
