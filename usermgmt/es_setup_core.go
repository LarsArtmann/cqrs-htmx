package usermgmt

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// eventSourcedSetupCore holds the infrastructure shared by every SQL-backed
// event-sourced setup (Postgres, SQLite): the four aggregate repositories,
// the four read-model projections, the connection bundle, the live DB handle,
// and the Casbin + projection-host machinery.
//
// The concrete setup structs (PostgresEventSourcedSetup,
// SQLiteEventSourcedSetup) embed this core and only differ in how the bundle
// is constructed and which backendName is reported in error codes. Keeping the
// shared state + lifecycle methods here means the (build-tag-gated) setup
// files stay thin and the shared logic is compiled and verified even though
// the setup files themselves carry a `//go:build ignore` guard.
type eventSourcedSetupCore struct {
	UserRepository       *decider.Repository[UserState]
	MembershipRepository *decider.Repository[MembershipState]
	TenantRepository     *decider.Repository[TenantState]
	BotRepository        *decider.Repository[BotState]
	ReadModel            projection.Projection
	MembershipReadModel  projection.Projection
	TenantReadModel      projection.Projection
	BotReadModel         projection.Projection
	Bundle               *stack.Bundle
	DB                   *sql.DB

	// backendName is the short label ("postgres", "sqlite") used to build
	// stable error codes such as "usermgmt.postgres_setup.close".
	backendName string

	casbinProjection *CasbinProjection
	projectionHost   *projectionhost.Host
}

// Close stops the projection host and closes the bundle. It is the shared
// implementation backing the SQL-backed setups' Close method.
func (c *eventSourcedSetupCore) Close() error {
	stopProjections(c.projectionHost, c.backendName)
	if c.Bundle != nil {
		if err := c.Bundle.Close(); err != nil {
			return errorfamily.WrapTransient(
				err,
				"usermgmt."+c.backendName+"_setup.close",
				"close "+c.backendName+" bundle",
			)
		}
	}
	return nil
}

// GracefulClose is the context-aware variant of Close.
func (c *eventSourcedSetupCore) GracefulClose(ctx context.Context) error {
	stopProjections(c.projectionHost, c.backendName)
	if c.Bundle != nil {
		if err := c.Bundle.GracefulClose(ctx); err != nil {
			return errorfamily.WrapTransient(
				err,
				"usermgmt."+c.backendName+"_setup.graceful_close",
				"graceful close "+c.backendName+" bundle",
			)
		}
	}
	return nil
}

// Authz returns the live Casbin enforcer populated by the projection.
func (c *eventSourcedSetupCore) Authz() *Authz { return c.casbinProjection.authz }

// stopProjections stops the host (if any), logging the wrapped error so the
// backend close path still runs. Shared by Close and GracefulClose.
func stopProjections(host *projectionhost.Host, backendName string) {
	if host == nil {
		return
	}
	if err := host.Stop(); err != nil {
		slog.Warn(
			"usermgmt setup: failed to stop projection host during close",
			slog.String("backend", backendName),
			slog.String("error", err.Error()),
		)
		_ = errorfamily.WrapTransient(
			err,
			"usermgmt."+backendName+"_setup.stop_projections",
			"stop projection host",
		)
	}
}
