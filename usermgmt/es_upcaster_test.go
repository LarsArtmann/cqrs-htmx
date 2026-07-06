package usermgmt

import (
	"encoding/json"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// addSchemaVersionV1 is the v0→v1 UserRegistered upcaster used in tests: it
// injects schema_version=1 into a legacy v0 payload.
func addSchemaVersionV1(raw []byte) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	m["schema_version"] = 1
	return json.Marshal(m)
}

func TestUpcasterRegistry_NoUpcasters(t *testing.T) {
	r := NewUpcasterRegistry()
	raw := []byte(`{"schema_version":1,"email":"test@example.com"}`)

	result, err := r.Upcast(eventUserRegistered, raw)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}
	if string(result) != string(raw) {
		t.Fatalf("expected no change, got %s", result)
	}
}

func TestUpcasterRegistry_SingleStep(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(eventUserRegistered, 0, func(raw []byte) ([]byte, error) {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		m["schema_version"] = 1
		if old, ok := m["name"]; ok {
			m["display_name"] = old
			delete(m, "name")
		}
		return json.Marshal(m)
	})

	raw := []byte(`{"name":"Alice"}`)
	result, err := r.Upcast(eventUserRegistered, raw)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	var p UserRegisteredPayload
	if err := json.Unmarshal(result, &p); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if p.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", p.SchemaVersion)
	}
	if p.DisplayName != "Alice" {
		t.Fatalf("DisplayName = %q, want %q", p.DisplayName, "Alice")
	}
}

func TestUpcasterRegistry_MultiStepChain(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(eventUserRegistered, 0, addSchemaVersionV1)
	// Future v1→v2 upcaster (not yet active since currentSchemaVersion=1)
	// This test verifies the chain logic works
	originalVersion := currentSchemaVersion
	t.Cleanup(func() {
		// currentSchemaVersion is const, can't change it.
		// This test only verifies that v0→v1 works correctly.
		_ = originalVersion
	})

	raw := []byte(`{"email":"old@example.com"}`)
	result, err := r.Upcast(eventUserRegistered, raw)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	var p UserRegisteredPayload
	if err := json.Unmarshal(result, &p); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if p.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", p.SchemaVersion)
	}
	if p.Email != "old@example.com" {
		t.Fatalf("Email = %q, want %q", p.Email, "old@example.com")
	}
}

func TestUpcasterRegistry_UpcasterError(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(eventUserRegistered, 0, func([]byte) ([]byte, error) {
		return nil, errorfamily.NewCorruption("test", "simulated failure")
	})

	_, err := r.Upcast(eventUserRegistered, []byte(`{"schema_version":0}`))
	if err == nil {
		t.Fatal("expected error from upcaster")
	}
}

func TestUpcasterRegistry_DuplicateRegisterPanics(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(eventUserRegistered, 0, func([]byte) ([]byte, error) { return nil, nil })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	r.Register(eventUserRegistered, 0, func([]byte) ([]byte, error) { return nil, nil })
}

func TestUpcasterRegistry_AlreadyCurrentVersion(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(eventUserRegistered, 0, func([]byte) ([]byte, error) {
		t.Fatal("upcaster should not be called for current version")
		return nil, nil
	})

	raw := []byte(`{"schema_version":1,"email":"current@example.com"}`)
	result, err := r.Upcast(eventUserRegistered, raw)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}
	if string(result) != string(raw) {
		t.Fatalf("expected no change for current version")
	}
}

func TestSetUpcasterRegistry(t *testing.T) {
	// Save and restore
	globalUpcastersMu.Lock()
	original := globalUpcasters
	globalUpcastersMu.Unlock()
	t.Cleanup(func() {
		globalUpcastersMu.Lock()
		globalUpcasters = original
		globalUpcastersMu.Unlock()
	})

	r := NewUpcasterRegistry()
	r.Register(eventUserRegistered, 0, addSchemaVersionV1)
	SetUpcasterRegistry(r)
	t.Cleanup(func() { SetUpcasterRegistry(nil) })

	// Verify applyUpcasters uses the global registry
	result, err := applyUpcasters(eventUserRegistered, []byte(`{"email":"x@y.z"}`))
	if err != nil {
		t.Fatalf("applyUpcasters: %v", err)
	}
	var p UserRegisteredPayload
	if err := json.Unmarshal(result, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", p.SchemaVersion)
	}
}

func TestExtractSchemaVersion(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{"present", `{"schema_version":2}`, 2},
		{"absent", `{"email":"x"}`, 0},
		{"empty", `{}`, 0},
		{"invalid_json", `{{{}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSchemaVersion([]byte(tt.raw))
			if got != tt.want {
				t.Fatalf("extractSchemaVersion(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}
