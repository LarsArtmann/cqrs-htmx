package cqrshtmx

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
)

// Content type constants for consistent HTTP response headers.
const (
	ContentTypePlain = "text/plain; charset=utf-8"
	ContentTypeHTML  = "text/html; charset=utf-8"
	ContentTypeJSON  = "application/json; charset=utf-8"
)

// Response builds HTMX-aware HTTP responses with fluent method chaining.
//
// Usage:
//
//	resp := cqrshtmx.NewResponse(w, r)
//	resp.Trigger("userCreated").PushURL("/users").Apply()
//	w.Write(htmlBytes)
type Response struct {
	w           http.ResponseWriter
	r           *http.Request
	redirectURL string
	statusCode  int
}

// NewResponse creates an HTMX-aware response builder.
func NewResponse(w http.ResponseWriter, r *http.Request) *Response {
	return &Response{w: w, r: r, redirectURL: "", statusCode: 0}
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
// For HTMX requests, sets HX-Redirect header.
// For regular requests, defers the HTTP redirect until Apply() is called,
// allowing chaining with other response methods.
func (resp *Response) Redirect(url string) *Response {
	if resp.IsHTMX() {
		sanitized, safe := sanitizeRedirectURL(url)
		if safe {
			resp.w.Header().Set(headerRedirect, sanitized)
		}
		return resp
	}

	resp.redirectURL = url
	return resp
}

// Refresh triggers a full page refresh on the client.
func (resp *Response) Refresh() *Response {
	resp.w.Header().Set(headerRefresh, HeaderTrue)
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
//
// Calling both Trigger and TriggerWithDetail on the same header
// is unsupported — TriggerWithDetail uses JSON serialization which
// overwrites the simple event name set by Trigger. Use one or the other.
func (resp *Response) TriggerWithDetail(name string, detail any) *Response {
	setTriggerWithDetail(resp.w, headerTrigger, name, detail)
	return resp
}

// NotifySuccess triggers a success notification via HTMX event.
func (resp *Response) NotifySuccess(message string) *Response {
	return resp.triggerNotification(LevelSuccess, message)
}

// NotifyError triggers an error notification via HTMX event.
func (resp *Response) NotifyError(message string) *Response {
	return resp.triggerNotification(LevelError, message)
}

// NotifyWarning triggers a warning notification via HTMX event.
func (resp *Response) NotifyWarning(message string) *Response {
	return resp.triggerNotification(LevelWarning, message)
}

// NotifyInfo triggers an info notification via HTMX event.
func (resp *Response) NotifyInfo(message string) *Response {
	return resp.triggerNotification(LevelInfo, message)
}

func (resp *Response) triggerNotification(level NotificationLevel, message string) *Response {
	return resp.TriggerWithDetail(defaultNotificationEvent, notificationDetail(level, message))
}

// CSRFToken sets the X-CSRF-Token response header so HTMX clients can read
// the token and include it in subsequent requests via hx-headers.
//
// Usage in handlers:
//
//	token := cqrshtmx.CSRFTokenFromContext(r.Context())
//	resp := cqrshtmx.NewResponse(w, r)
//	resp.CSRFToken(token).Apply()
func (resp *Response) CSRFToken(token string) *Response {
	resp.w.Header().Set(defaultCSRFHeaderName, token)
	return resp
}

// Status sets the HTTP status code. The code is deferred to Apply() so
// subsequent header-setting methods (like Redirect, PushURL) still work.
func (resp *Response) Status(code int) *Response {
	resp.statusCode = code
	return resp
}

// Header sets a custom response header.
func (resp *Response) Header(key, value string) *Response {
	resp.w.Header().Set(key, value)
	return resp
}

// ContentType sets the Content-Type header.
func (resp *Response) ContentType(ct string) *Response {
	resp.w.Header().Set("Content-Type", ct)
	return resp
}

// Body writes the given bytes as the response body.
// Calls Apply first if not already applied.
func (resp *Response) Body(data []byte) *Response {
	resp.Apply()
	_, _ = resp.w.Write(data)
	return resp
}

// WriteString writes the given string as the response body.
// Calls Apply first if not already applied.
// Uses io.StringWriter when available to avoid []byte(string) allocation.
func (resp *Response) WriteString(s string) *Response {
	resp.Apply()
	if sw, ok := resp.w.(io.StringWriter); ok {
		_, _ = sw.WriteString(s)
	} else {
		_, _ = resp.w.Write([]byte(s))
	}
	return resp
}

// JSON encodes v as JSON, sets Content-Type, and writes it as the response body.
// Calls Apply first if not already applied. Writes HTTP 500 on marshal failure.
func (resp *Response) JSON(v any) *Response {
	resp.ContentType(ContentTypeJSON)
	resp.Apply()
	encoded, err := json.Marshal(v)
	if err != nil {
		http.Error(resp.w, "json marshal error", http.StatusInternalServerError)
		return resp
	}
	_, _ = resp.w.Write(encoded)
	return resp
}

// Apply finalizes the response. For non-HTMX redirects, writes the redirect
// response immediately. For HTMX requests, sets Content-Type.
// Returns true if the response was written (redirect), false if the caller
// must still write the body.
func (resp *Response) Apply() bool {
	if resp.redirectURL != "" {
		redirectURL, safe := sanitizeRedirectURL(resp.redirectURL)
		if safe {
			//nolint:gosec // G710: sanitizeRedirectURL validates URL is safe (relative path only)
			http.Redirect(resp.w, resp.r, redirectURL, http.StatusSeeOther)
			return true
		}
		// Unsafe redirect - fall through to render content instead
		resp.w.WriteHeader(http.StatusBadRequest)
		return true
	}

	if resp.IsHTMX() {
		resp.w.Header().Set("Content-Type", ContentTypeHTML)
	}

	if resp.statusCode != 0 {
		resp.w.WriteHeader(resp.statusCode)
	}

	return false
}

// sanitizeRedirectURL validates that a redirect URL is safe (relative path only).
// Returns the sanitized URL and whether it's safe to redirect.
// Blocks absolute URLs, scheme/host references, and paths that escape above root.
func sanitizeRedirectURL(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}

	if u.Scheme != "" || u.Host != "" {
		return "", false
	}

	if u.Opaque != "" {
		return "", false
	}

	cleaned := path.Clean(u.Path)

	if pathEscapesRoot(u.Path) {
		return "", false
	}

	return cleaned, u.Path != ""
}

// pathEscapesRoot checks whether a URL path contains ".." segments that would
// escape above the root. Legitimate normalizations like "/a/../b" are allowed,
// but "/../../etc/passwd" is rejected.
func pathEscapesRoot(p string) bool {
	depth := 0
	for i := 0; i < len(p); {
		if p[i] != '/' {
			j := i + 1
			for j < len(p) && p[j] != '/' {
				j++
			}
			seg := p[i:j]
			switch seg {
			case "..":
				depth--
			case ".", "":
			default:
				depth++
			}
			if depth < 0 {
				return true
			}
			i = j
		} else {
			i++
		}
	}
	return false
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
	existing := w.Header().Get(header)

	encoded, err := json.Marshal(map[string]any{name: detail})
	if err != nil {
		w.Header().Set(header, name)
		return
	}

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
