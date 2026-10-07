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
//		"file:events.db", "file:projections.db",
//	)
//	sys, err := system.New(ctx, systemadapter.DomainConfig(
//		systemadapter.WithCheckpointStore(durableCheckpoints),
//	), deployment)
func RecommendedSplitSQLiteDeployment(eventsDSN, projectionsDSN string) system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			engineEvents:      {Driver: driverSQLite, DSN: eventsDSN, Pragmas: sqlitePragmas()},
			engineProjections: {Driver: driverSQLite, DSN: projectionsDSN, Pragmas: sqlitePragmas()},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{engineEvents}},
			{Role: system.RoleProjections, Engines: []string{engineProjections}},
		},
	}
}
