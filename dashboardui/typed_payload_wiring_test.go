package dashboardui

import (
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type wirePayload struct {
	Name string `json:"name"`
}

type wireRenderer struct{ marker string }

func (r wireRenderer) RenderTyped(p wirePayload, _ codec.Encoding) ([]byte, error) {
	return []byte(r.marker + ":" + p.Name), nil
}

type wireUntyped struct{ out string }

func (s wireUntyped) Render(_ []byte, _ codec.Encoding) ([]byte, error) {
	return []byte(s.out), nil
}

func wireTestEvent(t *testing.T, evtType event.Type, payload any) event.Event {
	t.Helper()

	evt, err := event.New(evtType, id.NewStreamID(), "WireAgg", event.Version(1), payload)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}

func TestPayloadBytes_TypedRegistryChain(t *testing.T) {
	t.Parallel()

	t.Run("registered type wins over the renderer", func(t *testing.T) {
		t.Parallel()

		reg := NewPayloadTypeRegistry()

		RegisterPayloadTypeIn(reg, "wire.named", wireRenderer{marker: "TYPED"})

		cfg := Config{PayloadTypes: reg, PayloadRenderer: wireUntyped{out: "UNTYPED"}}
		evt := wireTestEvent(t, "wire.named", wirePayload{Name: "a"})

		if got := string(payloadBytes(cfg, evt)); got != "TYPED:a" {
			t.Errorf("payloadBytes = %q, want the typed renderer's output", got)
		}
	})

	t.Run("decode mismatch falls back to the renderer", func(t *testing.T) {
		t.Parallel()

		reg := NewPayloadTypeRegistry()

		RegisterPayloadTypeIn(reg, "wire.drift", wireRenderer{marker: "TYPED"})

		cfg := Config{PayloadTypes: reg, PayloadRenderer: wireUntyped{out: "UNTYPED"}}
		evt := wireTestEvent(t, "wire.drift", []int{1})

		if got := string(payloadBytes(cfg, evt)); got != "UNTYPED" {
			t.Errorf("payloadBytes = %q, want the renderer fallback", got)
		}
	})

	t.Run("unregistered type uses the renderer", func(t *testing.T) {
		t.Parallel()

		cfg := Config{PayloadTypes: NewPayloadTypeRegistry(), PayloadRenderer: wireUntyped{out: "UNTYPED"}}
		evt := wireTestEvent(t, "wire.other", wirePayload{Name: "b"})

		if got := string(payloadBytes(cfg, evt)); got != "UNTYPED" {
			t.Errorf("payloadBytes = %q, want the renderer output", got)
		}
	})

	t.Run("nil registry defaults to the package registry", func(t *testing.T) {
		t.Parallel()

		const evtType = event.Type("wire.default.v1")

		RegisterPayloadType(evtType, wireRenderer{marker: "GLOBAL"})

		cfg := Config{PayloadRenderer: wireUntyped{out: "UNTYPED"}}
		evt := wireTestEvent(t, evtType, wirePayload{Name: "c"})

		if got := string(payloadBytes(cfg, evt)); got != "GLOBAL:c" {
			t.Errorf("payloadBytes = %q, want the package-registry output", got)
		}
	})
}

func TestConfig_WithDefaultsResolvesPayloadTypes(t *testing.T) {
	t.Parallel()

	cfg, err := Config{EventSource: memorystorage.NewMemoryStore()}.withDefaults()
	if err != nil {
		t.Fatalf("withDefaults: %v", err)
	}

	if cfg.PayloadTypes == nil {
		t.Fatal("withDefaults must resolve a nil PayloadTypes to the package registry")
	}

	explicit := NewPayloadTypeRegistry()

	cfg2, err := Config{EventSource: memorystorage.NewMemoryStore(), PayloadTypes: explicit}.withDefaults()
	if err != nil {
		t.Fatalf("withDefaults: %v", err)
	}

	if cfg2.PayloadTypes != explicit {
		t.Error("withDefaults must keep an explicitly provided registry")
	}
}
