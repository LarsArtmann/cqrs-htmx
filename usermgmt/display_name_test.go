package usermgmt

import (
	"context"
	"testing"
)

func TestDisplayName_LadderOrder(t *testing.T) {
	svc := newTestService(t)

	withName := registerTestUser(t, svc, "dn1", "dn1@test.com")
	if got := svc.DisplayName(context.Background(), withName.User.ID.Get().String()); got != "dn1@test.com" {
		// registerTestUser sets no display name; email is the fallback rung.
		t.Errorf("DisplayName(no display name) = %q, want email %q", got, "dn1@test.com")
	}

	if err := svc.ChangeDisplayName(context.Background(), withName.User.ID, "Ada Lovelace"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}
	if got := svc.DisplayName(context.Background(), withName.User.ID.Get().String()); got != "Ada Lovelace" {
		t.Errorf("DisplayName(display name set) = %q, want %q", got, "Ada Lovelace")
	}
}

func TestDisplayName_AcceptsBothIdShapes(t *testing.T) {
	svc := newTestService(t)
	reg := registerTestUser(t, svc, "dn2", "dn2@test.com")
	_ = svc.ChangeDisplayName(
		context.Background(),
		reg.User.ID,
		"Grace Hopper",
	) //nolint:errcheck // ladder asserted below

	bare := reg.User.ID.Get().String()
	if got := svc.DisplayName(context.Background(), bare); got != "Grace Hopper" {
		t.Errorf("DisplayName(bare) = %q, want %q", got, "Grace Hopper")
	}
	if got := svc.DisplayName(context.Background(), "user:"+bare); got != "Grace Hopper" {
		t.Errorf("DisplayName(user-prefixed) = %q, want %q", got, "Grace Hopper")
	}
}

func TestDisplayName_MissingAndGarbageResolveEmpty(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	for _, in := range []string{
		"",
		"ghost",
		"user:ghost",
		"bot:01JXTENANT0000000000000000B", // wrong actor kind: no bare ULID under the prefix
	} {
		if got := svc.DisplayName(ctx, in); got != "" {
			t.Errorf("DisplayName(%q) = %q, want empty", in, got)
		}
	}
}

func TestDisplayName_TombstonedUserResolvesEmpty(t *testing.T) {
	svc := newTestService(t)
	reg := registerTestUser(t, svc, "dn3", "dn3@test.com")

	if err := svc.DeleteUser(context.Background(), reg.User.ID, "display-name test"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if got := svc.DisplayName(context.Background(), reg.User.ID.Get().String()); got != "" {
		t.Errorf("DisplayName(removed user) = %q, want empty", got)
	}
}
