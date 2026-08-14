package usermgmt

import (
	"database/sql"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func newSQLiteTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestNewSQLReadModelsForDialect_SQLiteFamily(t *testing.T) {
	db := newSQLiteTestDB(t)

	for _, dialect := range []string{"", "sqlite", "sqlite3"} {
		t.Run("dialect_"+dialect, func(t *testing.T) {
			rms, err := newSQLReadModelsForDialect(db, dialect)
			if err != nil {
				t.Fatalf("newSQLReadModelsForDialect(%q): %v", dialect, err)
			}
			if rms.user == nil || rms.membership == nil || rms.tenant == nil || rms.bot == nil {
				t.Fatal("expected all four read models to be non-nil")
			}
		})
	}
}

func TestNewSQLReadModelsForDialect_Unsupported(t *testing.T) {
	db := newSQLiteTestDB(t)

	_, err := newSQLReadModelsForDialect(db, "oracle")
	if err == nil {
		t.Fatal("expected error for unsupported dialect")
	}
	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Errorf("expected Rejection classification, got %v", errorfamily.Classify(err))
	}
}
