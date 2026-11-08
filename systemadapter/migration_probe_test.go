package systemadapter_test

// Migration probes for the usermgmt -> system.New migration plan
// (docs/planning/2026-11-07_10-45_SUPERB-usermgmt-system-metaengine-migration-plan.md
// §7 W1.1/W1.2, executed 2026-11-08 as the 06-35 plan's M03).
//
// W1.1 pins the FAÇADE SEAM SURFACE: every system accessor the
// NewServiceFromSystem façade needs (plan §4's "replaced" column) is
// compile-pinned here, so an upstream rename or removal breaks this file
// first — not the façade spike — and the composed wiring is proven live on
// a preset boot.
//
// W1.2 is the STARVATION PROBE — the honest gap of §6: a volatile
// projection engine paired with a durable checkpoint store can, after a
// restart, resume past events into a fresh (empty) view. The probe asserts
// the plan's bar for W3/W4: the failure mode is ABSENT on preset
// deployments, and on the mismatch deployment it is NEVER SILENT (data, or
// a surfaced diagnostic). A silent emptiness here flips the §6 gate to
// NO-GO.

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// -------------------------------------------------------------------------
// W1.1 — façade seam surface (compile-pins + live preset boot)
// -------------------------------------------------------------------------

// TestMigrationProbe_W11_FacadeSeamSurface compile-pins the seam accessors
// the NewService façade consumes (migration plan §4). Method values bound
// to a nil *System: taking the value never calls the method, but any
// upstream signature change stops this file from compiling.
// The alias types ARE the pin: each blank-var assignment compiles only
// while the accessor keeps exactly the aliased signature (an upstream
// rename/arity change breaks this file first, not the façade spike).
type (
	facadeMetaEngine        = func() *metaengine.Store
	facadeCommandDispatcher = func() *command.Dispatcher
	facadeQueryDispatcher   = func() *query.Dispatcher
	facadePublisher         = func() event.Publisher
	facadeProjectionHost    = func() *projectionhost.Host
	facadeBus               = func() event.Bus
	facadeEventStore        = func() event.Store
	facadeSnapshotStore     = func() snapshot.SnapshotStore
	facadeUseCmdMiddleware  = func(...command.Middleware)
	facadeProjectionExplain = func() string
)

var (
	_ facadeMetaEngine        = (*system.System)(nil).MetaEngine
	_ facadeCommandDispatcher = (*system.System)(nil).CommandDispatcher
	_ facadeQueryDispatcher   = (*system.System)(nil).QueryDispatcher
	_ facadePublisher         = (*system.System)(nil).Publisher
	_ facadeProjectionHost    = (*system.System)(nil).ProjectionHost
	_ facadeBus               = (*system.System)(nil).Bus
	_ facadeEventStore        = (*system.System)(nil).EventStore
	_ facadeSnapshotStore     = (*system.System)(nil).SnapshotStore
	_ facadeUseCmdMiddleware  = (*system.System)(nil).UseCommandMiddleware
	_ facadeProjectionExplain = (*system.System)(nil).ProjectionExplain
)

// TestMigrationProbe_W11_FacadeSeamSurface proves the pinned seams are live:
// a preset boot reaches ready with every accessor non-nil, and one command
// through the system dispatcher lands in the declarative view.
func TestMigrationProbe_W11_FacadeSeamSurface(t *testing.T) {
	sys, err := system.New(context.Background(),
		systemadapter.DomainConfig(),
		systemadapter.RecommendedMemoryDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	if err := sys.Start(context.Background()); err != nil {
		t.Fatalf("sys.Start: %v", err)
	}

	waitForProjectionsReady(t, sys)

	if sys.CommandDispatcher() == nil {
		t.Fatal("façade seam: CommandDispatcher() is nil after Start")
	}

	if sys.EventStore() == nil {
		t.Fatal("façade seam: EventStore() is nil after Start")
	}

	if sys.Bus() == nil {
		t.Fatal("façade seam: Bus() is nil after Start")
	}

	if sys.ProjectionHost() == nil {
		t.Fatal("façade seam: ProjectionHost() is nil after Start")
	}

	if sys.ScreamReport() == nil {
		t.Fatal("façade seam: ScreamReport() is nil — diagnostics surface gone")
	}

	if explain := sys.ProjectionExplain(); explain == "" {
		t.Fatal("façade seam: ProjectionExplain() is empty on a booted system")
	}

	// The wiring map composes end-to-end: one command through the system
	// dispatcher lands in the declarative view (the seam replacement proof).
	ctx := context.Background()

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		id.NewStreamID(), "Facade Probe Corp", "facade-probe",
	)))

	eventually(t, 10*time.Second, func() error {
		_, err := systemadapter.FindTenantByName(ctx, sys, "Facade Probe Corp")

		return err
	})
}

// -------------------------------------------------------------------------
// W1.2 — starvation probe
// -------------------------------------------------------------------------

// dispatchTenantAndWait boots flow: dispatch CreateTenant, wait for the
// view AND for a non-zero checkpoint through the store, so the restart leg
// cannot race an unfinished drain.
func dispatchTenantAndWait(t *testing.T, sys *system.System, cp *recordingCheckpointStore, name string) {
	t.Helper()

	ctx := context.Background()

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		id.NewStreamID(), name, strings.ToLower(strings.Join(strings.Fields(name), "-")),
	)))

	eventually(t, 10*time.Second, func() error {
		_, err := systemadapter.FindTenantByName(ctx, sys, name)
		if err != nil {
			return err
		}

		if cp != nil {
			for _, checkpoint := range cp.snapshotSaved() {
				if !checkpoint.EventID.IsZero() {
					return nil
				}
			}

			return errors.New("view visible but no checkpoint saved yet")
		}

		return nil
	})
}

func tenantVisible(t *testing.T, sys *system.System, name string) bool {
	t.Helper()

	_, err := systemadapter.FindTenantByName(context.Background(), sys, name)

	return err == nil
}

// TestMigrationProbe_W12_PresetRestartsDoNotStarve proves §6's gate
// condition (a) for both persistent presets: after a full restart the view
// is populated again — engine-resident fold state (single-file preset) and
// engine-resident state plus a surviving consumer checkpoint store (split
// preset) both re-serve the tenant without a replay.
func TestMigrationProbe_W12_PresetRestartsDoNotStarve(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("single-file preset, engine-backed checkpoints", func(t *testing.T) {
		dsn := "file:" + filepath.Join(t.TempDir(), "single.db")

		sys1, err := system.New(ctx, systemadapter.DomainConfig(),
			systemadapter.RecommendedSQLiteDeployment(dsn))
		if err != nil {
			t.Fatalf("boot 1: %v", err)
		}

		if err := sys1.Start(ctx); err != nil {
			_ = sys1.Close()

			t.Fatalf("start 1: %v", err)
		}

		waitForProjectionsReady(t, sys1)
		dispatchTenantAndWait(t, sys1, nil, "Preset One Corp")

		if err := sys1.Close(); err != nil {
			t.Fatalf("close 1: %v", err)
		}

		sys2, err := system.New(ctx, systemadapter.DomainConfig(),
			systemadapter.RecommendedSQLiteDeployment(dsn))
		if err != nil {
			t.Fatalf("boot 2: %v", err)
		}

		t.Cleanup(func() { _ = sys2.Close() })

		if err := sys2.Start(ctx); err != nil {
			t.Fatalf("start 2: %v", err)
		}

		waitForProjectionsReady(t, sys2)

		if !tenantVisible(t, sys2, "Preset One Corp") {
			t.Fatal("STARVATION on the single-file preset: view empty after restart " +
				"(engine-resident state was expected to survive)")
		}
	})

	t.Run("split preset, durable consumer checkpoints", func(t *testing.T) {
		eventsDSN := "file:" + filepath.Join(t.TempDir(), "events.db")
		projectionsDSN := "file:" + filepath.Join(t.TempDir(), "projections.db")
		durable := newRecordingCheckpointStore(memorystorage.NewMemoryCheckpointStore())

		boot := func() (*system.System, error) {
			return system.New(ctx, systemadapter.DomainConfig(
				systemadapter.WithCheckpointStore(durable),
			), systemadapter.RecommendedSplitSQLiteDeployment(systemadapter.SplitSQLiteDSNs{
				Events:      eventsDSN,
				Projections: projectionsDSN,
			}))
		}

		sys1, err := boot()
		if err != nil {
			t.Fatalf("boot 1: %v", err)
		}

		if err := sys1.Start(ctx); err != nil {
			_ = sys1.Close()

			t.Fatalf("start 1: %v", err)
		}

		waitForProjectionsReady(t, sys1)
		dispatchTenantAndWait(t, sys1, durable, "Preset Two Corp")

		if err := sys1.Close(); err != nil {
			t.Fatalf("close 1: %v", err)
		}

		sys2, err := boot()
		if err != nil {
			t.Fatalf("boot 2: %v", err)
		}

		t.Cleanup(func() { _ = sys2.Close() })

		if err := sys2.Start(ctx); err != nil {
			t.Fatalf("start 2: %v", err)
		}

		waitForProjectionsReady(t, sys2)

		if !tenantVisible(t, sys2, "Preset Two Corp") {
			t.Fatal("STARVATION on the split preset: view empty after restart " +
				"(persistent projection engine + durable checkpoints were expected to re-serve state)")
		}
	})
}

// TestMigrationProbe_W12_MismatchNeverSilent is the honest-gap probe: a
// durable journal + VOLATILE projection engine + a checkpoint store that
// survives the restart (the pairing WithCheckpointStore's contract warns
// about). After the restart the view is empty BY CONSTRUCTION; the probe
// requires the system to SAY SO — plan diagnostics (ProjectionExplain),
// the scream report, or a boot-time WARN log. Silent emptiness flips the
// migration plan §6 gate to NO-GO.
func TestMigrationProbe_W12_MismatchNeverSilent(t *testing.T) {
	ctx := context.Background()

	journal := "file:" + filepath.Join(t.TempDir(), "events.db")
	durable := newRecordingCheckpointStore(memorystorage.NewMemoryCheckpointStore())

	mismatchDeployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"events": {Driver: "sqlite", DSN: journal, Pragmas: []string{"journal_mode=wal"}},
			"views":  {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{"events"}},
			{Role: system.RoleProjections, Engines: []string{"views"}},
		},
	}

	boot := func() (*system.System, error) {
		return system.New(ctx, systemadapter.DomainConfig(
			systemadapter.WithCheckpointStore(durable),
		), mismatchDeployment)
	}

	sys1, err := boot()
	if err != nil {
		t.Fatalf("boot 1: %v", err)
	}

	if err := sys1.Start(ctx); err != nil {
		_ = sys1.Close()

		t.Fatalf("start 1: %v", err)
	}

	waitForProjectionsReady(t, sys1)
	dispatchTenantAndWait(t, sys1, durable, "Mismatch Corp")

	if err := sys1.Close(); err != nil {
		t.Fatalf("close 1: %v", err)
	}

	// Boot two: fresh volatile engine, same journal, same surviving
	// checkpoints — capture every log line the boot path emits.
	var logs bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	sys2, err := boot()
	if err != nil {
		t.Fatalf("boot 2: %v", err)
	}

	t.Cleanup(func() { _ = sys2.Close() })

	if err := sys2.Start(ctx); err != nil {
		t.Fatalf("start 2: %v", err)
	}

	waitForProjectionsReady(t, sys2)

	slog.SetDefault(previous)

	if tenantVisible(t, sys2, "Mismatch Corp") {
		t.Log("mismatch boot re-served the view (full replay) — no starvation to diagnose")

		return
	}

	// The view IS empty (expected for a volatile engine). Require a loud
	// diagnostic: plan diagnostics, scream findings, or a boot WARN.
	explain := sys2.ProjectionExplain()

	scream := sys2.ScreamReport()
	screamCount := 0
	if scream != nil {
		screamCount = len(scream.Diagnostics)
	}

	volatileSignals := []string{"volatile", "rebuild", "lost on restart", "durab"}

	explainLoud := containsAny(t, "ProjectionExplain", explain, volatileSignals)
	logLoud := containsAny(t, "boot logs", logs.String(), volatileSignals)

	t.Logf("W1.2 mismatch outcome: view empty (volatile engine, expected); "+
		"explain-loud=%v log-loud=%v scream-findings=%d", explainLoud, logLoud, screamCount)

	if !explainLoud && !logLoud && screamCount == 0 {
		t.Fatal("SILENT STARVATION: volatile projection engine + durable checkpoint " +
			"restart left the view empty with NO surfaced diagnostic — migration plan §6 " +
			"gate flips to NO-GO (W5 durability-pairing ask becomes blocking)")
	}
}

func containsAny(t *testing.T, what, haystack string, needles []string) bool {
	t.Helper()

	lower := strings.ToLower(haystack)

	for _, needle := range needles {
		if strings.Contains(lower, needle) {
			t.Logf("%s surfaces a volatility diagnostic (matched %q)", what, needle)

			return true
		}
	}

	return false
}
