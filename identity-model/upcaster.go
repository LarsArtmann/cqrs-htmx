package identitymodel

import (
	"encoding/json/v2"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Upcaster transforms raw event payload bytes from schema version N to version N+1.
// It receives the JSON bytes at version N and must return JSON bytes at version N+1.
type Upcaster func(raw []byte) ([]byte, error)

// UpcasterRegistry holds versioned upcasters per event type.
// It enables backward-compatible schema evolution: old events are transparently
// upgraded to the current schema version before being decoded.
type UpcasterRegistry struct {
	mu     sync.RWMutex
	chains map[event.Type]map[int]Upcaster
}

// NewUpcasterRegistry creates an empty UpcasterRegistry.
func NewUpcasterRegistry() *UpcasterRegistry {
	return &UpcasterRegistry{
		mu:     sync.RWMutex{},
		chains: make(map[event.Type]map[int]Upcaster),
	}
}

// Register adds an upcaster for a specific event type, transforming payloads
// from version fromVersion to fromVersion+1.
// Panics if an upcaster for the same (eventType, fromVersion) is already registered.
func (r *UpcasterRegistry) Register(eventType event.Type, fromVersion int, fn Upcaster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.chains[eventType] == nil {
		r.chains[eventType] = make(map[int]Upcaster)
	}
	if _, exists := r.chains[eventType][fromVersion]; exists {
		panic(
			fmt.Sprintf(
				"upcaster already registered for %s v%d→v%d", //cqrs-lint:ignore(C009) init-time duplicate registration guard
				eventType,
				fromVersion,
				fromVersion+1,
			),
		)
	}
	r.chains[eventType][fromVersion] = fn
}

// Upcast applies the chain of upcasters for the given event type, transforming
// the raw payload from its embedded schema version up to CurrentSchemaVersion.
// If no upcasters are registered for the event type, the raw bytes are returned as-is.
func (r *UpcasterRegistry) Upcast(eventType event.Type, raw []byte) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chain, exists := r.chains[eventType]
	if !exists || len(chain) == 0 {
		return raw, nil
	}

	version := extractSchemaVersion(raw)
	current := raw

	for version < CurrentSchemaVersion {
		fn, ok := chain[version]
		if !ok {
			break
		}
		upcasted, err := fn(current)
		if err != nil {
			return nil, errorfamily.Wrapf(err, event.Corruption, "identitymodel.upcast.failed",
				"upcast %s v%d→v%d", eventType, version, version+1)
		}
		current = upcasted
		version++
	}

	return current, nil
}

// extractSchemaVersion reads the schema_version field from raw JSON bytes
// without fully decoding the payload. Returns 0 if the field is absent
// (legacy events pre-dating schema versioning).
func extractSchemaVersion(raw []byte) int {
	var probe struct {
		SchemaVersion int `json:"schema_version"`
	}
	_ = json.Unmarshal(raw, &probe)
	return probe.SchemaVersion
}

// --- package-level registry ---

var (
	globalUpcastersMu sync.RWMutex
	globalUpcasters   *UpcasterRegistry
)

// SetUpcasterRegistry configures the global upcaster registry used by all
// event decode paths (FoldUser, read models, projections).
// Call once during service initialization. Pass nil to disable upcasting.
func SetUpcasterRegistry(r *UpcasterRegistry) {
	globalUpcastersMu.Lock()
	globalUpcasters = r
	globalUpcastersMu.Unlock()
}

func applyUpcasters(eventType event.Type, raw []byte) ([]byte, error) {
	globalUpcastersMu.RLock()
	r := globalUpcasters
	globalUpcastersMu.RUnlock()

	if r == nil {
		return raw, nil
	}
	return r.Upcast(eventType, raw)
}
