package core

import (
	"strconv"
	"sync"
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type demoPayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type markerRenderer struct{ marker string }

func (r markerRenderer) RenderTyped(p demoPayload, _ codec.Encoding) ([]byte, error) {
	return []byte(r.marker + ":" + p.Name + ":" + strconv.Itoa(p.Count)), nil
}

type stubUntyped struct{ out string }

func (s stubUntyped) Render(_ []byte, _ codec.Encoding) ([]byte, error) {
	return []byte(s.out), nil
}

func typedTestEvent(t *testing.T, evtType event.Type, payload any) event.Event {
	t.Helper()

	evt, err := event.New(evtType, id.NewStreamID(), "DemoAgg", event.Version(1), payload)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}

func TestTypedRegistry_UnregisteredFallsThroughToRenderer(t *testing.T) {
	t.Parallel()

	reg := NewPayloadTypeRegistry()
	evt := typedTestEvent(t, "demo.unregistered", demoPayload{Name: "x", Count: 1})

	out, ok := reg.Render(evt.Type(), evt.Payload(), evt.Encoding())
	if ok {
		t.Fatalf("unregistered type must not render, got %q", out)
	}

	got := string(RenderPayload(stubUntyped{out: "UNTYPED"}, evt))
	if got != "UNTYPED" {
		t.Errorf("RenderPayload = %q, want the untyped renderer's output", got)
	}
}

func TestTypedRegistry_RegisteredTypeWinsOverRenderer(t *testing.T) {
	t.Parallel()

	reg := NewPayloadTypeRegistry()
	RegisterPayloadTypeIn(reg, "demo.named", markerRenderer{marker: "TYPED"})

	evt := typedTestEvent(t, "demo.named", demoPayload{Name: "file", Count: 7})

	got := string(RenderPayload(stubUntyped{out: "UNTYPED"}, evt))
	if got != "TYPED:file:7" {
		t.Errorf("RenderPayload = %q, want the typed renderer's output", got)
	}
}

func TestTypedRegistry_DecodeMismatchFallsBack(t *testing.T) {
	t.Parallel()

	reg := NewPayloadTypeRegistry()
	RegisterPayloadTypeIn(reg, "demo.drift", markerRenderer{marker: "TYPED"})

	evt := typedTestEvent(t, "demo.drift", []int{1, 2, 3})

	if _, ok := reg.Render(evt.Type(), evt.Payload(), evt.Encoding()); ok {
		t.Error("array bytes must not decode into a struct payload; typed path must report failure")
	}

	got := string(RenderPayload(stubUntyped{out: "UNTYPED"}, evt))
	if got != "UNTYPED" {
		t.Errorf("RenderPayload = %q, want fallback after decode mismatch", got)
	}
}

func TestTypedRegistry_ReRegistrationReplaces(t *testing.T) {
	t.Parallel()

	reg := NewPayloadTypeRegistry()
	RegisterPayloadTypeIn(reg, "demo.replace", markerRenderer{marker: "FIRST"})
	RegisterPayloadTypeIn(reg, "demo.replace", markerRenderer{marker: "SECOND"})

	evt := typedTestEvent(t, "demo.replace", demoPayload{Name: "x", Count: 0})

	got := string(RenderPayload(stubUntyped{out: "UNTYPED"}, evt))
	if got != "SECOND:x:0" {
		t.Errorf("RenderPayload = %q, want the last registration to win", got)
	}
}

func TestTypedRegistry_RawEncodingUnsupported(t *testing.T) {
	t.Parallel()

	reg := NewPayloadTypeRegistry()
	RegisterPayloadTypeIn(reg, "demo.raw", markerRenderer{marker: "TYPED"})

	if _, ok := reg.Render("demo.raw", []byte("raw-bytes"), codec.EncodingRaw); ok {
		t.Error("raw encoding has no typed decode; Render must report false")
	}
}

func TestTypedRegistry_NilRegistrySafe(t *testing.T) {
	t.Parallel()

	var reg *PayloadTypeRegistry

	if _, ok := reg.Render("demo.anything", []byte("{}"), codec.EncodingJSON); ok {
		t.Error("nil registry must report false, not panic")
	}
}

func TestTypedRegistry_ConcurrentRegisterAndRender(t *testing.T) {
	t.Parallel()

	reg := NewPayloadTypeRegistry()
	evt := typedTestEvent(t, "demo.race", demoPayload{Name: "r", Count: 1})

	var wg sync.WaitGroup

	for i := range 8 {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()
			RegisterPayloadTypeIn(reg, "demo.race", markerRenderer{marker: "R" + strconv.Itoa(i)})
		}(i)

		go func() {
			defer wg.Done()
			reg.Render(evt.Type(), evt.Payload(), evt.Encoding())
		}()
	}

	wg.Wait()
}

func TestRegisterPayloadType_DefaultRegistryConsultedByRenderPayload(t *testing.T) {
	t.Parallel()

	const evtType = event.Type("demo.default-registry.v1")
	RegisterPayloadType(evtType, markerRenderer{marker: "GLOBAL"})

	evt := typedTestEvent(t, evtType, demoPayload{Name: "g", Count: 2})

	got := string(RenderPayload(stubUntyped{out: "UNTYPED"}, evt))
	if got != "GLOBAL:g:2" {
		t.Errorf("RenderPayload = %q, want the default-registry renderer", got)
	}
}

func TestTypedRegistry_UnregisteredKeepsDefaultPrettyJSON(t *testing.T) {
	t.Parallel()

	evt := typedTestEvent(t, "demo.pretty", demoPayload{Name: "p", Count: 3})

	got := string(RenderPayload(DefaultPayloadRenderer{}, evt))
	want := "{\n  \"name\": \"p\",\n  \"count\": 3\n}"
	if got != want {
		t.Errorf("RenderPayload = %q, want default pretty JSON %q", got, want)
	}
}
