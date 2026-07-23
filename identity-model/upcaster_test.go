package identitymodel

import (
	"encoding/json/v2"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

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
	raw := []byte(`{"schema_version":2,"email":"test@example.com"}`)

	result, err := r.Upcast(EventUserRegistered, raw)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}
	if string(result) != string(raw) {
		t.Fatalf("expected no change, got %s", result)
	}
}

func TestUpcasterRegistry_SingleStep(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(EventUserRegistered, 0, func(raw []byte) ([]byte, error) {
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
	result, err := r.Upcast(EventUserRegistered, raw)
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

func TestUpcasterRegistry_UpcasterError(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(EventUserRegistered, 0, func([]byte) ([]byte, error) {
		return nil, errorfamily.NewCorruption("test", "simulated failure")
	})

	_, err := r.Upcast(EventUserRegistered, []byte(`{"schema_version":0}`))
	if err == nil {
		t.Fatal("expected error from upcaster")
	}
}

func TestUpcasterRegistry_DuplicateRegisterPanics(t *testing.T) {
	r := NewUpcasterRegistry()
	r.Register(EventUserRegistered, 0, func([]byte) ([]byte, error) { return nil, nil })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	r.Register(EventUserRegistered, 0, func([]byte) ([]byte, error) { return nil, nil })
}

func TestSetUpcasterRegistry(t *testing.T) {
	globalUpcastersMu.Lock()
	original := globalUpcasters
	globalUpcastersMu.Unlock()
	t.Cleanup(func() {
		globalUpcastersMu.Lock()
		globalUpcasters = original
		globalUpcastersMu.Unlock()
	})

	r := NewUpcasterRegistry()
	r.Register(EventUserRegistered, 0, addSchemaVersionV1)
	SetUpcasterRegistry(r)
	t.Cleanup(func() { SetUpcasterRegistry(nil) })

	result, err := applyUpcasters(EventUserRegistered, []byte(`{"email":"x@y.z"}`))
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
