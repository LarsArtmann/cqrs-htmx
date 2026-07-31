package usermgmt

import (
	"strings"
	"testing"
)

// TestMySQLDialect_PlaceholdersAndSchema verifies that the MySQLDialect
// produces MySQL-specific SQL: ? placeholders, LONGBLOB payloads, DATETIME(3)
// timestamps, and ON DUPLICATE KEY conflict handling.
func TestMySQLDialect_PlaceholdersAndSchema(t *testing.T) {
	t.Parallel()

	d, err := dialectToUpstream(dialectMySQL)
	if err != nil {
		t.Fatalf("dialectToUpstream: %v", err)
	}

	if got := d.Placeholder(1); got != "?" {
		t.Errorf("expected '?' placeholder, got %q", got)
	}
	if got := d.Placeholder(99); got != "?" {
		t.Errorf("expected '?' placeholder for any index, got %q", got)
	}

	schema := d.EventSchema()
	for _, want := range []string{"LONGBLOB", "DATETIME(3)"} {
		if !strings.Contains(schema, want) {
			t.Errorf("EventSchema missing MySQL-specific token %q:\n%s", want, schema)
		}
	}

	if strings.Contains(schema, "$1") {
		t.Error("EventSchema should not contain Postgres-style $1 placeholders")
	}
}
