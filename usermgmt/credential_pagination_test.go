package usermgmt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListCredentials_Pagination(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	registerTestUser(t, svc, "pg1", "pg1@test.com")

	for i := range 5 {
		cred := WebAuthnCredential{
			ID:   []byte{byte(i + 1)},
			Name: "key-" + string(rune('A'+i)),
		}
		if err := svc.AddCredential(context.Background(), NewUserID("pg1"), cred); err != nil {
			t.Fatalf("AddCredential %d: %v", i, err)
		}
	}

	user, err := svc.GetUser(context.Background(), NewUserID("pg1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
		wantCount    int
		wantTotal    int
		wantPages    int
	}{
		{"default", "", 1, 20, 5, 5, 1},
		{"page1_size2", "?page=1&page_size=2", 1, 2, 2, 5, 3},
		{"page2_size2", "?page=2&page_size=2", 2, 2, 2, 5, 3},
		{"page3_size2", "?page=3&page_size=2", 3, 2, 1, 5, 3},
		{"over_max", "?page_size=999", 1, 100, 5, 5, 1},
		{"beyond_last", "?page=99", 1, 20, 5, 5, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/auth/credentials"+tc.query, nil)
			req = req.WithContext(WithUser(context.Background(), user))
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}

			var result credentialListResponse
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(result.Credentials) != tc.wantCount {
				t.Errorf("credentials count = %d, want %d", len(result.Credentials), tc.wantCount)
			}
			if result.Page != tc.wantPage {
				t.Errorf("page = %d, want %d", result.Page, tc.wantPage)
			}
			if result.PageSize != tc.wantPageSize {
				t.Errorf("page_size = %d, want %d", result.PageSize, tc.wantPageSize)
			}
			if result.TotalCount != tc.wantTotal {
				t.Errorf("total_count = %d, want %d", result.TotalCount, tc.wantTotal)
			}
			if result.TotalPages != tc.wantPages {
				t.Errorf("total_pages = %d, want %d", result.TotalPages, tc.wantPages)
			}
		})
	}
}

func TestHandleListCredentials_EmptyList(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	registerTestUser(t, svc, "empty1", "empty1@test.com")

	user, err := svc.GetUser(context.Background(), NewUserID("empty1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/credentials", nil)
	req = req.WithContext(WithUser(context.Background(), user))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result credentialListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Credentials) != 0 {
		t.Errorf("expected 0 credentials, got %d", len(result.Credentials))
	}
}
