package systemadapter

import (
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Engine and driver names shared by the Recommended* presets.
const (
	enginePrimary     = "primary"
	engineEvents      = "events"
	engineProjections = "projections"
	driverMemory      = "memory"
	driverSQLite      = "sqlite"
	driverPebble      = "pebble"
	driverBBolt       = "bbolt"
	driverPostgres    = "postgres"
	driverDuckDB      = "duckdb"
	driverTurso       = "turso"
)

// sqlitePragmas returns the recommended SQLite posture for an event
// journal: WAL keeps readers (projections, dashboards) concurrent with the
// writer. A fresh slice per call — engines may mutate their config.
func sqlitePragmas() []string {
	return []string{"journal_mode=wal"}
}

// RecommendedMemoryDeployment returns the recommended dev/test deployment:
// ONE memory engine holding every role. Everything lives in-process and dies
// with it — zero setup, zero persistence. The instance triple is filled by
// system.Standard (ADR-0165): source-of-truth + queries on the engine,
// projections pooled over it.
//
//	deployment := systemadapter.RecommendedMemoryDeployment()
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedMemoryDeployment() system.DeploymentConfig {
	return system.Standard(system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			enginePrimary: {Driver: driverMemory},
		},
	})
}

// RecommendedSQLiteDeployment returns the recommended single-file persistent
// deployment: one SQLite engine (WAL) holding every role. The DSN is the
// database file, e.g. "file:app.db" — back it up and the whole system state
// travels with it. The instance triple is filled by system.Standard
// (ADR-0165).
//
//	deployment := systemadapter.RecommendedSQLiteDeployment("file:app.db")
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedSQLiteDeployment(dsn string) system.DeploymentConfig {
	return system.Standard(system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			enginePrimary: {Driver: driverSQLite, DSN: dsn, Pragmas: sqlitePragmas()},
		},
	})
}

// RecommendedSplitSQLiteDeployment keeps its EXPLICIT instances on purpose:
// the split topology isolates the journal and projection files, while
// system.Standard pools projections over EVERY declared engine — pooling here
// would write projections into the journal's file. Standard is a passthrough
// over explicit instances (ADR-0165), so this preset stays hand-written.
//
//	deployment := systemadapter.RecommendedSplitSQLiteDeployment(
//		systemadapter.SplitSQLiteDSNs{
//			Events:      "file:events.db",
//			Projections: "file:projections.db",
//		},
//	)
//	sys, err := system.New(ctx, systemadapter.DomainConfig(
//		systemadapter.WithCheckpointStore(durableCheckpoints),
//	), deployment)

// SplitSQLiteDSNs carries the two file DSNs a split deployment needs as
// NAMED fields: two bare string parameters shared one type, so transposing
// them compiled silently and wired the projections engine onto the journal's
// file (the PHANTOM-PLUMBING swap-hazard class).
type SplitSQLiteDSNs struct {
	Events      string
	Projections string
}

func RecommendedSplitSQLiteDeployment(dsns SplitSQLiteDSNs) system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			engineEvents:      {Driver: driverSQLite, DSN: dsns.Events, Pragmas: sqlitePragmas()},
			engineProjections: {Driver: driverSQLite, DSN: dsns.Projections, Pragmas: sqlitePragmas()},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{engineEvents}},
			{Role: system.RoleProjections, Engines: []string{engineProjections}},
		},
	}
}

// RecommendedPebbleDeployment returns a persistent deployment on the Pebble
// LSM-tree engine: one engine holding every role, DSN is the data directory
// (created if missing). Pebble's point reads make it a good fit for
// read-heavy projection workloads on a single node.
// The driver self-registers; link it with a blank import:
//
//	import _ "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
//
//	deployment := systemadapter.RecommendedPebbleDeployment("./data/pebble")
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedPebbleDeployment(dir string) system.DeploymentConfig {
	return singleEngineDeployment(driverPebble, dir)
}

// RecommendedBBoltDeployment returns a persistent deployment on the bbolt
// B+tree engine: one engine holding every role, DSN is the single database
// FILE (created if missing). bbolt is fully transactional and dependency-light
// — the embedded choice when one file must travel with the app.
// The driver self-registers; link it with a blank import:
//
//	import _ "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
//
//	deployment := systemadapter.RecommendedBBoltDeployment("app.db")
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedBBoltDeployment(path string) system.DeploymentConfig {
	return singleEngineDeployment(driverBBolt, path)
}

// RecommendedPostgresDeployment returns a server deployment on PostgreSQL:
// one engine holding every role, DSN is the postgres connection string
// (e.g. "postgres://user:pass@host:5432/app?sslmode=require"). The engine
// creates its schema on first start; point every replica at the same DSN.
// The driver self-registers; link it with a blank import:
//
//	import _ "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
//
//	deployment := systemadapter.RecommendedPostgresDeployment(dsn)
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedPostgresDeployment(dsn string) system.DeploymentConfig {
	return singleEngineDeployment(driverPostgres, dsn)
}

// RecommendedDuckDBDeployment returns a deployment on the DuckDB analytics
// engine: one engine holding every role, DSN is the database file (":memory:"
// for in-process). DuckDB's columnar scans make it the choice when projection
// queries are aggregate-heavy rather than point reads.
// The driver self-registers; link it with a blank import:
//
//	import _ "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
//
//	deployment := systemadapter.RecommendedDuckDBDeployment("analytics.db")
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedDuckDBDeployment(path string) system.DeploymentConfig {
	return singleEngineDeployment(driverDuckDB, path)
}

// RecommendedTursoDeployment returns a deployment on the Turso/libSQL engine:
// one engine holding every role. An empty DSN selects the embedded in-memory
// database (":memory:"); a "libsql://" URL selects a remote Turso server —
// the shape that unlocks operator-owned materialized views. Authentication
// for remote URLs rides the DSN (query parameter or embedded token).
// The driver self-registers; link it with a blank import:
//
//	import _ "github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
//
//	deployment := systemadapter.RecommendedTursoDeployment("libsql://db.turso.io? authToken=...")
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedTursoDeployment(dsn string) system.DeploymentConfig {
	return singleEngineDeployment(driverTurso, dsn)
}

// singleEngineDeployment is the shared shape of the one-engine presets: a
// single named engine holds every role (source-of-truth, the query audit
// log, AND projections), which is the correct topology for embedded engines
// and single-server deployments. system.Standard fills the canonical triple
// (ADR-0165) — including the explicit projections instance that keeps views
// off the implicit volatile fallback the W1.2 migration probe caught
// (2026-11-08) and a named queries instance the audit log can later move to
// its own engine from.
func singleEngineDeployment(driver, dsn string) system.DeploymentConfig {
	return system.Standard(system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			enginePrimary: {Driver: driver, DSN: dsn},
		},
	})
}
