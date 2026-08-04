//go:build ignore

// Shared helpers used by the SQL-backed setup templates (sqlite_setup.go,
// postgres_setup.go, mysql_setup.go). Copy this file alongside whichever
// template you use and remove the //go:build ignore directive from all of them.

package usermgmt

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// readModelFactory creates the four domain read models (user, membership,
// tenant, bot) from a SQL database connection. When db is nil, implementations
// should return in-memory read models.
type readModelFactory func(db *sql.DB) (
	projection.Projection, projection.Projection, projection.Projection, projection.Projection, error,
)

// buildSQLEventSourcedSetupCore assembles the common eventSourcedSetupCore
// shared by all SQL-backed setup templates (SQLite, Postgres, MySQL).
// It builds repositories, read models, Casbin authz, and projection host
// from a stack.Bundle, closing the bundle on any error.
func buildSQLEventSourcedSetupCore(
	backendName string,
	bundle *stack.Bundle,
	auditLog *AuditLog,
	checkpointStore event.CheckpointStore,
	snap SnapshotConfig,
	createReadModels readModelFactory,
) (eventSourcedSetupCore, error) {
	repos, err := buildStackRepositories(bundle, snap)
	if err != nil {
		return eventSourcedSetupCore{}, err
	}

	db := extractDB(bundle)
	rm, memRm, tenRm, botRm, err := createReadModels(db)
	if err != nil {
		_ = bundle.Close()
		return eventSourcedSetupCore{}, err
	}

	casbinProj, err := createAuthzAndCasbin()
	if err != nil {
		_ = bundle.Close()
		return eventSourcedSetupCore{}, err
	}

	host, err := StartProjections(
		bundle.Journal, bundle.Subscriber,
		checkpointStore,
		rm, memRm, tenRm, botRm, casbinProj, auditLog,
	)
	if err != nil {
		_ = bundle.Close()
		return eventSourcedSetupCore{}, errorfamily.WrapTransient(err, "usermgmt.projection.start", "start projections")
	}

	return eventSourcedSetupCore{
		backendName:          backendName,
		UserRepository:       repos.User,
		MembershipRepository: repos.Membership,
		TenantRepository:     repos.Tenant,
		BotRepository:        repos.Bot,
		ReadModel:            rm,
		MembershipReadModel:  memRm,
		TenantReadModel:      tenRm,
		BotReadModel:         botRm,
		Bundle:               bundle,
		DB:                   db,
		casbinProjection:     casbinProj,
		projectionHost:       host,
	}, nil
}

// extractDB pulls the *sql.DB from a stack bundle's database handle.
func extractDB(bundle *stack.Bundle) *sql.DB {
	db, _ := bundle.Database().(*sql.DB)
	return db
}

// createAuthzAndCasbin creates a fresh Authz engine and Casbin projection.
func createAuthzAndCasbin() (*CasbinProjection, error) {
	authz, err := NewAuthz()
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.authz.create", "create authz")
	}
	casbinProj, err := NewCasbinProjection(authz)
	if err != nil {
		return nil, errorfamily.WrapTransient(
			err,
			"usermgmt.authz.create_casbin_projection",
			"create casbin projection",
		)
	}
	return casbinProj, nil
}
