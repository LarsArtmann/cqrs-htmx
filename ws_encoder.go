package cqrshtmx

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
)

// WriteWSMessage serializes a WSMessage to the HTMX WebSocket JSON format and
// writes it to w. This is the outbound counterpart to ParseWSMessage.
//
// The output format merges Headers under the "HEADERS" key with Body fields:
//
//	{"HEADERS": {"HX-Request": "true"}, "field1": "value1"}
//
// Use this to send structured messages to HTMX WS clients that expect the
// same JSON shape as incoming messages.
func WriteWSMessage(w io.Writer, msg WSMessage) error {
	combined := make(map[string]any, len(msg.Body)+1)

	maps.Copy(combined, msg.Body)

	if len(msg.Headers) > 0 {
		combined["HEADERS"] = msg.Headers
	}

	data, err := json.Marshal(combined)
	if err != nil {
		return fmt.Errorf("write ws message: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write ws message: %w", err)
	}

	return nil
}

// WriteWSMessageInto serializes a typed body struct with separate headers to
// the HTMX WebSocket JSON format. This is the outbound counterpart to
// ParseWSMessageInto[T].
//
// The body struct is merged with a "HEADERS" key containing the headers map:
//
//	{"HEADERS": {"HX-Request": "true"}, "name": "Alice", "age": 30}
//
// Example:
//
//	type ChatMsg struct {
//	    Channel string `json:"channel"`
//	    Text    string `json:"text"`
//	}
//
//	err := cqrshtmx.WriteWSMessageInto(conn, ChatMsg{
//	    Channel: "general",
//	    Text:    "hello",
//	}, map[string]string{"HX-Request": "true"})
func WriteWSMessageInto[T any](w io.Writer, body T, headers map[string]string) error {
	bodyData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("write ws message into: marshal body: %w", err)
	}

	var combined map[string]any
	if err := json.Unmarshal(bodyData, &combined); err != nil {
		return fmt.Errorf("write ws message into: unmarshal body: %w", err)
	}

	if len(headers) > 0 {
		combined["HEADERS"] = headers
	}

	data, err := json.Marshal(combined)
	if err != nil {
		return fmt.Errorf("write ws message into: marshal combined: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write ws message into: write: %w", err)
	}

	return nil
}
