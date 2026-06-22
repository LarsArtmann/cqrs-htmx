package usermgmt

import (
	"context"
	"testing"
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
		credentialCore: credentialCore{
			ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
		},
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
		credentialCore: credentialCore{
			ID: []byte{1},
		},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestService_RemoveCredential_Success(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "remcred@test.com")

	cred := WebAuthnCredential{
		credentialCore: credentialCore{
			ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
		},
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
		credentialCore: credentialCore{
			ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
		},
	}
	if err := svc.AddCredential(ctx, NewUserID("u1"), cred); err != nil {
		t.Fatalf("first AddCredential: %v", err)
	}

	err := svc.AddCredential(ctx, NewUserID("u1"), cred)
	if err == nil {
		t.Fatal("expected error adding duplicate credential")
	}
}
