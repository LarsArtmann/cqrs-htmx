package cqrshtmx

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"
)

// The ADR-0056 wire DTO hardening (master plan S44): typed vocabulary plus
// coherence-checked decoding, with zero wire-byte change.

func TestSyncPushResult_StatusUnionCoherence(t *testing.T) {
	t.Parallel()

	valid := []string{
		`{"commandId":"a","status":"confirmed"}`,
		`{"commandId":"b","status":"rejected","error":{"message":"nope","family":"rejection"}}`,
	}
	for _, body := range valid {
		var r SyncPushResult
		if err := json.Unmarshal([]byte(body), &r); err != nil {
			t.Errorf("valid body rejected: %s: %v", body, err)
		}
	}

	invalid := map[string]string{
		"confirmed with error": `{"commandId":"a","status":"confirmed","error":{"message":"x","family":"rejection"}}`,
		"rejected without":     `{"commandId":"a","status":"rejected"}`,
		"unknown status":       `{"commandId":"a","status":"maybe"}`,
	}
	for name, body := range invalid {
		var r SyncPushResult
		if err := json.Unmarshal([]byte(body), &r); err == nil {
			t.Errorf("%s decoded without complaint: %s", name, body)
		}
	}
}

func TestSyncEvent_PayloadExclusiveOr(t *testing.T) {
	t.Parallel()

	valid := []string{
		`{"eventId":"1","type":"T","streamType":"User","streamId":"u1","version":1,"occurredAt":"t","payloadEncoding":"json","payload":{"a":1}}`,
		`{"eventId":"2","type":"T","streamType":"User","streamId":"u2","version":2,"occurredAt":"t","payloadEncoding":"cbor","payloadB64":"AAE="}`,
		`{"eventId":"3","type":"T","streamType":"User","streamId":"u3","version":3,"occurredAt":"t","payloadEncoding":"opaque","payloadB64":"AAE="}`,
	}
	for _, body := range valid {
		var e SyncEvent
		if err := json.Unmarshal([]byte(body), &e); err != nil {
			t.Errorf("valid event rejected: %s: %v", body, err)
		}
	}

	invalid := map[string]string{
		"both payloads":     `{"eventId":"1","type":"T","streamType":"User","streamId":"u","version":1,"occurredAt":"t","payloadEncoding":"json","payload":{"a":1},"payloadB64":"AAE="}`,
		"neither payload":   `{"eventId":"1","type":"T","streamType":"User","streamId":"u","version":1,"occurredAt":"t","payloadEncoding":"json"}`,
		"json promises b64": `{"eventId":"1","type":"T","streamType":"User","streamId":"u","version":1,"occurredAt":"t","payloadEncoding":"json","payloadB64":"AAE="}`,
		"inline not json":   `{"eventId":"1","type":"T","streamType":"User","streamId":"u","version":1,"occurredAt":"t","payloadEncoding":"cbor","payload":{"a":1}}`,
	}
	for name, body := range invalid {
		var e SyncEvent
		if err := json.Unmarshal([]byte(body), &e); err == nil {
			t.Errorf("%s decoded without complaint: %s", name, body)
		}
	}
}

// TestSyncWire_ZeroByteChange pins the wire contract: the typed constants and
// custom unmarshalers marshal to exactly the bytes the bare strings produced.
func TestSyncWire_ZeroByteChange(t *testing.T) {
	t.Parallel()

	push, err := json.Marshal(SyncPushResult{
		CommandID: "cmd-1",
		Status:    SyncPushStatusRejected,
		Error:     &SyncPushError{Message: "nope", Family: "rejection"},
	})
	if err != nil {
		t.Fatalf("marshal push: %v", err)
	}

	want := `{"commandId":"cmd-1","status":"rejected","error":{"message":"nope","family":"rejection"}}`
	if string(push) != want {
		t.Fatalf("push wire drifted:\n got %s\nwant %s", push, want)
	}

	evt, err := json.Marshal(SyncEvent{
		EventID: "1", Type: "T", StreamType: "User", StreamID: "u", Version: 1,
		OccurredAt: "t", PayloadEncoding: SyncPayloadEncodingJSON, Payload: jsontext.Value(`{"a":1}`),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if !strings.Contains(string(evt), `"payloadEncoding":"json"`) ||
		!strings.Contains(string(evt), `"payload":{"a":1}`) {
		t.Fatalf("event wire drifted: %s", evt)
	}
}
