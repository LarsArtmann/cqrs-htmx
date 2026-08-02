//nolint:gosec // G202: parameterized placeholder substitution, not user input interpolation
package usermgmt

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// SQLSessionStore implements SessionStore using a SQL database.
// It works with any database/sql-compatible driver (Postgres, SQLite, MySQL).
//
// Call [NewSQLSessionStore] to create and auto-migrate the schema.
//
// Usage:
//
//	db, _ := sql.Open("pgx", "postgres://localhost/users")
//	store, _ := usermgmt.NewSQLSessionStore(ctx, db, "postgres")
//	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
//	    SessionStore: store,
//	    // ... event store, config, etc.
//	})
type SQLSessionStore struct {
	db          *sql.DB
	placeholder placeholderFunc
}

// placeholderFunc returns the dialect-specific SQL placeholder for a given
// 1-based parameter index ($1 for Postgres, ? for SQLite/MySQL).
type placeholderFunc func(i int) string

// placeholderFor maps a dialect string to its placeholder function.
// Used only by SQLSessionStore — the event store delegates to
// go-cqrs-lite/storage/v4's Dialect abstraction.
func placeholderFor(dialect string) (placeholderFunc, error) {
	switch dialect {
	case dialectPostgres, dialectPgx:
		return func(i int) string { return fmt.Sprintf("$%d", i) }, nil
	case dialectSQLite, dialectSQLite3, dialectMySQL:
		return func(i int) string { return "?" }, nil
	default:
		return nil, errorfamily.Newf(event.Rejection, "usermgmt.sql_session.unsupported_dialect",
			"unsupported dialect %q: use postgres, pgx, sqlite, sqlite3, or mysql", dialect).
			WithContext("dialect", dialect)
	}
}

// OptimizeSQLiteDB applies production PRAGMAs to a SQLite *sql.DB:
// WAL mode, synchronous=NORMAL (3-10x faster writes, safe with WAL),
// busy_timeout=5000ms, 64 MB page cache, temp_store=MEMORY, and 256 MB mmap.
//
// Call this BEFORE creating any stores that share the connection:
//
//	db, _ := sql.Open("sqlite", "file:app.db?_pragma=foreign_keys(1)")
//	_ = usermgmt.OptimizeSQLiteDB(ctx, db)
//	sessionStore, _ := usermgmt.NewSQLSessionStore(ctx, db, "sqlite")
//	eventStore, _ := usermgmt.NewSQLEventStore(db, "sqlite")
//
// For Postgres or MySQL this function is a no-op.
func OptimizeSQLiteDB(ctx context.Context, db *sql.DB) error {
	if err := storage.SQLiteEnableWAL(ctx, db); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sqlite.enable_wal", "enable SQLite WAL")
	}
	//cqrs-lint:ignore(C036) library helper: consumer configures both session and event stores with the same backend
	if err := storage.SQLiteApplyOptimizations(ctx, db); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sqlite.apply_optimizations", "apply SQLite optimizations")
	}
	return nil
}

// NewSQLSessionStore creates a SQLSessionStore and auto-migrates the sessions table.
// The dialect must be "postgres", "pgx", "sqlite", "sqlite3", or "mysql".
//
// For SQLite, call [OptimizeSQLiteDB] before this constructor to enable WAL mode
// and production-safe PRAGMAs (3-10x write throughput improvement).
func NewSQLSessionStore(ctx context.Context, db *sql.DB, dialect string) (*SQLSessionStore, error) {
	pf, err := placeholderFor(dialect)
	if err != nil {
		return nil, err
	}
	s := &SQLSessionStore{db: db, placeholder: pf}
	if err := s.migrateSessions(ctx, dialect); err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sql_session.migrate_failed", "migrate sql session store")
	}
	if err := s.migrateOriginColumns(ctx, dialect); err != nil {
		return nil, errorfamily.WrapTransient(
			err,
			"usermgmt.sql_session.migrate_failed",
			"migrate session origin columns",
		)
	}
	return s, nil
}

func (s *SQLSessionStore) migrateSessions(ctx context.Context, dialect string) error {
	var ddl string
	switch dialect {
	case dialectPostgres, dialectPgx:
		ddl = `
		CREATE TABLE IF NOT EXISTS user_sessions (
			token       TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL,
			expires_at  TIMESTAMPTZ NOT NULL,
			origin_type TEXT,
			origin_data TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions (user_id);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions (expires_at);`
	case dialectSQLite, dialectSQLite3:
		ddl = `
		CREATE TABLE IF NOT EXISTS user_sessions (
			token       TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			created_at  DATETIME NOT NULL,
			expires_at  DATETIME NOT NULL,
			origin_type TEXT,
			origin_data TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions (user_id);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions (expires_at);`
	case dialectMySQL:
		ddl = `
		CREATE TABLE IF NOT EXISTS user_sessions (
			token       VARCHAR(255) PRIMARY KEY,
			user_id     VARCHAR(255) NOT NULL,
			created_at  DATETIME(3) NOT NULL,
			expires_at  DATETIME(3) NOT NULL,
			origin_type TEXT,
			origin_data TEXT,
			INDEX idx_user_sessions_user (user_id),
			INDEX idx_user_sessions_expires (expires_at)
		);`
	default:
		return errorfamily.Newf(
			event.Rejection,
			"usermgmt.sql_session.unsupported_dialect",
			"unsupported dialect %q",
			dialect,
		).WithContext("dialect", dialect)
	}
	_, err := s.db.ExecContext(ctx, ddl)
	return wrapTransientOrOK(err, "usermgmt.sql_session.exec_ddl_failed", "exec ddl")
}

// origin_type discriminator values persisted in the user_sessions table, and
// the column names holding the discriminator and its JSON payload.
const (
	originTypeDirect        = "direct"
	originTypeImpersonation = "impersonation"

	originColType = "origin_type"
	originColData = "origin_data"
)

// migrateOriginColumns adds the origin_type/origin_data columns to tables
// created by earlier versions of this store (which lacked them). Fresh tables
// already include the columns via migrateSessions, but CREATE TABLE IF NOT
// EXISTS is a no-op for pre-existing tables, so an explicit ALTER is required.
//
// Only Postgres supports ADD COLUMN IF NOT EXISTS. SQLite and MySQL reject the
// syntax, so for them a plain ADD COLUMN is attempted and the resulting
// duplicate-column error (which both emit as "duplicate column name") is
// tolerated.
func (s *SQLSessionStore) migrateOriginColumns(ctx context.Context, dialect string) error {
	cols := [2]string{originColType, originColData}
	postgres := dialect == dialectPostgres || dialect == dialectPgx
	for _, col := range cols {
		stmt := fmt.Sprintf("ALTER TABLE user_sessions ADD COLUMN %s TEXT", col)
		if postgres {
			stmt = "ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS " + col + " TEXT"
		}
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			if !postgres && isDuplicateColumnErr(err) {
				continue
			}
			return errorfamily.WrapTransient(err, "usermgmt.sql_session.add_origin_column_failed",
				"add column "+col)
		}
	}
	return nil
}

func isDuplicateColumnErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate column")
}

// directLoginRow and impersonationRow are the JSON payloads stored in
// origin_data for the corresponding SessionOrigin variants.
type directLoginRow struct {
	AuthenticatedAs string `json:"authenticated_as"`
}

type impersonationRow struct {
	By     string    `json:"by"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

// marshalSessionOrigin serializes a SessionOrigin into a type discriminator
// and a JSON payload suitable for the origin_type/origin_data columns.
func marshalSessionOrigin(origin SessionOrigin) (originType, data string, err error) {
	switch o := origin.(type) {
	case DirectLogin:
		b, mErr := json.Marshal(directLoginRow{AuthenticatedAs: o.AuthenticatedAs.PrefixedString()})
		if mErr != nil {
			return "", "", errorfamily.Wrapf(mErr, event.Infrastructure,
				"usermgmt.sql_session.marshal_origin_failed", "marshal direct-login origin")
		}
		return originTypeDirect, string(b), nil
	case Impersonation:
		b, mErr := json.Marshal(impersonationRow{
			By:     o.By.PrefixedString(),
			Reason: o.Reason,
			At:     o.At,
		})
		if mErr != nil {
			return "", "", errorfamily.Wrapf(mErr, event.Infrastructure,
				"usermgmt.sql_session.marshal_origin_failed", "marshal impersonation origin")
		}
		return originTypeImpersonation, string(b), nil
	default:
		return "", "", errorfamily.NewRejection("usermgmt.sql_session.unknown_origin_type",
			fmt.Sprintf("unsupported session origin type %T", origin))
	}
}

// unmarshalSessionOrigin reconstructs a SessionOrigin from the stored
// discriminator and JSON payload. An empty originType (NULL or missing column
// on a legacy row) is treated as DirectLogin.
func unmarshalSessionOrigin(originType, data string) (SessionOrigin, error) {
	switch originType {
	case originTypeDirect, "":
		var row directLoginRow
		if data != "" && data != "null" {
			if err := json.Unmarshal([]byte(data), &row); err != nil {
				return nil, errorfamily.Wrapf(err, event.Rejection,
					"usermgmt.sql_session.unmarshal_origin_failed", "unmarshal direct-login origin")
			}
		}
		return DirectLogin{AuthenticatedAs: parseActorIDPrefixed(row.AuthenticatedAs)}, nil
	case originTypeImpersonation:
		var row impersonationRow
		if err := json.Unmarshal([]byte(data), &row); err != nil {
			return nil, errorfamily.Wrapf(err, event.Rejection,
				"usermgmt.sql_session.unmarshal_origin_failed", "unmarshal impersonation origin")
		}
		return Impersonation{
			By:     parseActorIDPrefixed(row.By),
			Reason: row.Reason,
			At:     row.At,
		}, nil
	default:
		return nil, errorfamily.NewRejection("usermgmt.sql_session.unknown_origin_type",
			fmt.Sprintf("unknown session origin type %q", originType))
	}
}

// parseActorIDPrefixed reconstructs an ActorID from its prefixed string form
// ("user:<id>" / "bot:<id>"), the inverse of ActorID.PrefixedString.
func parseActorIDPrefixed(s string) ActorID {
	if s == "" {
		return ActorID{}
	}
	kindStr, raw, found := strings.Cut(s, ":")
	if !found {
		return ActorID{}
	}
	switch kindStr {
	case actorKindUserStr:
		return NewActorID(ActorUser, raw)
	case actorKindBotStr:
		return NewActorID(ActorBot, raw)
	default:
		return ActorID{}
	}
}

// Close closes the underlying database connection.
func (s *SQLSessionStore) Close() error {
	if err := s.db.Close(); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_session.close_failed", "close sql session store db")
	}
	return nil
}

// Create persists a pre-built session, including its SessionOrigin.
func (s *SQLSessionStore) Create(ctx context.Context, session *Session) error {
	originType, originData, err := marshalSessionOrigin(session.Origin)
	if err != nil {
		return err
	}

	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	p3 := s.placeholder(3)
	p4 := s.placeholder(4)
	p5 := s.placeholder(5)
	p6 := s.placeholder(6)

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO user_sessions (token, user_id, created_at, expires_at, origin_type, origin_data)
		 VALUES (`+p1+`, `+p2+`, `+p3+`, `+p4+`, `+p5+`, `+p6+`)`,
		session.Token, session.UserID.Get().String(), session.CreatedAt, session.ExpiresAt, originType, originData,
	)
	if err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_session.insert_failed", "insert session")
	}
	return nil
}

// Find returns the session for the given token, or ErrSessionNotFound.
// Expired sessions are still returned — callers should check [Session.IsExpired].
func (s *SQLSessionStore) Find(ctx context.Context, token string) (*Session, error) {
	p1 := s.placeholder(1)
	var (
		dbToken    string
		dbUserID   string
		createdAt  time.Time
		expiresAt  time.Time
		originType sql.NullString
		originData sql.NullString
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT token, user_id, created_at, expires_at, origin_type, origin_data FROM user_sessions WHERE token = `+p1,
		token,
	).Scan(&dbToken, &dbUserID, &createdAt, &expiresAt, &originType, &originData)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sql_session.find_failed", "find session").
			WithContext("token", token)
	}

	origin, err := unmarshalSessionOrigin(originType.String, originData.String)
	if err != nil {
		return nil, err
	}

	return &Session{
		Token:     dbToken,
		UserID:    NewUserID(dbUserID),
		ActorID:   ActorIDFromUser(NewUserID(dbUserID)),
		Origin:    origin,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

// Delete removes the session with the given token. It is idempotent — deleting a
// non-existent token is not an error.
func (s *SQLSessionStore) Delete(ctx context.Context, token string) error {
	p1 := s.placeholder(1)
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM user_sessions WHERE token = `+p1,
		token,
	)
	return wrapTransientOrOK(err, "usermgmt.sql_session.delete_failed", "delete session")
}

// DeleteByUserID removes all sessions belonging to the given user.
func (s *SQLSessionStore) DeleteByUserID(ctx context.Context, userID UserID) error {
	p1 := s.placeholder(1)
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM user_sessions WHERE user_id = `+p1,
		userID.Get().String(),
	)
	return wrapTransientOrOK(err, "usermgmt.sql_session.delete_by_user_failed", "delete sessions by user")
}

// EvictExpired removes all expired sessions and returns the count evicted.
// Call periodically to keep the table small.
func (s *SQLSessionStore) EvictExpired(ctx context.Context) (int64, error) {
	p1 := s.placeholder(1)
	res, err := s.db.ExecContext(
		ctx,
		`DELETE FROM user_sessions WHERE expires_at < `+p1,
		time.Now().UTC(),
	)
	if err != nil {
		return 0, errorfamily.WrapTransient(err, "usermgmt.sql_session.evict_failed", "evict expired sessions")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, errorfamily.WrapTransient(err, "usermgmt.sql_session.rows_affected_failed", "rows affected")
	}
	return n, nil
}

// StartCleanupSweeper launches a background goroutine that periodically calls
// EvictExpired. Call Close to stop it. The returned function can be used for
// early shutdown (e.g., defer in tests or context-cancellation in main).
func (s *SQLSessionStore) StartCleanupSweeper(ctx context.Context, interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				_, _ = s.EvictExpired(ctx)
			}
		}
	}()
	return func() { close(done) }
}
