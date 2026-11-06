package core

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-codec"
)

// TestDefaultPayloadRenderer_DiagnoseFallback pins the A19 degraded paths:
// a payload whose stamped encoding fails to decode renders as CBOR extended
// diagnostic notation (EDN) instead of surfacing binary bytes in the UI —
// both the lying-JSON-stamp case (metadata-only WithEncoding; event.New
// still CBOR-encodes map payloads) and the corrupt-CBOR case.
func TestDefaultPayloadRenderer_DiagnoseFallback(t *testing.T) {
	t.Parallel()

	cborBytes, err := codec.CBORCodec{}.Encode(map[string]any{"email": "diagnose@example.com"})
	if err != nil {
		t.Fatalf("encode CBOR fixture: %v", err)
	}

	r := DefaultPayloadRenderer{}

	t.Run("LyingJSONStampRendersDiagnosticNotation", func(t *testing.T) {
		t.Parallel()

		out, err := r.Render(cborBytes, codec.EncodingJSON)
		if err != nil {
			t.Fatalf("Render with lying json stamp: %v", err)
		}

		got := string(out)
		if !strings.Contains(got, "email") || !strings.Contains(got, "diagnose@example.com") {
			t.Errorf("diagnostic notation lost the payload content: %q", got)
		}

		if got == string(cborBytes) {
			t.Error("degraded path returned the raw binary bytes, want EDN")
		}
	})

	t.Run("ValidCBORStillPrettyPrintsJSON", func(t *testing.T) {
		t.Parallel()

		out, err := r.Render(cborBytes, codec.EncodingCBOR)
		if err != nil {
			t.Fatalf("Render valid CBOR: %v", err)
		}

		if !strings.Contains(string(out), `"email"`) {
			t.Errorf("happy path should stay pretty-printed JSON, got %q", string(out))
		}
	})

	t.Run("UndiagnosableGarbageKeepsCorruptionError", func(t *testing.T) {
		t.Parallel()

		if _, err := r.Render([]byte{0xFF, 0x00, 0x01, 0x02}, codec.EncodingJSON); err == nil {
			t.Error("non-JSON non-CBOR garbage should surface the decode error (RenderPayload then shows raw bytes)")
		}
	})
}
