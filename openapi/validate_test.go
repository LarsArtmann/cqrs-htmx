package openapi

import (
	"strings"
	"testing"
)

// The 2026-10-24 builder-validity pins (master plan S34): invalid specs must
// fail loudly, never silently serialize.

func TestValidate_RejectsOperationWithoutResponses(t *testing.T) {
	t.Parallel()

	spec := New("T", "1.0.0").
		Path("/items", Post("CreateItem").Summary("no responses declared"))

	err := spec.Validate()
	if err == nil {
		t.Fatal("expected a no-responses violation, got nil")
	}

	if !strings.Contains(err.Error(), "no responses") {
		t.Errorf("violation should name the missing responses, got: %v", err)
	}

	if _, jsonErr := spec.JSON(); jsonErr == nil {
		t.Error("JSON() must refuse to serialize an invalid spec")
	}
}

func TestValidate_RejectsDuplicateOperationIDs(t *testing.T) {
	t.Parallel()

	spec := New("T", "1.0.0").
		Path("/a", Get("Same").Response(200, "OK")).
		Path("/b", Get("Same").Response(200, "OK"))

	err := spec.Validate()
	if err == nil {
		t.Fatal("expected a duplicate-operationId violation, got nil")
	}

	if !strings.Contains(err.Error(), `duplicate operationId "Same"`) {
		t.Errorf("violation should name the duplicate id, got: %v", err)
	}
}

func TestValidate_AcceptsSoundSpec(t *testing.T) {
	t.Parallel()

	spec := New("T", "1.0.0").
		Schema("Item", Object(Prop("id", String()))).
		Path("/items",
			Post("CreateItem").
				JSONBody(Object(Prop("name", String().WithMinLength(1)))).
				Response(201, "Created", JSON(Ref("Item"))),
		).
		Path("/items/{id}",
			Get("GetItem").
				PathParam("id", String(), "the item id").
				Response(200, "OK", JSON(Ref("Item"))),
		)

	if err := spec.Validate(); err != nil {
		t.Fatalf("sound spec rejected: %v", err)
	}

	if _, err := spec.JSON(); err != nil {
		t.Fatalf("JSON: %v", err)
	}
}

func TestPath_DoubleMethodRegistrationPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "already registered") {
			t.Fatalf("expected a double-registration panic, got %v", r)
		}
	}()

	_ = New("T", "1.0.0").
		Path("/items", Get("List").Response(200, "OK")).
		Path("/items", Get("ListAgain").Response(200, "OK"))
}

func TestResponse_MultipleMediaTypesPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "ResponseWithContent") {
			t.Fatalf("expected a pointer to ResponseWithContent, got %v", r)
		}
	}()

	_ = Post("X").Response(200, "OK", JSON(String()), JSON(Integer()))
}

func TestResponseWithContent_ExplicitMediaTypes(t *testing.T) {
	t.Parallel()

	op := Post("Push").
		ResponseWithContent(200, "live feed", map[string]*MediaType{
			EventStreamContentType: EventStream(),
			HTMLContentType:        HTML(String()),
		}).Op()

	got := op.Responses[200]
	if got == nil {
		t.Fatal("missing 200 response")
	}

	if _, ok := got.Content["text/event-stream"]; !ok {
		t.Errorf("expected text/event-stream key, got %v", got.Content)
	}

	if _, ok := got.Content["text/html"]; !ok {
		t.Errorf("expected text/html key, got %v", got.Content)
	}
}
