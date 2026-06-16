//nolint:gosec // G202: parameterized placeholder substitution, not user input interpolation
package usermgmt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// SQLEventStore implements event.Store and event.Journal using a SQL database.
// It works with any database/sql-compatible driver (Postgres, SQLite, MySQL).
//
// Call [NewSQLEventStore] to create and auto-migrate the schema.
//
// Usage:
//
//	db, _ := sql.Open("pgx", "postgres://localhost/users")
//	store, _ := usermgmt.NewSQLEventStore(db)
//	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
//	    EventStore: store,
//	})
type SQLEventStore struct {
	db          *sql.DB
	placeholder placeholderFunc
}

type placeholderFunc func(i int) string

// NewSQLEventStore creates a SQLEventStore and auto-migrates the events table.
// The dialect must be "postgres", "sqlite", or "mysql".
func NewSQLEventStore(ctx context.Context, db *sql.DB, dialect string) (*SQLEventStore, error) {
	pf, err := placeholderFor(dialect)
	if err != nil {
		return nil, err
	}
	s := &SQLEventStore{db: db, placeholder: pf}
	if err := s.migrate(ctx, dialect); err != nil {
		return nil, fmt.Errorf("migrate sql event store: %w", err)
	}
	return s, nil
}

func placeholderFor(dialect string) (placeholderFunc, error) {
	switch dialect {
	case "postgres", "pgx":
		return func(i int) string { return fmt.Sprintf("$%d", i) }, nil
	case "sqlite", "sqlite3":
		return func(i int) string { return "?" }, nil
	case "mysql":
		return func(i int) string { return "?" }, nil
	default:
		return nil, fmt.Errorf("unsupported dialect %q: use postgres, sqlite, or mysql", dialect)
	}
}

func (s *SQLEventStore) migrate(ctx context.Context, dialect string) error {
	var ddl string
	switch dialect {
	case "postgres", "pgx":
		ddl = `
		CREATE TABLE IF NOT EXISTS user_events (
			event_id TEXT PRIMARY KEY,
			event_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			aggregate_type TEXT NOT NULL,
			version INTEGER NOT NULL,
			payload BYTEA NOT NULL,
			metadata JSONB,
			occurred_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_user_events_agg ON user_events (aggregate_id, version);
		CREATE INDEX IF NOT EXISTS idx_user_events_time ON user_events (occurred_at);`
	case "sqlite", "sqlite3":
		ddl = `
		CREATE TABLE IF NOT EXISTS user_events (
			event_id TEXT PRIMARY KEY,
			event_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			aggregate_type TEXT NOT NULL,
			version INTEGER NOT NULL,
			payload BLOB NOT NULL,
			metadata TEXT,
			occurred_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_user_events_agg ON user_events (aggregate_id, version);
		CREATE INDEX IF NOT EXISTS idx_user_events_time ON user_events (occurred_at);`
	case "mysql":
		ddl = `
		CREATE TABLE IF NOT EXISTS user_events (
			event_id VARCHAR(255) PRIMARY KEY,
			event_type VARCHAR(255) NOT NULL,
			aggregate_id VARCHAR(255) NOT NULL,
			aggregate_type VARCHAR(255) NOT NULL,
			version INT NOT NULL,
			payload BLOB NOT NULL,
			metadata JSON,
			occurred_at DATETIME(3) NOT NULL,
			INDEX idx_user_events_agg (aggregate_id, version),
			INDEX idx_user_events_time (occurred_at)
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

func (s *SQLEventStore) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close sql event store db: %w", err)
	}
	return nil
}

func (s *SQLEventStore) Save(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Optimistic concurrency check
	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	var currentVersion int
	err = tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(version), 0) FROM user_events WHERE aggregate_id = `+p1+` AND aggregate_type = `+p2,
		ref.ID.String(), string(ref.Type),
	).Scan(&currentVersion)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check version: %w", err)
	}
	if event.Version(currentVersion) != expectedVersion {
		return event.NewConflict("sql_event_store.version_mismatch",
			fmt.Sprintf("expected version %d, got %d", expectedVersion, currentVersion))
	}

	for _, evt := range events {
		if err := s.insertEvent(ctx, tx, evt); err != nil {
			return fmt.Errorf("insert event %s: %w", evt.ID(), err)
		}
	}

	return commitTx(tx)
}

func (s *SQLEventStore) AppendBatch(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, evt := range events {
		if err := s.insertEvent(ctx, tx, evt); err != nil {
			return fmt.Errorf("insert event %s: %w", evt.ID(), err)
		}
	}
	return commitTx(tx)
}

func (s *SQLEventStore) insertEvent(ctx context.Context, tx *sql.Tx, evt event.Event) error {
	metadataJSON, err := json.Marshal(evt.Metadata())
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	p3 := s.placeholder(3)
	p4 := s.placeholder(4)
	p5 := s.placeholder(5)
	p6 := s.placeholder(6)
	p7 := s.placeholder(7)
	p8 := s.placeholder(8)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_events (event_id, event_type, aggregate_id, aggregate_type, version, payload, metadata, occurred_at)
		 VALUES (`+p1+`, `+p2+`, `+p3+`, `+p4+`, `+p5+`, `+p6+`, `+p7+`, `+p8+`)`,
		evt.ID().String(),
		string(evt.Type()),
		evt.AggregateID().String(),
		string(evt.AggregateType()),
		int(evt.Version()),
		evt.Payload(),
		metadataJSON,
		evt.OccurredAt(),
	)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}
	return nil
}

func (s *SQLEventStore) Load(
	ctx context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT event_id, event_type, aggregate_id, aggregate_type, version, payload, metadata, occurred_at
		 FROM user_events WHERE aggregate_id = `+p1+` AND aggregate_type = `+p2+`
		 ORDER BY version ASC`,
		ref.ID.String(), string(ref.Type),
	)
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return s.scanEvents(rows)
}

func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	p3 := s.placeholder(3)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT event_id, event_type, aggregate_id, aggregate_type, version, payload, metadata, occurred_at
		 FROM user_events WHERE aggregate_id = `+p1+` AND aggregate_type = `+p2+` AND version > `+p3+`
		 ORDER BY version ASC`,
		ref.ID.String(), string(ref.Type), int(version),
	)
	if err != nil {
		return nil, fmt.Errorf("load events from version: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return s.scanEvents(rows)
}

func (s *SQLEventStore) LoadToVersion(
	ctx context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	p3 := s.placeholder(3)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT event_id, event_type, aggregate_id, aggregate_type, version, payload, metadata, occurred_at
		 FROM user_events WHERE aggregate_id = `+p1+` AND aggregate_type = `+p2+` AND version <= `+p3+`
		 ORDER BY version ASC`,
		ref.ID.String(), string(ref.Type), int(maxVersion),
	)
	if err != nil {
		return nil, fmt.Errorf("load events to version: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return s.scanEvents(rows)
}

func (s *SQLEventStore) LoadToTimestamp(
	ctx context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	p3 := s.placeholder(3)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT event_id, event_type, aggregate_id, aggregate_type, version, payload, metadata, occurred_at
		 FROM user_events WHERE aggregate_id = `+p1+` AND aggregate_type = `+p2+` AND occurred_at <= `+p3+`
		 ORDER BY version ASC`,
		ref.ID.String(), string(ref.Type), maxTime,
	)
	if err != nil {
		return nil, fmt.Errorf("load events to timestamp: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return s.scanEvents(rows)
}

func (s *SQLEventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT event_id, event_type, aggregate_id, aggregate_type, version, payload, metadata, occurred_at
		 FROM user_events ORDER BY occurred_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("read all events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return s.scanEvents(rows)
}

func commitTx(tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *SQLEventStore) scanEvents(rows *sql.Rows) ([]event.Event, error) {
	var events []event.Event
	for rows.Next() {
		evt, err := s.scanRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return events, nil
}

func (*SQLEventStore) scanRow(rows *sql.Rows) (event.Event, error) {
	var (
		eventID      string
		eventType    string
		aggID        string
		aggType      string
		version      int
		payload      []byte
		metadataJSON []byte
		occurredAt   time.Time
	)
	if err := rows.Scan(
		&eventID, &eventType, &aggID, &aggType, &version,
		&payload, &metadataJSON, &occurredAt,
	); err != nil {
		return nil, fmt.Errorf("scan event row: %w", err)
	}

	evtID, err := id.ParseEventID(eventID)
	if err != nil {
		return nil, fmt.Errorf("parse event ID %q: %w", eventID, err)
	}

	parsedAggID, err := id.ParseAggregateID(aggID)
	if err != nil {
		return nil, fmt.Errorf("parse aggregate ID %q: %w", aggID, err)
	}

	opts := []event.Option{
		event.WithEventID(evtID),
		event.WithOccurredAt(occurredAt),
	}

	if len(metadataJSON) > 0 {
		var meta event.Metadata
		if err := json.Unmarshal(metadataJSON, &meta); err == nil {
			opts = append(opts, event.WithMetadata(meta))
		}
	}

	evt, err := event.NewEvent(
		event.Type(eventType),
		parsedAggID,
		event.AggregateType(aggType),
		event.Version(version),
		payload,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct event %q: %w", eventType, err)
	}
	return evt, nil
}

var (
	_ event.Store   = (*SQLEventStore)(nil)
	_ event.Journal = (*SQLEventStore)(nil)
)
