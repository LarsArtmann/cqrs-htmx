package cqrshtmx

// DefaultNotificationEvent is the default HTMX trigger event name for client-side notifications.
var DefaultNotificationEvent = "showMessage"

// NotifySuccess triggers an HTMX notification with success level.
func NotifySuccess(message string) HandlerOption {
	return TriggerWithDetail(DefaultNotificationEvent, map[string]string{
		"level":   "success",
		"message": message,
	})
}

// NotifyError triggers an HTMX notification with error level.
func NotifyError(message string) HandlerOption {
	return TriggerWithDetail(DefaultNotificationEvent, map[string]string{
		"level":   "error",
		"message": message,
	})
}

// NotifyWarning triggers an HTMX notification with warning level.
func NotifyWarning(message string) HandlerOption {
	return TriggerWithDetail(DefaultNotificationEvent, map[string]string{
		"level":   "warning",
		"message": message,
	})
}

// NotifyInfo triggers an HTMX notification with info level.
func NotifyInfo(message string) HandlerOption {
	return TriggerWithDetail(DefaultNotificationEvent, map[string]string{
		"level":   "info",
		"message": message,
	})
}
