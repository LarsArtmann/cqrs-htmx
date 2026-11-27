package usermgmt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestService_ChangeEmail_Success(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "old@test.com")
	err := svc.ChangeEmail(ctx, NewUserID("u1"), "new@test.com")
	if err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}

	user, _ := svc.GetUser(ctx, NewUserID("u1"))
	if user.Email != "new@test.com" {
		t.Errorf("email = %q, want new@test.com", user.Email)
	}
}

func TestService_ChangeEmail_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	assertChangeEmailError(t, svc, NewUserID("ghost"), "ghost@test.com")
}

// assertUserIntactAfterRejectedChangeEmail is the regression check that
// matters most: a refused ChangeEmail must leave the user exactly as it was.
// ChangeEmail(id, "") used to return nil and blank the stored email — and
// since UserState.Exists() is `Email != ""`, every later command on that user
// answered *.not_found. The user must still exist and still be writable.
func assertUserIntactAfterRejectedChangeEmail(t *testing.T, svc *Service, userID UserID, wantEmail string) {
	t.Helper()
	ctx := context.Background()

	user, err := svc.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetUser after rejected ChangeEmail: %v (user was erased)", err)
	}
	if user.Email != wantEmail {
		t.Errorf("email = %q, want %q (a rejected change must not touch it)", user.Email, wantEmail)
	}
	if err := svc.ChangeDisplayName(ctx, userID, "Still Here"); err != nil {
		t.Fatalf("ChangeDisplayName after rejected ChangeEmail: %v (user no longer exists)", err)
	}
}

// TestService_ChangeEmail_EmptyEmail pins that an empty address is a
// validation error — the same shape Register answers with — instead of a
// silent success that blanked the user.
func TestService_ChangeEmail_EmptyEmail(t *testing.T) {
	t.Parallel()

	svc, ctx, _ := newTestServiceWithUser(t, "u1", "empty-change@test.com")
	t.Cleanup(svc.Stop)

	err := svc.ChangeEmail(ctx, NewUserID("u1"), "")
	assertErrorIs(t, err, ErrValidation, `ChangeEmail(id, "")`)

	assertUserIntactAfterRejectedChangeEmail(t, svc, NewUserID("u1"), "empty-change@test.com")
}

// TestService_ChangeEmail_WhitespaceEmail covers the whitespace spellings
// that slipped past the old `email == ""` comparisons.
func TestService_ChangeEmail_WhitespaceEmail(t *testing.T) {
	t.Parallel()

	for _, tc := range blankEmailCases() {
		if tc.email == "" { // the empty spelling has its own test
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, ctx, _ := newTestServiceWithUser(t, "u1", "blank-change@test.com")
			t.Cleanup(svc.Stop)

			err := svc.ChangeEmail(ctx, NewUserID("u1"), tc.email)
			assertErrorIs(t, err, ErrValidation, "whitespace-only email")

			assertUserIntactAfterRejectedChangeEmail(t, svc, NewUserID("u1"), "blank-change@test.com")
		})
	}
}

// TestService_ChangeEmail_InvalidEmail pins that a non-empty but unparseable
// address is refused before any command is dispatched.
func TestService_ChangeEmail_InvalidEmail(t *testing.T) {
	t.Parallel()

	svc, ctx, _ := newTestServiceWithUser(t, "u1", "invalid-change@test.com")
	t.Cleanup(svc.Stop)

	err := svc.ChangeEmail(ctx, NewUserID("u1"), "not-an-email")
	assertErrorIs(t, err, ErrValidation, "malformed email")

	assertUserIntactAfterRejectedChangeEmail(t, svc, NewUserID("u1"), "invalid-change@test.com")
	if holders := countUsersWithEmail(svc, "not-an-email"); holders != 0 {
		t.Errorf("rows holding %q = %d, want 0", "not-an-email", holders)
	}
}

// TestService_ChangeEmail_DuplicateEmail pins that an address owned by a
// DIFFERENT user is refused with the same answer Register gives
// (usermgmt.email_exists / ErrEmailExists) and that neither user changes.
func TestService_ChangeEmail_DuplicateEmail(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()
	registerTestUser(t, svc, "u1", "first@test.com")
	registerTestUser(t, svc, "u2", "second@test.com")

	err := svc.ChangeEmail(ctx, NewUserID("u1"), "second@test.com")
	assertErrorIs(t, err, ErrEmailExists, "address owned by another user")
	ef := requireDomainError(t, err, errorfamily.Rejection, "usermgmt.email_exists")
	if got := ef.ErrorContext()["user_id"]; got != NewUserID("u1").Get().String() {
		t.Errorf("user_id context = %q, want the requesting user %q", got, NewUserID("u1").Get().String())
	}

	if holders := countUsersWithEmail(svc, "second@test.com"); holders != 1 {
		t.Errorf("rows holding second@test.com = %d, want exactly 1", holders)
	}
	assertUserIntactAfterRejectedChangeEmail(t, svc, NewUserID("u1"), "first@test.com")
}

// TestService_ChangeEmail_CaseInsensitiveDuplicate: addresses are normalised
// (trimmed, lower-cased), so a differently-spelled duplicate is still a
// duplicate.
func TestService_ChangeEmail_CaseInsensitiveDuplicate(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()
	registerTestUser(t, svc, "u1", "first@test.com")
	registerTestUser(t, svc, "u2", "Case@Test.COM")

	err := svc.ChangeEmail(ctx, NewUserID("u1"), "  CASE@test.com  ")
	assertErrorIs(t, err, ErrEmailExists, "case-insensitive duplicate")

	if holders := countUsersWithEmail(svc, "case@test.com"); holders != 1 {
		t.Errorf("rows holding case@test.com = %d, want exactly 1", holders)
	}
	assertUserIntactAfterRejectedChangeEmail(t, svc, NewUserID("u1"), "first@test.com")
}

// TestService_ChangeEmail_NormalisesEmail pins that the stored address is the
// ParseEmail form — trimmed and lower-cased — exactly like Register stores it.
func TestService_ChangeEmail_NormalisesEmail(t *testing.T) {
	t.Parallel()

	svc, ctx, _ := newTestServiceWithUser(t, "u1", "before@test.com")
	t.Cleanup(svc.Stop)

	if err := svc.ChangeEmail(ctx, NewUserID("u1"), "  New@Example.COM "); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}

	user, err := svc.GetUser(ctx, NewUserID("u1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("stored email = %q, want the normalised %q", user.Email, "new@example.com")
	}
	if _, ok := svc.readModel.FindByEmail("new@example.com"); !ok {
		t.Error("FindByEmail(new@example.com) miss: the normalised address is not indexed")
	}
	if holders := countUsersWithEmail(svc, "before@test.com"); holders != 0 {
		t.Errorf("rows still holding the old address = %d, want 0", holders)
	}
}

// TestService_ChangeEmail_OwnEmailIsNoOp pins that the uniqueness check only
// refuses a DIFFERENT user's address: re-setting the user's own address is the
// decider's idempotent no-op, whatever its spelling.
func TestService_ChangeEmail_OwnEmailIsNoOp(t *testing.T) {
	t.Parallel()

	svc, ctx, _ := newTestServiceWithUser(t, "u1", "own@test.com")
	t.Cleanup(svc.Stop)

	if err := svc.ChangeEmail(ctx, NewUserID("u1"), "OWN@Test.com"); err != nil {
		t.Fatalf("ChangeEmail to the user's own address: %v", err)
	}

	user, err := svc.GetUser(ctx, NewUserID("u1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.Email != "own@test.com" {
		t.Errorf("email = %q, want own@test.com", user.Email)
	}
}

// TestService_ChangeEmail_ConcurrentSameEmail: several distinct users change
// to the same free address at once. The per-address stripe lock spans both the
// uniqueness check and the dispatch, so exactly one of them succeeds, the
// rest answer ErrEmailExists, and exactly one row holds the address.
func TestService_ChangeEmail_ConcurrentSameEmail(t *testing.T) {
	t.Parallel()

	const (
		workers = 8
		target  = "shared@example.com"
	)
	svc := newTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	ids := make([]UserID, workers)
	for i := range workers {
		registerTestUser(t, svc, fmt.Sprintf("racer-%d", i), fmt.Sprintf("racer-%d@test.com", i))
		ids[i] = NewUserID(fmt.Sprintf("racer-%d", i))
	}

	errs := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // barrier: every goroutine races from the same gun
			errs[i] = svc.ChangeEmail(ctx, ids[i], target)
		}()
	}
	close(start)
	wg.Wait()

	var succeeded, refused int
	for i, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrEmailExists):
			refused++
		default:
			t.Errorf("worker %d: unexpected error: %v", i, err)
		}
	}
	if succeeded != 1 {
		t.Errorf("succeeded = %d, want exactly 1", succeeded)
	}
	if refused != workers-1 {
		t.Errorf("refused with ErrEmailExists = %d, want %d", refused, workers-1)
	}
	if holders := countUsersWithEmail(svc, target); holders != 1 {
		t.Errorf("rows holding %s = %d, want exactly 1", target, holders)
	}
	if got := svc.readModel.Count(); got != workers {
		t.Errorf("read model count = %d, want %d (no user may be lost)", got, workers)
	}
}

func TestService_ChangeDisplayName_Success(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "dn@test.com")
	err := svc.ChangeDisplayName(ctx, NewUserID("u1"), "New Display Name")
	if err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	user, _ := svc.GetUser(ctx, NewUserID("u1"))
	if user.DisplayName != "New Display Name" {
		t.Errorf("displayName = %q, want 'New Display Name'", user.DisplayName)
	}
}

func TestService_ChangeDisplayName_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	err := svc.ChangeDisplayName(context.Background(), NewUserID("ghost"), "Ghost")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestService_DeleteUser_Success(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "del@test.com")
	err := svc.DeleteUser(ctx, NewUserID("u1"), "GDPR")
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err = svc.GetUser(ctx, NewUserID("u1"))
	if err == nil {
		t.Fatal("expected error getting deleted user")
	}
}

func TestService_DeleteUser_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	err := svc.DeleteUser(context.Background(), NewUserID("ghost"), "test")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestService_DeleteUser_RevokesSessions(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "u1", "rev@test.com")
	err := svc.DeleteUser(ctx, NewUserID("u1"), "test")
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err = svc.sessions.Find(ctx, reg.Session.Token)
	if err == nil {
		t.Fatal("expected session to be revoked after delete")
	}
}

func TestService_AddCredential_Success(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "addcred@test.com")
	cred := WebAuthnCredential{
		ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
	}
	err := svc.AddCredential(ctx, NewUserID("u1"), cred)
	if err != nil {
		t.Fatalf("AddCredential: %v", err)
	}

	user, _ := svc.GetUser(ctx, NewUserID("u1"))
	if len(user.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(user.Credentials))
	}
}

func TestService_AddCredential_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	err := svc.AddCredential(context.Background(), NewUserID("ghost"), WebAuthnCredential{
		ID: []byte{1},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestService_RemoveCredential_Success(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "remcred@test.com")

	cred := WebAuthnCredential{
		ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
	}
	if err := svc.AddCredential(ctx, NewUserID("u1"), cred); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}

	err := svc.RemoveCredential(ctx, NewUserID("u1"), []byte{1, 2, 3})
	if err != nil {
		t.Fatalf("RemoveCredential: %v", err)
	}

	user, _ := svc.GetUser(ctx, NewUserID("u1"))
	if len(user.Credentials) != 0 {
		t.Fatalf("expected 0 credentials after removal, got %d", len(user.Credentials))
	}
}

func TestService_RemoveCredential_NotFound(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "remcred2@test.com")

	err := svc.RemoveCredential(ctx, NewUserID("u1"), []byte{9, 9, 9})
	if err == nil {
		t.Fatal("expected error for nonexistent credential")
	}
}

func TestService_RemoveCredential_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	err := svc.RemoveCredential(context.Background(), NewUserID("ghost"), []byte{1})
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestService_AddCredential_Duplicate(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "dupcred@test.com")

	cred := WebAuthnCredential{
		ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
	}
	if err := svc.AddCredential(ctx, NewUserID("u1"), cred); err != nil {
		t.Fatalf("first AddCredential: %v", err)
	}

	err := svc.AddCredential(ctx, NewUserID("u1"), cred)
	if err == nil {
		t.Fatal("expected error adding duplicate credential")
	}
}
