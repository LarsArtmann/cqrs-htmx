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
// with it — zero setup, zero persistence.
//
//	deployment := systemadapter.RecommendedMemoryDeployment()
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedMemoryDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			enginePrimary: {Driver: driverMemory},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{enginePrimary}},
			{Role: system.RoleProjections, Engines: []string{enginePrimary}},
		},
	}
}

// RecommendedSQLiteDeployment returns the recommended single-file persistent
// deployment: one SQLite engine (WAL) holding every role. The DSN is the
// database file, e.g. "file:app.db" — back it up and the whole system state
// travels with it.
//
//	deployment := systemadapter.RecommendedSQLiteDeployment("file:app.db")
//	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
func RecommendedSQLiteDeployment(dsn string) system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			enginePrimary: {Driver: driverSQLite, DSN: dsn, Pragmas: sqlitePragmas()},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{enginePrimary}},
			{Role: system.RoleProjections, Engines: []string{enginePrimary}},
		},
	}
}

// RecommendedSplitSQLiteDeployment returns the recommended production shape:
// the event journal and the projection engines live on SEPARATE SQLite files,
// so projection reads and replays never contend with the write path. Pair it
// with [WithCheckpointStore] + [WithDeadLetterStore] on stores backed by the
// projections file (or their own) for restart-safe positions.
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
// single named engine holds every role (source of truth AND projections),
// which is the correct topology for embedded engines and single-server
// deployments. The explicit RoleProjections instance is load-bearing: without
// one, the system wires projection state into an IMPLICIT in-memory engine,
// so a persistent single-file deployment loses every view on restart while
// its engine-backed checkpoints say nothing is left to replay — the exact
// starvation the W1.2 migration probe caught (2026-11-08).
func singleEngineDeployment(driver, dsn string) system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			enginePrimary: {Driver: driver, DSN: dsn},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{enginePrimary}},
			{Role: system.RoleProjections, Engines: []string{enginePrimary}},
		},
	}
}
