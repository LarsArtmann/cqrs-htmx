package cqrshtmx

import (
	"encoding/json"
	"net/http"
)

// Response builds HTMX-aware HTTP responses with fluent method chaining.
//
// Usage:
//
//	resp := cqrshtmx.NewResponse(w, r)
//	resp.Trigger("userCreated").PushURL("/users").Apply()
//	w.Write(htmlBytes)
type Response struct {
	w http.ResponseWriter
	r *http.Request
}

// NewResponse creates an HTMX-aware response builder.
func NewResponse(w http.ResponseWriter, r *http.Request) *Response {
	return &Response{w: w, r: r}
}

// IsHTMX returns true if the current request is from HTMX.
func (resp *Response) IsHTMX() bool {
	return IsHTMXRequest(resp.r)
}

// PushURL pushes a new URL into the browser history.
func (resp *Response) PushURL(url string) *Response {
	resp.w.Header().Set(headerPushURL, url)
	return resp
}

// ReplaceURL replaces the current URL in the browser address bar.
func (resp *Response) ReplaceURL(url string) *Response {
	resp.w.Header().Set(headerReplaceURL, url)
	return resp
}

// Redirect performs a client-side redirect (HTMX-aware).
// For HTMX requests, uses HX-Redirect; for regular requests, uses HTTP redirect.
func (resp *Response) Redirect(url string) *Response {
	if resp.IsHTMX() {
		resp.w.Header().Set(headerRedirect, url)
		return resp
	}

	http.Redirect(resp.w, resp.r, url, http.StatusSeeOther)
	return resp
}

// Refresh triggers a full page refresh on the client.
func (resp *Response) Refresh() *Response {
	resp.w.Header().Set(headerRefresh, headerTrue)
	return resp
}

// Location performs a client-side redirect without full page reload.
func (resp *Response) Location(url string) *Response {
	resp.w.Header().Set(headerLocation, url)
	return resp
}

// Reswap changes how the response content is swapped.
func (resp *Response) Reswap(strategy SwapStrategy) *Response {
	resp.w.Header().Set(headerReswap, string(strategy))
	return resp
}

// Retarget changes the target element for the response.
func (resp *Response) Retarget(selector string) *Response {
	resp.w.Header().Set(headerRetarget, selector)
	return resp
}

// Reselect changes the selector for the response content.
func (resp *Response) Reselect(selector string) *Response {
	resp.w.Header().Set(headerReselect, selector)
	return resp
}

// Trigger fires a client-side event as soon as the response is received.
// Accepts either a simple event name or a JSON object with details.
func (resp *Response) Trigger(event string) *Response {
	setTriggerHeader(resp.w, headerTrigger, event)
	return resp
}

// TriggerAfterSwap fires a client-side event after the swap step.
func (resp *Response) TriggerAfterSwap(event string) *Response {
	setTriggerHeader(resp.w, headerTriggerAfterSwap, event)
	return resp
}

// TriggerAfterSettle fires a client-side event after the settle step.
func (resp *Response) TriggerAfterSettle(event string) *Response {
	setTriggerHeader(resp.w, headerTriggerAfterSettle, event)
	return resp
}

// TriggerWithDetail fires a client-side event with JSON detail data.
func (resp *Response) TriggerWithDetail(name string, detail any) *Response {
	setTriggerWithDetail(resp.w, headerTrigger, name, detail)
	return resp
}

// NotifySuccess triggers a success notification via HTMX event.
func (resp *Response) NotifySuccess(message string) *Response {
	return resp.triggerNotification("success", message)
}

// NotifyError triggers an error notification via HTMX event.
func (resp *Response) NotifyError(message string) *Response {
	return resp.triggerNotification("error", message)
}

// NotifyWarning triggers a warning notification via HTMX event.
func (resp *Response) NotifyWarning(message string) *Response {
	return resp.triggerNotification("warning", message)
}

// NotifyInfo triggers an info notification via HTMX event.
func (resp *Response) NotifyInfo(message string) *Response {
	return resp.triggerNotification("info", message)
}

func (resp *Response) triggerNotification(level, message string) *Response {
	return resp.TriggerWithDetail(DefaultNotificationEvent, map[string]string{
		"level":   level,
		"message": message,
	})
}

// Apply finalizes the response headers. Call this before writing the body.
func (resp *Response) Apply() {
	if resp.IsHTMX() {
		resp.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
}

func setTriggerHeader(w http.ResponseWriter, header, event string) {
	existing := w.Header().Get(header)
	if existing == "" {
		w.Header().Set(header, event)
		return
	}

	w.Header().Set(header, existing+","+event)
}

func setTriggerWithDetail(w http.ResponseWriter, header, name string, detail any) {
	data := map[string]any{name: detail}

	encoded, err := json.Marshal(data)
	if err != nil {
		w.Header().Set(header, name)
		return
	}

	existing := w.Header().Get(header)
	if existing == "" {
		w.Header().Set(header, string(encoded))
		return
	}

	var existingMap map[string]any
	if json.Unmarshal([]byte(existing), &existingMap) == nil {
		existingMap[name] = detail
		merged, err := json.Marshal(existingMap)
		if err == nil {
			w.Header().Set(header, string(merged))
			return
		}
	}

	w.Header().Set(header, existing+","+name)
}
