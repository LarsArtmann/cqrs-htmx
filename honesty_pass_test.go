package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// The 2026-10-24 exported-contract honesty pass (master plan S33) pins one test
// per fixed contract so the old behavior cannot silently return.

// TestNewAck_SanitizesServerFaults: the ACK broadcast reaches EVERY connected
// SSE client, so rejection detail (the client contract) passes through but
// 5xx-family internals are replaced by the family default message — the same
// rule SafeDetail applies to HTTP bodies.
func TestNewAck_SanitizesServerFaults(t *testing.T) {
	t.Parallel()

	internal := errorfamily.NewInfrastructure("db.explode", "postgres://secret@db.internal:5432 users table missing")
	ack := newAck("cmd-1", internal)

	if strings.Contains(ack.Error, "postgres://") || strings.Contains(ack.Error, "db.internal") {
		t.Fatalf("ack broadcast leaks internals: %q", ack.Error)
	}

	if ack.Status != AckRejected {
		t.Fatalf("status = %q, want rejected", ack.Status)
	}

	// Rejection-family detail IS the client contract and must survive.
	rejection := errorfamily.NewRejection("email_taken", "email already registered")
	ackR := newAck("cmd-2", rejection)

	if !strings.Contains(ackR.Error, "email already registered") {
		t.Fatalf("rejection detail = %q, want the client-facing message", ackR.Error)
	}
}

// TestFormDecode_RejectsCaseCollisions: two form keys differing only by case
// used to pick a winner by map-iteration order; now the request is rejected
// deterministically.
func TestFormDecode_RejectsCaseCollisions(t *testing.T) {
	t.Parallel()

	form := url.Values{}
	form.Set("Email", "a@example.com")
	form.Set("email", "b@example.com")

	err := decodeFormValues(form, &struct {
		Email string `json:"email"`
	}{})
	if err == nil {
		t.Fatal("expected an ambiguous-key rejection, got nil")
	}

	if !strings.Contains(err.Error(), "email") {
		t.Errorf("error should name the colliding key, got: %v", err)
	}
}

// denyAllEnforcer denies everything, proving the authorize path ran.
type denyAllEnforcer struct{}

func (denyAllEnforcer) Enforce(...any) (bool, error) { return false, nil }

type noopCommand struct{ command.Command }

// TestRequireAuth_DoesNotDowngradeAuthorize: option order must not downgrade a
// permission-checked handler to merely-authenticated.
func TestRequireAuth_DoesNotDowngradeAuthorize(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("DoThing", func(_ context.Context, _ command.Command) error { return nil })

	app, err := New(Config{Commands: disp, Enforcer: denyAllEnforcer{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	h := app.Command("DoThing",
		DecodeJSON(func(_ any) (command.Command, error) { return noopCommand{}, nil }),
		Authorize("things", "write"),
		RequireAuth(), // applied AFTER Authorize — must not downgrade
	)

	r := httptest.NewRequest(http.MethodPost, "/do", strings.NewReader("{}"))
	r = r.WithContext(WithUserID(r.Context(), MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (enforcer denial proves the authorize check survived)", w.Code)
	}
}

// TestOpenAPIRoutes_DeepCopy: mutating a returned route must not reach the
// App's collected state.
func TestOpenAPIRoutes_DeepCopy(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("CreateItem", func(_ context.Context, _ command.Command) error { return nil })

	app, err := New(Config{Commands: disp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	op := openapi.Post("CreateItem").Summary("original").Op()
	op.Responses = map[int]*openapi.Response{
		400: {Description: "bad request"},
	}

	_ = app.Command("CreateItem",
		DecodeJSON(func(_ any) (command.Command, error) { return noopCommand{}, nil }),
		WithOpenAPI(op),
	)

	routes := app.OpenAPIRoutes()
	if len(routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(routes))
	}

	routes[0].Operation.Summary = "mutated"
	routes[0].Operation.Responses[400].Description = "mutated"

	fresh := app.OpenAPIRoutes()
	if fresh[0].Operation.Summary == "mutated" {
		t.Error("summary mutation reached App state — slice copy is not deep")
	}

	if fresh[0].Operation.Responses[400].Description == "mutated" {
		t.Error("responses mutation reached App state — map copy is not deep")
	}
}

// TestOOBHTML_EscapesID: the generated attribute must not be escapable via a
// hostile id.
func TestOOBHTML_EscapesID(t *testing.T) {
	t.Parallel()

	got := OOBHTML(`x" onmouseover="alert(1)`, "<p>hi</p>")
	if strings.Contains(got, `" onmouseover="`) {
		t.Fatalf("id broke out of the attribute: %s", got)
	}

	if !strings.Contains(got, "x&#34; onmouseover=&#34;alert(1)") {
		t.Fatalf("expected escaped id in output, got: %s", got)
	}
}
