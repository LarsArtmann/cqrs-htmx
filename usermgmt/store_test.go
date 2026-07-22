package usermgmt_test

import (
	"context"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt"
)

type testWidget struct {
	ID    int
	Name  string
	Count int
}

func TestInMemoryStore_CreateSaveFindDelete(t *testing.T) {
	ctx := context.Background()
	store := usermgmt.NewInMemoryStore(func(w testWidget) int { return w.ID })

	w1 := testWidget{ID: 1, Name: "one", Count: 10}
	if err := store.Create(ctx, w1); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Create(ctx, w1); err == nil {
		t.Fatal("Create duplicate should error")
	}

	w1.Count = 20
	if err := store.Save(ctx, w1); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.FindByID(ctx, 1)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Count != 20 {
		t.Fatalf("FindByID Count = %d, want 20", got.Count)
	}

	if _, err := store.FindByID(ctx, 2); err != nil {
		t.Fatalf("FindByID missing: %v", err)
	}

	if err := store.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	after, _ := store.FindByID(ctx, 1)
	if after.ID != 0 {
		t.Fatalf("FindByID after Delete = %+v, want zero", after)
	}
}

func TestStore_InterfaceUsage(t *testing.T) {
	ctx := context.Background()
	impl := usermgmt.NewInMemoryStore(func(w testWidget) int { return w.ID })
	var store usermgmt.Store[testWidget, int] = impl

	if err := store.Create(ctx, testWidget{ID: 42, Name: "x"}); err != nil {
		t.Fatalf("Create via interface: %v", err)
	}
}
