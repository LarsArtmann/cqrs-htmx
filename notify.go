package cqrshtmx

// DefaultNotificationEvent is the default HTMX trigger event name for client-side notifications.
var DefaultNotificationEvent = "showMessage"

// NotifySuccess triggers an HTMX notification with success level.
func NotifySuccess(message string) HandlerOption {
	return notifyOption("success", message)
}

// NotifyError triggers an HTMX notification with error level.
func NotifyError(message string) HandlerOption {
	return notifyOption("error", message)
}

// NotifyWarning triggers an HTMX notification with warning level.
func NotifyWarning(message string) HandlerOption {
	return notifyOption("warning", message)
}

// NotifyInfo triggers an HTMX notification with info level.
func NotifyInfo(message string) HandlerOption {
	return notifyOption("info", message)
}

func notifyOption(level, message string) HandlerOption {
	return TriggerWithDetail(DefaultNotificationEvent, map[string]string{
		"level":   level,
		"message": message,
	})
}
