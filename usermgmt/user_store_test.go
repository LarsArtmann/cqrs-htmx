package usermgmt

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryUserStore(t *testing.T) {
	store := NewInMemoryUserStore()

	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	_ = u.SetPasswordWithCost("pass", minBcryptCost)

	if err := store.Create(context.Background(), u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.FindByID(context.Background(), NewUserID("user-1"))
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", found.Email)
	}

	byEmail, err := store.FindByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if byEmail.ID != NewUserID("user-1") {
		t.Errorf("expected ID user-1, got %s", byEmail.ID)
	}

	_, err = store.FindByID(context.Background(), NewUserID("nonexistent"))
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")

	if err := store.Delete(context.Background(), NewUserID("user-1")); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.FindByID(context.Background(), NewUserID("user-1"))
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")
}

func TestInMemoryUserStore_CreateDuplicate(t *testing.T) {
	store := NewInMemoryUserStore()
	u1 := NewUser(NewUserID("u1"), "dup@test.com", "One")
	_ = store.Create(context.Background(), u1)

	u2 := NewUser(NewUserID("u2"), "dup@test.com", "Two")
	if err := store.Create(context.Background(), u2); !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestInMemoryUserStore_SaveUpdatesEmailIndex(t *testing.T) {
	store := NewInMemoryUserStore()
	ctx := context.Background()
	u := NewUser(NewUserID("u1"), "old@test.com", "Test")
	_ = store.Create(ctx, u)

	updated := NewUser(NewUserID("u1"), "new@test.com", "Test")
	_ = store.Save(ctx, updated)

	_, err := store.FindByEmail(ctx, "old@test.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Error("expected old email to be gone from index")
	}
	found, err := store.FindByEmail(ctx, "new@test.com")
	if err != nil {
		t.Fatalf("FindByEmail new: %v", err)
	}
	if found.ID != NewUserID("u1") {
		t.Errorf("expected u1, got %s", found.ID)
	}
}

func TestInMemoryUserStore_CreateDuplicateID(t *testing.T) {
	store := NewInMemoryUserStore()
	u := NewUser(NewUserID("dup-id"), "a@b.com", "Test")
	_ = store.Create(context.Background(), u)

	u2 := NewUser(NewUserID("dup-id"), "other@b.com", "Other")
	if err := store.Create(context.Background(), u2); !errors.Is(err, ErrUserIDExists) {
		t.Errorf("expected ErrUserIDExists, got %v", err)
	}
}
