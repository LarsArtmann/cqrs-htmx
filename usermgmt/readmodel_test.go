package usermgmt

import (
	"bytes"
	"context"
	"encoding/csv"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestUserReadModel_AllUsers(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	reg1 := registerTestUser(t, svc, "rm1", "rm1@test.com")
	reg2 := registerTestUser(t, svc, "rm2", "rm2@test.com")

	all := svc.readModel.AllUsers()
	if len(all) != 2 {
		t.Fatalf("expected 2 users, got %d", len(all))
	}

	ids := make([]string, len(all))
	for i, u := range all {
		ids[i] = u.ID.Get().String()
	}
	if !slices.Contains(ids, reg1.User.ID.Get().String()) || !slices.Contains(ids, reg2.User.ID.Get().String()) {
		t.Errorf("expected both user IDs in result: %v", ids)
	}

	// Mutating returned users must not affect the read model.
	all[0].Email = "mutated@test.com"
	aggID, err := id.ParseAggregateID(reg1.User.ID.Get().String())
	if err != nil {
		t.Fatalf("parse aggregate id: %v", err)
	}
	u, ok := svc.readModel.FindByID(aggID)
	if !ok {
		t.Fatal("user not found")
	}
	if u.Email == "mutated@test.com" {
		t.Error("AllUsers() returned a mutable reference to the internal user")
	}
}

func TestUserReadModel_AllUsersSorted(t *testing.T) {
	m := NewUserReadModel()
	now := nowUTC()
	m.users[id.NewAggregateID()] = &User{ID: NewUserID("z"), CreatedAt: now}
	m.users[id.NewAggregateID()] = &User{ID: NewUserID("a"), CreatedAt: now.Add(-1)}

	all := m.AllUsers()
	if len(all) != 2 {
		t.Fatalf("expected 2 users, got %d", len(all))
	}
	if all[0].ID.Get().String() != NewUserID("a").Get().String() ||
		all[1].ID.Get().String() != NewUserID("z").Get().String() {
		t.Errorf("expected sorted order a,z, got %s,%s", all[0].ID.Get().String(), all[1].ID.Get().String())
	}
}

func TestWebAuthnCredential_Clone(t *testing.T) {
	orig := WebAuthnCredential{
		CredentialCore: CredentialCore{
			ID:         []byte{1, 2, 3},
			PublicKey:  []byte{4, 5, 6},
			Transports: []string{"usb", "nfc"},
			AAGUID:     []byte{7, 8, 9},
		},
	}
	cp := orig.Clone()

	cp.ID[0] = 99
	cp.PublicKey[0] = 99
	cp.Transports[0] = "ble"
	cp.AAGUID[0] = 99

	if orig.ID[0] == 99 || orig.PublicKey[0] == 99 || orig.AAGUID[0] == 99 {
		t.Error("Clone() shares byte slices with original")
	}
	if orig.Transports[0] == "ble" {
		t.Error("Clone() shares transports slice with original")
	}
}

func TestUser_Clone_DeepCopiesCredentials(t *testing.T) {
	u := &User{
		ID: NewUserID("u1"),
		Credentials: []WebAuthnCredential{{
			CredentialCore: CredentialCore{
				ID:        []byte{1},
				PublicKey: []byte{2},
			},
		}},
	}
	cp := u.Clone()
	cp.Credentials[0].ID[0] = 99
	cp.Credentials[0].PublicKey[0] = 99

	if u.Credentials[0].ID[0] == 99 || u.Credentials[0].PublicKey[0] == 99 {
		t.Error("User.Clone() shares credential inner slices")
	}
}

func TestService_ExportUsersToCSV_IncludesTOTPEnabled(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()
	registerTestUser(t, svc, "csvtotp", "csvtotp@test.com")

	var buf bytes.Buffer
	if err := svc.ExportUsersToCSV(ctx, &buf); err != nil {
		t.Fatalf("ExportUsersToCSV: %v", err)
	}
	rows, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	if len(rows) < 1 {
		t.Fatal("expected header row")
	}
	want := []string{
		csvColumnID,
		csvColumnEmail,
		csvColumnDisplayName,
		csvColumnEmailVerified,
		csvColumnTOTPEnabled,
	}
	if !slices.Equal(rows[0], want) {
		t.Errorf("header = %v, want %v", rows[0], want)
	}
}
