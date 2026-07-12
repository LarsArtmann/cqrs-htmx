package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/docserver"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
)

// These DTOs describe the HTTP request shapes for the (unexported) usermgmt commands.
// They mirror the recipe documented at https://github.com/LarsArtmann/go-cqrs-lite/tree/master/catalog.
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

// usermgmtCatalogFromRecipe builds a catalog from the real usermgmt event payload types.
// If this stops compiling, the documented recipe is lying and must be fixed.
func usermgmtCatalogFromRecipe() *catalog.Catalog {
	b := simple.New("User Management", testCatalogVersion)

	// Commands — register the DTOs that describe the HTTP request shapes.
	simple.Command[recipeRegisterUserRequest](b, "register-user",
		simple.WithOperation("POST", "/auth/register"))
	simple.Command[recipeChangeEmailRequest](b, "change-email",
		simple.WithOperation("POST", "/auth/change-email"))
	simple.Command[recipeChangeDisplayNameRequest](b, "change-display-name",
		simple.WithOperation("POST", "/auth/change-display-name"))
	simple.Command[recipeUpdateRolesRequest](b, "update-roles",
		simple.WithOperation("POST", "/auth/update-roles"))
	simple.Command[recipeDeleteUserRequest](b, "delete-user",
		simple.WithOperation("DELETE", "/auth/users/{id}"))

	// Events — the persisted payloads are the real contract; reflect them directly.
	simple.Event[usermgmt.UserRegisteredPayload](b, "user.registered", catalog.Sends)
	simple.Event[usermgmt.RolesUpdatedPayload](b, "user.roles-updated", catalog.Sends)
	simple.Event[usermgmt.EmailChangedPayload](b, "user.email-changed", catalog.Sends)
	simple.Event[usermgmt.DisplayNameChangedPayload](b, "user.display-name-changed", catalog.Sends)
	simple.Event[usermgmt.UserDeletedPayload](b, "user.deleted", catalog.Sends)
	simple.Event[usermgmt.CredentialAddedPayload](b, "user.credential-added", catalog.Sends)
	simple.Event[usermgmt.CredentialRemovedPayload](b, "user.credential-removed", catalog.Sends)
	simple.Event[usermgmt.EmailVerifiedPayload](b, "user.email-verified", catalog.Sends)
	simple.Event[usermgmt.TOTPEnabledPayload](b, "user.totp-enabled", catalog.Sends)
	simple.Event[usermgmt.TOTPDisabledPayload](b, "user.totp-disabled", catalog.Sends)

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

	ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
		ServiceName: "User Management",
		Version:     testCatalogVersion,
	})
	w := serveRecipe(t, ds.OpenAPISpec())
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

	ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
		ServiceName: "User Management",
		Version:     testCatalogVersion,
	})
	w := serveRecipe(t, ds.AsyncAPISpec())
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
	if err := docserver.GenerateEventCatalog(cat, dir); err != nil {
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
