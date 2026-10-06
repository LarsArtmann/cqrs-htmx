package systemadapter_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// recordingCheckpointStore observes checkpoint traffic through a backing
// store, so the restart test can prove the declarative host both persists
// and resumes positions through the consumer-supplied store.
type recordingCheckpointStore struct {
	mu     sync.Mutex
	inner  event.CheckpointStore
	saved  map[string]event.Checkpoint
	loaded map[string]event.Checkpoint
}

func newRecordingCheckpointStore(inner event.CheckpointStore) *recordingCheckpointStore {
	return &recordingCheckpointStore{
		inner:  inner,
		saved:  map[string]event.Checkpoint{},
		loaded: map[string]event.Checkpoint{},
	}
}

func (r *recordingCheckpointStore) Save(
	ctx context.Context, projectionName string, cp event.Checkpoint,
) error {
	if err := r.inner.Save(ctx, projectionName, cp); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.saved[projectionName] = cp

	return nil
}

func (r *recordingCheckpointStore) Load(
	ctx context.Context, projectionName string,
) (event.Checkpoint, error) {
	cp, err := r.inner.Load(ctx, projectionName)
	if err != nil {
		return event.Checkpoint{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.loaded[projectionName] = cp

	return cp, nil
}

func (r *recordingCheckpointStore) snapshotSaved() map[string]event.Checkpoint {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := make(map[string]event.Checkpoint, len(r.saved))
	for name, cp := range r.saved {
		snapshot[name] = cp
	}

	return snapshot
}

func (r *recordingCheckpointStore) snapshotLoaded() map[string]event.Checkpoint {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := make(map[string]event.Checkpoint, len(r.loaded))
	for name, cp := range r.loaded {
		snapshot[name] = cp
	}

	return snapshot
}

func (r *recordingCheckpointStore) resetLoaded() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.loaded = map[string]event.Checkpoint{}
}

// fileSQLiteDeployment returns a DeploymentConfig backed by a FILE-based
// SQLite journal, so the event store survives Close() and a second
// system.New() sees the same events.
func fileSQLiteDeployment(t *testing.T) system.DeploymentConfig {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "journal.db")

	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {
				Driver:  "sqlite",
				DSN:     "file:" + dsn,
				Pragmas: []string{"journal_mode=wal"},
			},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}
}

// TestDeclarative_DurableCheckpointSurvivesRestart proves the declarative
// path resumes projection positions from a consumer-supplied checkpoint
// store across a full system restart: system one saves non-zero checkpoints
// while draining a persistent journal, and system two — same journal, same
// store, fresh host — loads exactly those positions instead of starting
// blind.
//
// It deliberately does NOT assert read-model content after the restart: a
// fresh in-memory read model with an up-to-date checkpoint processes nothing
// new (that is the point of the checkpoint); durable read models or explicit
// rebuilds are the pairing documented on WithCheckpointStore.
func TestDeclarative_DurableCheckpointSurvivesRestart(t *testing.T) {
	deployment := fileSQLiteDeployment(t)
	durable := newRecordingCheckpointStore(memorystorage.NewMemoryCheckpointStore())
	ctx := context.Background()

	sys1, err := system.New(ctx, systemadapter.DomainConfig(
		systemadapter.WithCheckpointStore(durable),
	), deployment)
	if err != nil {
		t.Fatalf("system.New (first boot): %v", err)
	}

	if err := sys1.Start(ctx); err != nil {
		_ = sys1.Close()

		t.Fatalf("sys1.Start: %v", err)
	}

	waitForProjectionsReady(t, sys1)

	must(t, sys1.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		id.NewStreamID(), "Acme Corp", "Acme",
	)))

	eventually(t, 10*time.Second, func() error {
		for _, cp := range durable.snapshotSaved() {
			if !cp.EventID.IsZero() {
				return nil
			}
		}

		return errors.New("no projection has saved a non-zero checkpoint yet")
	})

	if err := sys1.Close(); err != nil {
		t.Fatalf("sys1.Close: %v", err)
	}

	saved := durable.snapshotSaved()
	if len(saved) == 0 {
		t.Fatal("no checkpoints were saved through the consumer-supplied store")
	}

	durable.resetLoaded()

	sys2, err := system.New(ctx, systemadapter.DomainConfig(
		systemadapter.WithCheckpointStore(durable),
	), deployment)
	if err != nil {
		t.Fatalf("system.New (second boot): %v", err)
	}

	t.Cleanup(func() { _ = sys2.Close() })

	if err := sys2.Start(ctx); err != nil {
		t.Fatalf("sys2.Start: %v", err)
	}

	waitForProjectionsReady(t, sys2)

	loaded := durable.snapshotLoaded()
	if len(loaded) == 0 {
		t.Fatal("second boot never consulted the durable checkpoint store — CheckpointStore not wired")
	}

	for name, want := range saved {
		got, ok := loaded[name]
		if !ok {
			t.Errorf("projection %q saved a checkpoint but the second boot never loaded it", name)

			continue
		}

		if want.EventID.IsZero() {
			t.Errorf("projection %q saved a zero checkpoint — no progress was recorded", name)

			continue
		}

		if got.EventID != want.EventID {
			t.Errorf("projection %q resumed at %v, want the saved %v", name, got.EventID, want.EventID)
		}
	}
}

// TestDomainConfig_DefaultsAndOverrides pins the option plumbing: no options
// means the curated default host options and no checkpoint override, while
// WithHostOptions appends after the defaults so per-field consumer options
// win.
func TestDomainConfig_DefaultsAndOverrides(t *testing.T) {
	base := systemadapter.DomainConfig()

	if len(base.ProjectionHostOptions) != 4 {
		t.Errorf(
			"default ProjectionHostOptions = %d, want 4 (dead-letter, restarts, backoff, batch)",
			len(base.ProjectionHostOptions),
		)
	}

	if base.CheckpointStore != nil {
		t.Error("CheckpointStore must stay nil by default — system.New's engine-backed default beats a memory store")
	}

	store := memorystorage.NewMemoryCheckpointStore()
	custom := systemadapter.DomainConfig(
		systemadapter.WithCheckpointStore(store),
		systemadapter.WithHostOptions(projectionhost.WithBatchSize(7)),
	)

	if custom.CheckpointStore == nil {
		t.Error("WithCheckpointStore was not applied")
	}

	if len(custom.ProjectionHostOptions) != 5 {
		t.Errorf(
			"custom ProjectionHostOptions = %d, want 4 defaults + 1 consumer option",
			len(custom.ProjectionHostOptions),
		)
	}
}
