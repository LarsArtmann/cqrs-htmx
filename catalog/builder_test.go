package cataloghtmx_test

import (
	"testing"

	cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

type testCmd struct {
	Email string `json:"email" doc:"Email address"`
}

type testQuery struct {
	ID string `json:"id" doc:"User ID"`
}

type testEvent struct {
	UserID string `json:"user_id" doc:"User identifier"`
}

// buildTestCatalog returns a Builder populated with the standard test
// registrations used across the catalog test suite. The title and service ID
// are the only knobs callers typically vary.
func buildTestCatalog(title, serviceID string) *cataloghtmx.Builder {
	b := cataloghtmx.New(title, "1.0.0", cataloghtmx.WithServiceID(serviceID))
	cataloghtmx.Command[testCmd](
		b, "create-thing",
		cataloghtmx.WithOperation("POST", "/api/things"),
	)
	cataloghtmx.Query[testQuery](
		b, "get-thing",
		cataloghtmx.WithOperation("GET", "/api/things/{id}"),
	)
	cataloghtmx.Event[testEvent](b, "thing.created", catalog.Sends)
	return b
}

func assertCatalogCommandCount(t *testing.T, cat *catalog.Catalog, want int, msg string) {
	t.Helper()
	if got := len(cat.Services[0].Commands); got != want {
		t.Fatalf("%s: got %d, want %d", msg, got, want)
	}
}

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("User Service", "1.0.0")

	if b == nil {
		t.Fatal("New returned nil")
	}
}

func TestNew_WithServiceID(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New(
		"User Service", "1.0.0",
		cataloghtmx.WithServiceID("custom-id"),
	)

	cat := b.Build()

	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	if string(cat.Services[0].ID) != "custom-id" {
		t.Errorf("expected service ID 'custom-id', got %q", cat.Services[0].ID)
	}
}

func TestNew_DefaultServiceID_Kebab(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("User Service", "1.0.0")
	cat := b.Build()

	if string(cat.Services[0].ID) != "user-service" {
		t.Errorf("expected 'user-service', got %q", cat.Services[0].ID)
	}
}

func TestCommand_Registration(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Command[testCmd](b, "create-thing")
	cat := b.Build()

	assertCatalogCommandCount(t, cat, 1, "expected 1 command")

	cmd := cat.Services[0].Commands[0]
	if string(cmd.ID) != "create-thing" {
		t.Errorf("expected ID 'create-thing', got %q", cmd.ID)
	}

	if string(cmd.Name) != "Test" {
		t.Errorf("expected auto-derived name 'Test', got %q", cmd.Name)
	}
}

func TestQuery_Registration(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Query[testQuery](b, "get-thing")
	cat := b.Build()

	if len(cat.Services[0].Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(cat.Services[0].Queries))
	}
}

func TestEvent_Registration(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Event[testEvent](b, "thing.created", catalog.Sends)
	cat := b.Build()

	if len(cat.Services[0].Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(cat.Services[0].Events))
	}

	evt := cat.Services[0].Events[0]
	if evt.Direction != catalog.Sends {
		t.Errorf("expected direction Sends, got %q", evt.Direction)
	}
}

func TestEvent_DirectionRequired(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Event[testEvent](b, "thing.received", catalog.Receives)
	cat := b.Build()

	evt := cat.Services[0].Events[0]
	if evt.Direction != catalog.Receives {
		t.Errorf("expected direction Receives, got %q", evt.Direction)
	}
}

func TestAddMessage_Alternative(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	b.AddMessage(catalog.Command[testCmd]("via-add-message"))
	cat := b.Build()

	assertCatalogCommandCount(t, cat, 1, "expected 1 command via AddMessage")
}

func TestBuild_MultipleMessages(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Multi", "2.0.0")
	cataloghtmx.Command[testCmd](b, "cmd-1")
	cataloghtmx.Command[testCmd](b, "cmd-2")
	cataloghtmx.Query[testQuery](b, "qry-1")
	cataloghtmx.Event[testEvent](b, "evt-1", catalog.Sends)
	cataloghtmx.Event[testEvent](b, "evt-2", catalog.Receives)
	cat := b.Build()

	if len(cat.Services[0].Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(cat.Services[0].Commands))
	}

	if len(cat.Services[0].Queries) != 1 {
		t.Errorf("expected 1 query, got %d", len(cat.Services[0].Queries))
	}

	if len(cat.Services[0].Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(cat.Services[0].Events))
	}
}

func TestWithOperation(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Command[testCmd](
		b, "create",
		cataloghtmx.WithOperation("POST", "/api/create"),
	)
	cat := b.Build()

	cmd := cat.Services[0].Commands[0]
	if cmd.Operation == nil {
		t.Fatal("expected operation to be set")
	}

	if string(cmd.Operation.Method) != "POST" {
		t.Errorf("expected method POST, got %q", cmd.Operation.Method)
	}

	if cmd.Operation.Path != "/api/create" {
		t.Errorf("expected path /api/create, got %q", cmd.Operation.Path)
	}
}

func TestWithServiceSummary(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New(
		"Test", "1.0.0",
		cataloghtmx.WithServiceSummary("A test service"),
	)
	cat := b.Build()

	if string(cat.Services[0].Summary) != "A test service" {
		t.Errorf("expected summary, got %q", cat.Services[0].Summary)
	}
}

func TestChaining(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	result := cataloghtmx.Command[testCmd](b, "chained")

	if result != b {
		t.Error("expected Command to return the same builder for chaining")
	}
}

func TestSchemaDerivation(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Command[testCmd](b, "with-schema")
	cat := b.Build()

	cmd := cat.Services[0].Commands[0]
	if cmd.Schema == nil {
		t.Fatal("expected schema to be auto-derived")
	}

	emailProp, ok := cmd.Schema.Properties["email"]
	if !ok {
		t.Fatal("expected 'email' property in schema")
	}

	if emailProp.Description != "Email address" {
		t.Errorf("expected description 'Email address', got %q", emailProp.Description)
	}
}

func TestWithServiceName(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Default", "1.0.0", cataloghtmx.WithServiceName("Custom Name"))
	cat := b.Build()

	if string(cat.Services[0].Name) != "Custom Name" {
		t.Errorf("expected service name 'Custom Name', got %q", cat.Services[0].Name)
	}
}

func TestBuildValid_ValidCatalog(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Command[testCmd](b, "create-thing")

	cat, violations := b.BuildValid()
	if len(violations) != 0 {
		t.Fatalf("expected no violations for valid catalog, got %d: %v", len(violations), violations)
	}

	if cat == nil {
		t.Fatal("expected non-nil catalog")
	}
}

func TestBuildValid_DuplicateMessageIDs(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Command[testCmd](b, "dup-id")
	cataloghtmx.Command[testCmd](b, "dup-id")

	_, violations := b.BuildValid()
	if len(violations) == 0 {
		t.Fatal("expected violations for duplicate message IDs, got none")
	}
}

func TestBuild_PanicsOnDuplicateMessageIDs(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	cataloghtmx.Command[testCmd](b, "dup-id")
	cataloghtmx.Command[testCmd](b, "dup-id")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Build to panic on duplicate message IDs")
		}
	}()

	b.Build()
}

func TestRegistry(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	if b.Registry() == nil {
		t.Fatal("expected non-nil Registry")
	}
}

func TestInnerBuilder(t *testing.T) {
	t.Parallel()

	b := cataloghtmx.New("Test", "1.0.0")
	if b.InnerBuilder() == nil {
		t.Fatal("expected non-nil InnerBuilder")
	}
}
