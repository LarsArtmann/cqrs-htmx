//nolint:gosec // G202: parameterized placeholder substitution, not user input interpolation
package usermgmt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
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
// go-cqrs-lite/storage/v2's Dialect abstraction.
func placeholderFor(dialect string) (placeholderFunc, error) {
	switch dialect {
	case dialectPostgres, dialectPgx:
		return func(i int) string { return fmt.Sprintf("$%d", i) }, nil
	case dialectSQLite, dialectSQLite3, dialectMySQL:
		return func(i int) string { return "?" }, nil
	default:
		return nil, fmt.Errorf("unsupported dialect %q: use postgres, pgx, sqlite, sqlite3, or mysql", dialect)
	}
}

// NewSQLSessionStore creates a SQLSessionStore and auto-migrates the sessions table.
// The dialect must be "postgres", "pgx", "sqlite", "sqlite3", or "mysql".
func NewSQLSessionStore(ctx context.Context, db *sql.DB, dialect string) (*SQLSessionStore, error) {
	pf, err := placeholderFor(dialect)
	if err != nil {
		return nil, err
	}
	s := &SQLSessionStore{db: db, placeholder: pf}
	if err := s.migrateSessions(ctx, dialect); err != nil {
		return nil, fmt.Errorf("migrate sql session store: %w", err)
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
			expires_at  TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions (user_id);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions (expires_at);`
	case dialectSQLite, dialectSQLite3:
		ddl = `
		CREATE TABLE IF NOT EXISTS user_sessions (
			token       TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			created_at  DATETIME NOT NULL,
			expires_at  DATETIME NOT NULL
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
			INDEX idx_user_sessions_user (user_id),
			INDEX idx_user_sessions_expires (expires_at)
		);`
	default:
		return fmt.Errorf("unsupported dialect %q", dialect)
	}
	_, err := s.db.ExecContext(ctx, ddl)
	if err != nil {
		return fmt.Errorf("exec ddl: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *SQLSessionStore) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close sql session store db: %w", err)
	}
	return nil
}

// Create generates a new session for the user with the given TTL and persists it.
func (s *SQLSessionStore) Create(
	ctx context.Context, userID UserID, ttl time.Duration,
) (*Session, error) {
	session, err := NewSession(userID, ttl)
	if err != nil {
		return nil, event.NewTransient("session_create_failed",
			fmt.Sprintf("create session for user %q", userID)).WithCause(err)
	}

	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	p3 := s.placeholder(3)
	p4 := s.placeholder(4)

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO user_sessions (token, user_id, created_at, expires_at)
		 VALUES (`+p1+`, `+p2+`, `+p3+`, `+p4+`)`,
		session.Token, userID.Get(), session.CreatedAt, session.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}
	return session, nil
}

// Find returns the session for the given token, or ErrSessionNotFound.
// Expired sessions are still returned — callers should check [Session.IsExpired].
func (s *SQLSessionStore) Find(ctx context.Context, token string) (*Session, error) {
	p1 := s.placeholder(1)
	var (
		dbToken   string
		dbUserID  string
		createdAt time.Time
		expiresAt time.Time
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT token, user_id, created_at, expires_at FROM user_sessions WHERE token = `+p1,
		token,
	).Scan(&dbToken, &dbUserID, &createdAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}

	return &Session{
		Token:     dbToken,
		UserID:    NewUserID(dbUserID),
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
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteByUserID removes all sessions belonging to the given user.
func (s *SQLSessionStore) DeleteByUserID(ctx context.Context, userID UserID) error {
	p1 := s.placeholder(1)
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM user_sessions WHERE user_id = `+p1,
		userID.Get(),
	)
	if err != nil {
		return fmt.Errorf("delete sessions by user: %w", err)
	}
	return nil
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
		return 0, fmt.Errorf("evict expired sessions: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
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
