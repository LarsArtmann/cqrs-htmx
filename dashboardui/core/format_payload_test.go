package core

import (
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

func TestRelativeTime(t *testing.T) {
	t.Parallel()

	t.Run("zero time returns empty", func(t *testing.T) {
		t.Parallel()

		if got := RelativeTime(time.Time{}); got != "" {
			t.Errorf("RelativeTime(zero) = %q, want empty", got)
		}
	})

	t.Run("very recent returns just now", func(t *testing.T) {
		t.Parallel()

		now := time.Now().Add(-10 * time.Second)
		got := RelativeTime(now)
		if got != "just now" {
			t.Errorf("RelativeTime(recent) = %q, want %q", got, "just now")
		}
	})

	t.Run("past time contains ago", func(t *testing.T) {
		t.Parallel()

		past := time.Now().Add(-2 * time.Hour)
		got := RelativeTime(past)
		if !strings.Contains(got, "ago") {
			t.Errorf("RelativeTime(2h ago) = %q, should contain 'ago'", got)
		}
	})
}

func TestHumanByteSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bytes int
		want  string
	}{
		{0, "0 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got := HumanByteSize(tt.bytes)
			if got != tt.want {
				t.Errorf("HumanByteSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestDefaultPayloadRenderer_JSON(t *testing.T) {
	t.Parallel()

	r := DefaultPayloadRenderer{}
	payload := []byte(`{"name":"test","value":42}`)

	out, err := r.Render(payload, codec.EncodingJSON)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	result := string(out)
	if !strings.Contains(result, "name") || !strings.Contains(result, "test") {
		t.Errorf("JSON render output should contain original keys/values, got %q", result)
	}

	if !strings.Contains(result, "  ") {
		t.Error("JSON render should be pretty-printed with 2-space indent")
	}
}

func TestDefaultPayloadRenderer_EmptyPayload(t *testing.T) {
	t.Parallel()

	r := DefaultPayloadRenderer{}
	out, err := r.Render(nil, codec.EncodingJSON)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if string(out) != "{}" {
		t.Errorf("empty payload should render as {}, got %q", string(out))
	}
}

func TestDefaultPayloadRenderer_RawEncoding(t *testing.T) {
	t.Parallel()

	r := DefaultPayloadRenderer{}
	payload := []byte("binary data here")

	out, err := r.Render(payload, codec.EncodingRaw)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if string(out) != string(payload) {
		t.Errorf("raw encoding should return payload verbatim")
	}
}

func TestDefaultPayloadRenderer_InvalidJSON(t *testing.T) {
	t.Parallel()

	r := DefaultPayloadRenderer{}
	_, err := r.Render([]byte("not json"), codec.EncodingJSON)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestRenderPayload(t *testing.T) {
	t.Parallel()

	evt := makeTestEvent("test", 1)
	bytes := RenderPayload(DefaultPayloadRenderer{}, evt)

	if len(bytes) == 0 {
		t.Error("RenderPayload should return non-empty bytes")
	}
}

func TestPrettyJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid json", func(t *testing.T) {
		t.Parallel()

		raw := []byte(`{"a":1}`)
		result := PrettyJSON(raw)

		if !strings.Contains(result, "a") {
			t.Errorf("PrettyJSON lost key, got %q", result)
		}
	})

	t.Run("invalid json returns raw", func(t *testing.T) {
		t.Parallel()

		raw := []byte("not json")
		result := PrettyJSON(raw)

		if result != string(raw) {
			t.Errorf("PrettyJSON(invalid) = %q, want %q", result, string(raw))
		}
	})
}
