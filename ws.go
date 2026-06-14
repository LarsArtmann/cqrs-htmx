package cqrshtmx

import (
	"encoding/json"
	"fmt"
	"strings"
)

// WSMessage represents an incoming HTMX WebSocket message.
//
// When a form with ws-send is submitted, HTMX serializes its fields as JSON
// and includes a HEADERS object with the headers normally sent with HTMX requests.
// ParseWSMessage extracts these into a structured type for easy access.
//
// Server-side:
//
//	func handleWS(conn websocket.Conn) {
//	    for {
//	        _, msg, err := conn.ReadMessage()
//	        if err != nil { return }
//	        parsed, err := cqrshtmx.ParseWSMessage(msg)
//	        // use parsed.Body["field_name"], parsed.Headers["HX-Request"]
//	    }
//	}
//
// Client-side (HTML):
//
//	<div hx-ext="ws" ws-connect="/ws">
//	  <form ws-send>
//	    <input name="message">
//	    <button>Send</button>
//	  </form>
//	</div>
type WSMessage struct {
	// Headers contains the HTMX headers that would normally be sent with
	// an HTTP request. Includes HX-Request, HX-Trigger, HX-Target, etc.
	Headers map[string]string

	// Body contains all form field values from the submitted form.
	// The HEADERS key is extracted separately into the Headers field.
	Body map[string]any
}

// ParseWSMessage parses an incoming HTMX WebSocket JSON message.
// The HTMX WebSocket extension sends form data as JSON with a special
// HEADERS field containing HTMX request headers.
//
// Returns a WSMessage with separated Headers and Body fields.
// Returns an error if the input is not valid JSON.
func ParseWSMessage(data []byte) (*WSMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse ws message: %w", err)
	}

	msg := &WSMessage{
		Headers: make(map[string]string),
		Body:    make(map[string]any),
	}

	for key, value := range raw {
		if key == "HEADERS" {
			if headers, ok := value.(map[string]any); ok {
				for hk, hv := range headers {
					if s, ok := hv.(string); ok {
						msg.Headers[hk] = s
					}
				}
			}
			continue
		}
		msg.Body[key] = value
	}

	return msg, nil
}

func parseWSHeaders(raw json.RawMessage) map[string]string {
	var headersMap map[string]string
	if json.Unmarshal(raw, &headersMap) == nil {
		return headersMap
	}
	var anyHeaders map[string]any
	if json.Unmarshal(raw, &anyHeaders) != nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(anyHeaders))
	for k, v := range anyHeaders {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

// StringBody returns a body field as a string.
// Returns empty string if the field doesn't exist or isn't a string.
func (m *WSMessage) StringBody(key string) string {
	v, ok := m.Body[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// WSOOBHTML wraps an HTML fragment with HTMX out-of-band swap attributes.
// The HTMX WebSocket extension parses received messages as HTML and uses
// OOB swap logic to update elements by ID.
//
// By default, elements are replaced using innerHTML. Pass a SwapStrategy
// to use a different swap method:
//
//	html := cqrshtmx.WSOOBHTML("notifications", "<div>new item</div>",
//	    cqrshtmx.SwapBeforeEnd)
func WSOOBHTML(id, html string, swapStrategy ...SwapStrategy) string {
	swap := "true"
	if len(swapStrategy) > 0 && swapStrategy[0] != "" {
		swap = string(swapStrategy[0])
	}

	// If the HTML already contains an element with the target ID and
	// hx-swap-oob, return it as-is — the consumer knows best.
	if strings.Contains(html, "hx-swap-oob") {
		return html
	}

	// Wrap the HTML in a div with OOB swap attributes.
	// HTMX will find the element with matching ID and swap it.
	return fmt.Sprintf(
		`<div id="%s" hx-swap-oob="%s">%s</div>`,
		id, swap, html,
	)
}

// ParseWSMessageInto parses an incoming HTMX WebSocket JSON message into a
// typed struct. The HEADERS field is extracted separately; all other fields
// are deserialized into the typed struct T.
//
// This is the typed alternative to ParseWSMessage for consumers who prefer
// compile-time safety over dynamic map access.
//
// Usage:
//
//	type ChatMessage struct {
//	    Room    string `json:"room"`
//	    Message string `json:"chat_message"`
//	}
//
//	msg, headers, err := cqrshtmx.ParseWSMessageInto[ChatMessage](data)
//	// msg.Room, msg.Message are typed fields
//	// headers contains HTMX headers
func ParseWSMessageInto[T any](data []byte) (msg T, headers map[string]string, err error) {
	var raw map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
		return msg, nil, fmt.Errorf("parse ws message into: %w", unmarshalErr)
	}

	// Extract HEADERS separately.
	headers = make(map[string]string)
	if headersRaw, ok := raw["HEADERS"]; ok {
		headers = parseWSHeaders(headersRaw)
		delete(raw, "HEADERS")
	}

	// Re-serialize remaining fields and unmarshal into T.
	// raw already has HEADERS removed — marshal directly without copying.
	bodyJSON, marshalErr := json.Marshal(raw)
	if marshalErr != nil {
		return msg, headers, fmt.Errorf("parse ws message into: remarshal body: %w", marshalErr)
	}

	if unmarshalErr := json.Unmarshal(bodyJSON, &msg); unmarshalErr != nil {
		return msg, headers, fmt.Errorf("parse ws message into: %w", unmarshalErr)
	}

	return msg, headers, nil
}
