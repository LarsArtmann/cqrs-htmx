package setup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// The dashboard's two audit panels (/commands, /queries) mount only when the
// bundle is handed stores that can list their whole trail (Config.CommandStore
// and Config.QueryStore). A default bundle mounts neither.

// commandStoreWithoutJournal hides the memory store's ReadAll, so it is a
// command.Store that cannot list its whole trail.
type commandStoreWithoutJournal struct {
	command.Store
}

// queryStoreWithoutJournal is commandStoreWithoutJournal's query-side twin.
type queryStoreWithoutJournal struct {
	query.QueryStore
}

// dashboardAuditStatus serves one dashboard audit panel as an authenticated
// user and returns the response status.
func dashboardAuditStatus(t *testing.T, bundle *setup.Bundle, user, path string) int {
	t.Helper()

	reg, err := bundle.Service.Register(context.Background(), usermgmt.RegisterRequest{
		ID: usermgmt.SyntheticUserID(user), Email: user + "@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard"+path, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: reg.Session.Token})

	rec := httptest.NewRecorder()
	bundle.Handler(http.NewServeMux()).ServeHTTP(rec, req)

	return rec.Code
}

func TestNew_AuditStores_MountTheDashboardAuditPanels(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:        "Audit Panels",
		CommandStore: memorystorage.NewMemoryCommandStore(),
		QueryStore:   memorystorage.NewMemoryQueryStore(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	for path, user := range map[string]string{
		"/commands": "auditcommands",
		"/queries":  "auditqueries",
	} {
		if status := dashboardAuditStatus(t, bundle, user, path); status != http.StatusOK {
			t.Errorf("dashboard %s status = %d, want 200 once its store is wired", path, status)
		}
	}
}

func TestNew_WithoutAuditStores_MountsNoAuditPanel(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{Title: "No Audit Panels"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	for path, user := range map[string]string{
		"/commands": "noauditcommands",
		"/queries":  "noauditqueries",
	} {
		if status := dashboardAuditStatus(t, bundle, user, path); status != http.StatusNotFound {
			t.Errorf("dashboard %s status = %d, want 404 without a store", path, status)
		}
	}
}

func TestNew_AuditStoreThatCannotListItsTrail_IsRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config setup.Config
		want   string
	}{
		{
			name: "command store without ReadAll",
			config: setup.Config{
				CommandStore: commandStoreWithoutJournal{memorystorage.NewMemoryCommandStore()},
			},
			want: "command.CommandJournal",
		},
		{
			name: "query store without ReadAllQueries",
			config: setup.Config{
				QueryStore: queryStoreWithoutJournal{memorystorage.NewMemoryQueryStore()},
			},
			want: "query.QueryJournal",
		},
		{
			name:   "query store behind a disabled dashboard",
			config: setup.Config{QueryStore: memorystorage.NewMemoryQueryStore(), DisableDashboard: true},
			want:   "DisableDashboard",
		},
		{
			name: "both stores behind a disabled dashboard",
			config: setup.Config{
				CommandStore:     memorystorage.NewMemoryCommandStore(),
				QueryStore:       memorystorage.NewMemoryQueryStore(),
				DisableDashboard: true,
			},
			want: "DisableDashboard",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := setup.New(tc.config)
			if err == nil {
				t.Fatalf("New succeeded; want a rejection naming %q", tc.want)
			}

			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("New error = %q, want it to name %q", err, tc.want)
			}
		})
	}
}

func TestNewShell_AuditStores_AreRejected(t *testing.T) {
	t.Parallel()

	_, err := setup.NewShell(setup.Config{
		QueryStore:       memorystorage.NewMemoryQueryStore(),
		DisableAdmin:     true,
		DisableDashboard: true,
		DisableLogin:     true,
	})
	if err == nil {
		t.Fatal("NewShell succeeded with a QueryStore; want a rejection naming CommandStore and QueryStore")
	}

	if !strings.Contains(err.Error(), "CommandStore and QueryStore") {
		t.Errorf("NewShell error = %q, want it to name CommandStore and QueryStore", err)
	}
}
