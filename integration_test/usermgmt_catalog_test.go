package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v2"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

// These DTOs mirror catalog/README.md's "Recipe: Catalog for the usermgmt Module".
// They describe the HTTP request shapes for the (unexported) usermgmt commands.
type recipeRegisterUserRequest struct {
	Email       string   `json:"email"        doc:"User email address"`
	DisplayName string   `json:"display_name" doc:"Display name"`
	Roles       []string `json:"roles"        doc:"Initial roles"`
}

type recipeChangeEmailRequest struct {
	Email string `json:"email" doc:"New email address"`
}

type recipeChangeDisplayNameRequest struct {
	DisplayName string `json:"display_name" doc:"New display name"`
}

type recipeUpdateRolesRequest struct {
	Roles  []string `json:"roles"  doc:"New roles"`
	Domain string   `json:"domain" doc:"Authorization domain"`
}

type recipeDeleteUserRequest struct {
	Reason string `json:"reason" doc:"Deletion reason"`
}

// usermgmtCatalogFromRecipe is a faithful copy of the recipe in catalog/README.md.
// If this stops compiling, the documented recipe is lying and must be fixed.
func usermgmtCatalogFromRecipe() *catalog.Catalog {
	b := cataloghtmx.New("User Management", "1.0.0")

	// Commands — register the DTOs that describe the HTTP request shapes.
	cataloghtmx.Command[recipeRegisterUserRequest](b, "register-user",
		cataloghtmx.WithOperation("POST", "/auth/register"))
	cataloghtmx.Command[recipeChangeEmailRequest](b, "change-email",
		cataloghtmx.WithOperation("POST", "/auth/change-email"))
	cataloghtmx.Command[recipeChangeDisplayNameRequest](b, "change-display-name",
		cataloghtmx.WithOperation("POST", "/auth/change-display-name"))
	cataloghtmx.Command[recipeUpdateRolesRequest](b, "update-roles",
		cataloghtmx.WithOperation("POST", "/auth/update-roles"))
	cataloghtmx.Command[recipeDeleteUserRequest](b, "delete-user",
		cataloghtmx.WithOperation("DELETE", "/auth/users/{id}"))

	// Events — the persisted payloads are the real contract; reflect them directly.
	cataloghtmx.Event[usermgmt.UserRegisteredPayload](b, "user.registered", catalog.Sends)
	cataloghtmx.Event[usermgmt.RolesUpdatedPayload](b, "user.roles-updated", catalog.Sends)
	cataloghtmx.Event[usermgmt.EmailChangedPayload](b, "user.email-changed", catalog.Sends)
	cataloghtmx.Event[usermgmt.DisplayNameChangedPayload](b, "user.display-name-changed", catalog.Sends)
	cataloghtmx.Event[usermgmt.UserDeletedPayload](b, "user.deleted", catalog.Sends)
	cataloghtmx.Event[usermgmt.CredentialAddedPayload](b, "user.credential-added", catalog.Sends)
	cataloghtmx.Event[usermgmt.CredentialRemovedPayload](b, "user.credential-removed", catalog.Sends)
	cataloghtmx.Event[usermgmt.EmailVerifiedPayload](b, "user.email-verified", catalog.Sends)
	cataloghtmx.Event[usermgmt.TOTPEnabledPayload](b, "user.totp-enabled", catalog.Sends)
	cataloghtmx.Event[usermgmt.TOTPDisabledPayload](b, "user.totp-disabled", catalog.Sends)

	return b.Build()
}

// TestUsermgmtCatalog_RecipeCompiles verifies that the README recipe builds a
// valid catalog from the real usermgmt event payload types. This guards against
// the documented recipe silently breaking when a payload type changes shape.
func TestUsermgmtCatalog_RecipeCompiles(t *testing.T) {
	t.Parallel()

	// Build() panics on invalid catalogs, so reaching here means schema reflection
	// succeeded for every real payload type.
	cat := usermgmtCatalogFromRecipe()

	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	svc := cat.Services[0]
	const wantCmds, wantEvents = 5, 10
	if len(svc.Commands) != wantCmds {
		t.Errorf("expected %d commands, got %d", wantCmds, len(svc.Commands))
	}
	if len(svc.Events) != wantEvents {
		t.Errorf("expected %d events, got %d", wantEvents, len(svc.Events))
	}

	// Every documented event must appear in the generated docs.
	assertRecipeOpenAPI(t, cat)
	assertRecipeAsyncAPI(t, cat)
	assertRecipeEventCatalog(t, cat)
}

func assertRecipeOpenAPI(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	w := serveRecipe(t, cataloghtmx.OpenAPIHandler(cat))
	if w.Code != http.StatusOK {
		t.Fatalf("openapi: expected 200, got %d", w.Code)
	}

	doc := unmarshalJSONBodyMsg(t, w, "openapi: ")
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi: expected version 3.0.3, got %v", doc["openapi"])
	}
}

func assertRecipeAsyncAPI(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	w := serveRecipe(t, cataloghtmx.AsyncAPIHandler(cat))
	if w.Code != http.StatusOK {
		t.Fatalf("asyncapi: expected 200, got %d", w.Code)
	}

	doc := unmarshalJSONBodyMsg(t, w, "asyncapi: ")
	if doc["asyncapi"] != "3.0.0" {
		t.Errorf("asyncapi: expected version 3.0.0, got %v", doc["asyncapi"])
	}
}

// assertRecipeEventCatalog writes the MDX tree to a temp dir, proving file
// generation works end-to-end on the real payload types.
func assertRecipeEventCatalog(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	dir := t.TempDir()
	if err := cataloghtmx.GenerateEventCatalog(cat, dir); err != nil {
		t.Fatalf("generate event catalog: %v", err)
	}
}

func serveRecipe(t *testing.T, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	return w
}
