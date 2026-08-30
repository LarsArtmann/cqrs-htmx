package usermgmt

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type viewRecord struct {
	ID   string `view:"id"`
	Name string `view:"name"`
}

type stringKey string

func (s stringKey) String() string { return string(s) }

func TestNewViewStoreOrFail_Success(t *testing.T) {
	var called bool
	mapper := storage.AutoMapper[viewRecord]( //nolint:staticcheck // ADR-0123 v5
		"test",
	)
	create := func(_ *sql.DB, _ storage.ViewMapper[viewRecord], _ ...storage.ViewStoreOption) (*storage.SQLViewStore[viewRecord, stringKey], error) { //nolint:staticcheck // ADR-0123 v5
		called = true
		return &storage.SQLViewStore[viewRecord, stringKey]{}, nil //nolint:staticcheck // ADR-0123 v5
	}
	store, err := newViewStoreOrFail[viewRecord, stringKey](create, nil, mapper, "code", "msg")
	if err != nil {
		t.Fatalf("newViewStoreOrFail: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if !called {
		t.Fatal("expected create callback to be called")
	}
}

func TestNewViewStoreOrFail_Error(t *testing.T) {
	wantErr := errors.New("create failed")
	mapper := storage.AutoMapper[viewRecord]( //nolint:staticcheck // ADR-0123 v5
		"test",
	)
	create := func(_ *sql.DB, _ storage.ViewMapper[viewRecord], _ ...storage.ViewStoreOption) (*storage.SQLViewStore[viewRecord, stringKey], error) { //nolint:staticcheck // ADR-0123 v5
		return nil, wantErr
	}
	_, err := newViewStoreOrFail[viewRecord, stringKey](create, nil, mapper, "code", "msg")
	if err == nil {
		t.Fatal("expected error")
	}
	if errorfamily.Classify(err) != event.Transient {
		t.Fatalf("expected Transient error, got %v", err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestWrapTransientOrOK_Nil(t *testing.T) {
	if err := wrapTransientOrOK(nil, "code", "msg"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWrapTransientOrOK_Error(t *testing.T) {
	wantErr := errors.New("db failed")
	err := wrapTransientOrOK(wantErr, "code", "msg")
	if err == nil {
		t.Fatal("expected error")
	}
	if errorfamily.Classify(err) != event.Transient {
		t.Fatalf("expected Transient error, got %v", err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
