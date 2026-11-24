package adminui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// credentialRestoreConfirmCopy is the danger-gated escalation the restore
// CARD must carry in data-confirm when the user-detail pack wires the UI:
// a restored credential is a fully trusted factor minted from a file, and
// the confirm must say so before the admin commits. Pinned here so the card
// inherits the contract, not a paraphrase.
const credentialRestoreConfirmCopy = "Restore a passkey from backup JSON? " +
	"The uploaded credential becomes a trusted sign-in factor immediately — " +
	"restore it only from a backup you control."

func TestCredentialRestore_EscalationCopyPinned(t *testing.T) {
	for _, want := range []string{"trusted sign-in factor", "backup you control"} {
		if !strings.Contains(credentialRestoreConfirmCopy, want) {
			t.Errorf("escalation copy missing %q", want)
		}
	}
}

// F170: credential restore from backup JSON (the route the T23 agent had to
// cut because routes.go was owned elsewhere). Per-error honesty: every
// refusal names what failed, and success is asserted through the read model.

func credentialBackupJSON(t *testing.T, id []byte, name string) string {
	t.Helper()
	b, err := json.Marshal(identitymodel.WebAuthnCredential{
		CredentialCore: identitymodel.CredentialCore{ID: id, Name: name},
	})
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	return string(b)
}

func postRestore(t *testing.T, h http.Handler, userID string, backup string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"backup": []string{backup}}
	req := httptest.NewRequest(
		http.MethodPost,
		"/admin/users/"+userID+"/credentials/restore",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCredentialRestore_Success(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	target := mustRegister(t, svc, "u1", "u1@acme.dev")

	backup := credentialBackupJSON(t, []byte("cred-id-1"), "backup passkey")
	rec := postRestore(t, h, target.User.ID.Get().String(), backup)

	if msg := toastMessage(t, rec); !strings.Contains(msg, "Credential restored") {
		t.Errorf("success toast missing, got: %s", msg)
	}
	found := false
	// The read model is the truth: reload via the service's FindByUserID.
	if u, ok := svc.ReadModel().FindByUserID(target.User.ID); ok {
		for _, cred := range u.Credentials {
			if string(cred.ID) == "cred-id-1" {
				found = true
			}
		}
	}
	if !found {
		t.Error("restored credential must appear in the user's credentials")
	}
}

func TestCredentialRestore_InvalidJSONHonest(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	target := mustRegister(t, svc, "u1", "u1@acme.dev")

	rec := postRestore(t, h, target.User.ID.Get().String(), "{not json")

	if u, ok := svc.ReadModel().FindByUserID(target.User.ID); ok && len(u.Credentials) > 0 {
		t.Error("a refused restore must not add credentials")
	}
	if msg := toastMessage(t, rec); !strings.Contains(msg, "not valid credential JSON") {
		t.Errorf("refusal must name the bad JSON, got toast: %s", msg)
	}
}

func TestCredentialRestore_EmptyIDHonest(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	target := mustRegister(t, svc, "u1", "u1@acme.dev")

	rec := postRestore(t, h, target.User.ID.Get().String(), credentialBackupJSON(t, nil, "no id"))

	if u, ok := svc.ReadModel().FindByUserID(target.User.ID); ok && len(u.Credentials) > 0 {
		t.Error("a credential without an id must be refused")
	}
	if msg := toastMessage(t, rec); !strings.Contains(msg, "credential id") {
		t.Errorf("refusal must name the missing credential id, got toast: %s", msg)
	}
}
