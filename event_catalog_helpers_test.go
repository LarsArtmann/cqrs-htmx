package cqrshtmx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type fakeSerializer struct {
	data []byte
	err  error
}

func (f *fakeSerializer) JSON() ([]byte, error) {
	return f.data, f.err
}

func TestSerializeToImmutableHandler_Success(t *testing.T) {
	want := []byte(`{"hello":"world"}`)

	handler, err := serializeToImmutableHandler(
		&fakeSerializer{data: want},
		"code", "msg",
	)
	if err != nil {
		t.Fatalf("serializeToImmutableHandler: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != string(want) {
		t.Fatalf("expected body %s, got %s", want, w.Body.Bytes())
	}

	if w.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header")
	}
}

func TestSerializeToImmutableHandler_Error(t *testing.T) {
	wantErr := errors.New("serialize failed")

	_, err := serializeToImmutableHandler(
		&fakeSerializer{err: wantErr},
		"code", "msg",
	)
	if err == nil {
		t.Fatal("expected error")
	}

	if errorfamily.Classify(err) != event.Infrastructure {
		t.Fatalf("expected Infrastructure error, got %v", err)
	}

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
