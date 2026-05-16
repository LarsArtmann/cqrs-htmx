package cqrshtmx

const defaultNotificationEvent = "showMessage"

// NotificationLevel represents the severity level of a notification.
type NotificationLevel string

// Notification level constants for HTMX client-side notifications.
const (
	LevelSuccess NotificationLevel = "success"
	LevelError   NotificationLevel = "error"
	LevelWarning NotificationLevel = "warning"
	LevelInfo    NotificationLevel = "info"
)

// NotifySuccess triggers an HTMX notification with success level.
func NotifySuccess(message string) HandlerOption {
	return notifyOption(defaultNotificationEvent, LevelSuccess, message)
}

// NotifyError triggers an HTMX notification with error level.
func NotifyError(message string) HandlerOption {
	return notifyOption(defaultNotificationEvent, LevelError, message)
}

// NotifyWarning triggers an HTMX notification with warning level.
func NotifyWarning(message string) HandlerOption {
	return notifyOption(defaultNotificationEvent, LevelWarning, message)
}

// NotifyInfo triggers an HTMX notification with info level.
func NotifyInfo(message string) HandlerOption {
	return notifyOption(defaultNotificationEvent, LevelInfo, message)
}

// NotifyWithEvent returns notification builder using a custom HTMX event name.
// This allows different handlers to trigger notifications on different client-side events.
//
// Usage:
//
//	app.Command("CreateUser",
//	    cqrshtmx.NotifyWithEvent("showToast").Success("User created"),
//	)
func NotifyWithEvent(event string) NotifyEventBuilder {
	return NotifyEventBuilder{event: event}
}

// NotifyEventBuilder builds notification HandlerOptions with a custom HTMX event name.
type NotifyEventBuilder struct {
	event string
}

// Success triggers a success notification with the custom event name.
func (b NotifyEventBuilder) Success(message string) HandlerOption {
	return notifyOption(b.event, LevelSuccess, message)
}

// Error triggers an error notification with the custom event name.
func (b NotifyEventBuilder) Error(message string) HandlerOption {
	return notifyOption(b.event, LevelError, message)
}

// Warning triggers a warning notification with the custom event name.
func (b NotifyEventBuilder) Warning(message string) HandlerOption {
	return notifyOption(b.event, LevelWarning, message)
}

// Info triggers an info notification with the custom event name.
func (b NotifyEventBuilder) Info(message string) HandlerOption {
	return notifyOption(b.event, LevelInfo, message)
}

func notifyOption(event string, level NotificationLevel, message string) HandlerOption {
	return TriggerWithDetail(event, map[string]string{
		"level":   string(level),
		"message": message,
	})
}
