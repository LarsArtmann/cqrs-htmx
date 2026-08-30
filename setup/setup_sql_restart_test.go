package setup_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	_ "modernc.org/sqlite"
)

// TestBundle_SQLPersistence_SurvivesRestart is the BUNDLE-level persistence
// contract for the documented production config (ServiceConfig with
// CheckpointStore + ReadModelDB + sqlite dialect): everything the deployment
// cares about — registered users and live sessions — survives a full
// process restart when the state lives in SQL.
//
// User survival exercises the read-model HYDRATE path end to end (a fresh
// bundle on the same database must serve the existing user without a
// replay — the checkpoint says the journal is fully applied). Session
// survival exercises the consumer-supplied SessionStore hatch: two bundles
// share one store instance exactly the way two replicas share a SQL session
// store, so a session created by instance #1 authenticates on instance #2.
func TestBundle_SQLPersistence_SurvivesRestart(t *testing.T) {
	ctx := t.Context()
	dbFile := filepath.Join(t.TempDir(), "bundle-readmodels.sqlite")

	// Shared across "instances": stands in for the SQL session store a
	// multi-instance deployment would supply (in-memory per-process stores
	// would NOT survive, which is precisely why the hatch exists).
	sessions := usermgmt.NewInMemorySessionStore()

	openDB := func(t *testing.T) *sql.DB {
		t.Helper()
		db, err := sql.Open("sqlite", "file:"+dbFile)
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		db.SetMaxOpenConns(1)
		// The checkpoint store does not self-migrate; apply its schema the
		// way a deployment init script would.
		if _, err := db.Exec(storage.SQLiteCheckpointSchema()); err != nil {
			t.Fatalf("migrate checkpoints table: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		return db
	}

	newBundle := func(t *testing.T) (*setup.Bundle, *httptest.Server) {
		t.Helper()
		db := openDB(t)
		cp, err := storage.NewSQLiteCheckpointStore(db)
		if err != nil {
			t.Fatalf("NewSQLiteCheckpointStore: %v", err)
		}
		bundle, err := setup.New(setup.Config{
			Title: "Restart contract",
			ServiceConfig: &usermgmt.ServiceConfig{
				CheckpointStore:  cp,
				ReadModelDB:      db,
				ReadModelDialect: "sqlite",
				SessionStore:     sessions,
			},
		})
		if err != nil {
			t.Fatalf("setup.New: %v", err)
		}
		t.Cleanup(func() { _ = bundle.Close() })

		mux := http.NewServeMux()
		bundle.Mount(mux)
		// Session middleware FIRST (enriches the request with the user),
		// then the security chain — the order consumers are told to use.
		server := httptest.NewServer(bundle.Middleware()(bundle.SessionMiddleware()(mux)))
		t.Cleanup(server.Close)
		return bundle, server
	}

	registerStatus := func(t *testing.T, server *httptest.Server, email string) (int, []byte, []*http.Cookie) {
		t.Helper()
		body := fmt.Sprintf(`{"email":%q,"display_name":"Restart Contract"}`, email)
		resp, err := http.Post(server.URL+"/auth/register", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST /auth/register: %v", err)
		}
		defer resp.Body.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return resp.StatusCode, buf.Bytes(), resp.Cookies()
	}
	register := func(t *testing.T, server *httptest.Server, email string) int {
		t.Helper()
		code, _, _ := registerStatus(t, server, email)
		return code
	}
	registerCapture := func(t *testing.T, server *httptest.Server, email string) ([]byte, []*http.Cookie) {
		t.Helper()
		code, body, cookies := registerStatus(t, server, email)
		if code != http.StatusCreated {
			t.Fatalf("register on instance #1: status %d, want 201", code)
		}
		return body, cookies
	}

	// --- instance #1: register a user through the full HTTP surface --------
	_, server1 := newBundle(t)
	regBody, regCookies := registerCapture(t, server1, "restart@example.com")
	var sessionCookie *http.Cookie
	for _, c := range regCookies {
		if c.Value != "" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("register response set no session cookie")
	}
	t.Logf(
		"register cookie: name=%q value-prefix=%q",
		sessionCookie.Name,
		sessionCookie.Value[:min(8, len(sessionCookie.Value))],
	)

	var reg struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
		Session struct {
			Token string `json:"token"`
		} `json:"session"`
	}
	if err := json.Unmarshal(regBody, &reg); err != nil || reg.User.ID == "" || reg.Session.Token == "" {
		t.Fatalf("register response must carry user.id and session.token (body=%s err=%v)", regBody, err)
	}

	// CONTROL: the same token must authenticate on instance #1 first.
	ctrl, err := http.NewRequestWithContext(ctx, http.MethodGet, server1.URL+"/auth/me", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("new control request: %v", err)
	}
	ctrl.AddCookie(sessionCookie)
	ctrlResp, err := http.DefaultClient.Do(ctrl)
	if err != nil {
		t.Fatalf("GET /auth/me on instance #1: %v", err)
	}
	ctrlBody, _ := io.ReadAll(ctrlResp.Body)
	_ = ctrlResp.Body.Close()
	if ctrlResp.StatusCode != http.StatusOK {
		t.Fatalf("control: /auth/me on instance #1 failed: status %d body=%s", ctrlResp.StatusCode, ctrlBody)
	}

	// --- instance #2: same database, fresh bundle --------------------------
	_, server2 := newBundle(t)

	// Session survival: the session issued by instance #1 (now only in the
	// shared, SQL-in-production session store) must authenticate on
	// instance #2.
	me, err := http.NewRequestWithContext(ctx, http.MethodGet, server2.URL+"/auth/me", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	me.AddCookie(sessionCookie)
	meResp, err := http.DefaultClient.Do(me)
	if err != nil {
		t.Fatalf("GET /auth/me on instance #2: %v", err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("session issued on #1 must authenticate on #2 via the supplied store: status %d", meResp.StatusCode)
	}
	var meBody struct {
		Email string `json:"email"`
	}
	_ = json.NewDecoder(meResp.Body).Decode(&meBody)
	if meBody.Email != "restart@example.com" {
		t.Fatalf("instance #2 resolved the wrong user: %q", meBody.Email)
	}

	// User survival: re-registering the SAME email must conflict — the read
	// model was hydrated from SQL, not replayed from an empty journal.
	if code := register(t, server2, "restart@example.com"); code != http.StatusConflict {
		t.Fatalf("duplicate register on instance #2: status %d, want 409 (user survived via SQL)", code)
	}

	// The bundle itself must be healthy: hydrated projections report ready.
	health, err := http.Get(server2.URL + "/health")
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	defer health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("instance #2 health: status %d, want 200", health.StatusCode)
	}
}
