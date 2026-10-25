package setup

import (
	"context"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// validate checks the resolved config for common misconfigurations and returns
// a rejection error describing the first issue found, or nil if the config is sound.
func (c Config) validate() error {
	if err := c.validateDisableService(); err != nil {
		return err
	}

	if err := c.validateServiceSources(); err != nil {
		return err
	}

	if err := c.validateReadModelDialect(); err != nil {
		return err
	}

	if err := c.validateDisableAuth(); err != nil {
		return err
	}

	if err := c.validateAuthHandlerConfig(); err != nil {
		return err
	}

	if c.SSEMaxReplay < 0 {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"SSEMaxReplay must not be negative — 0 uses the transport default (1000), a positive value caps the backfill; got %d",
			c.SSEMaxReplay,
		)
	}

	if err := c.validatePathShapes(); err != nil {
		return err
	}

	if err := c.validatePathRoots(); err != nil {
		return err
	}

	if err := c.validateHealthChecks(); err != nil {
		return err
	}

	return requireDistinctPaths(c)
}

// validateServiceSources rejects configs that set more than one of the three
// service sources, or combine an explicit source (Service or ServiceConfig)
// with flattened service-construction fields. Silently ignoring them would be
// a footgun: a consumer setting TOTP alongside an adopted service would
// expect TOTP to be enabled and never learn it is not.
func (c Config) validateServiceSources() error {
	switch {
	case c.Service != nil && c.ServiceConfig != nil:
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"Service and ServiceConfig are mutually exclusive — Service adopts an existing *usermgmt.Service, ServiceConfig builds one; pick one",
		)
	case c.Service != nil:
		if conflicts := c.serviceConstructionConflicts(); len(conflicts) > 0 {
			return errorfamily.Newf(
				errorfamily.Rejection,
				"setup.invalid_config",
				"Service adopts an existing *usermgmt.Service, so service-construction fields are ignored — unset them: %s",
				strings.Join(conflicts, ", "),
			)
		}
	case c.ServiceConfig != nil:
		if conflicts := c.serviceConstructionConflicts(); len(conflicts) > 0 {
			return errorfamily.Newf(
				errorfamily.Rejection,
				"setup.invalid_config",
				"ServiceConfig fully specifies the service, so flattened service-construction fields are ignored — set them inside ServiceConfig instead: %s",
				strings.Join(conflicts, ", "),
			)
		}
	}

	return nil
}

// readModelDialectProbeTimeout bounds the dialect probe so a hung database
// cannot stall New indefinitely. The flattened path connects to ReadModelDB
// during New anyway (view-store auto-migration), so the probe adds no new
// connection behavior — only a bound on how long the first one may take.
const readModelDialectProbeTimeout = 5 * time.Second

// validateReadModelDialect rejects a ReadModelDB handle that does not speak
// SQLite. The flattened convenience path never sets ReadModelDialect, so
// usermgmt resolves "" to the SQLite read-model constructor family — against
// a Postgres or MySQL handle that family's DDL mostly parses and silently
// creates SQLite-schema tables, corrupting the deployment far from the
// misconfiguration. Failing at New with a pointer to the supported path
// (ServiceConfig.ReadModelDialect) is the honest alternative.
//
// Detection is a dialect probe, not a driver-name check: database/sql does
// not expose the registered driver name, so the validator asks the database
// itself. "SELECT sqlite_version()" answers successfully on every SQLite
// engine and fails on Postgres/MySQL. An unreachable database passes
// validation (it cannot be disproven) and fails later inside usermgmt with
// its own infrastructure error — a rejection here would lie about the cause.
//
// Must run after validateServiceSources: a non-nil ReadModelDB can only
// reach this check on the flattened path (the explicit service sources
// reject it as a conflict there).
func (c Config) validateReadModelDialect() error {
	if c.ReadModelDB == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), readModelDialectProbeTimeout)
	defer cancel()

	var version string
	probeErr := c.ReadModelDB.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	if probeErr == nil {
		return nil
	}

	// The probe failed for one of two reasons: the database is reachable and
	// does not speak SQLite (reject with the actionable pointer), or the
	// connection itself failed (pass — see the doc comment). Ping separates
	// the two cases.
	if err := c.ReadModelDB.PingContext(ctx); err != nil {
		return nil
	}

	return errorfamily.Newf(
		errorfamily.Rejection,
		"setup.invalid_config",
		"ReadModelDB does not speak SQLite (dialect probe failed: %s) — the flattened config path always builds SQLite-dialect read models; for this database use Config.ServiceConfig with ReadModelDB and ReadModelDialect set together (\"postgres\", \"pgx\", or \"mysql\")",
		probeErr,
	)
}

// validateAuthHandlerConfig rejects an AuthHandlerConfig whose CookieName
// disagrees with the flattened Config.CookieName. The auth handler must write
// the same cookie the session middleware reads — a mismatch silently breaks
// every authenticated request while the login flow still looks fine (the
// historical default-composition bug the SQL restart contract test guards).
// An empty CookieName inside AuthHandlerConfig is NOT a conflict: it inherits
// Config.CookieName at construction (see resolveAuthHandlerConfig).
//
// Must run after withDefaults (New applies defaults first), so an unset
// Config.CookieName has already become "session" by the time this compares.
func (c Config) validateAuthHandlerConfig() error {
	if c.AuthHandlerConfig == nil {
		return nil
	}

	if name := c.AuthHandlerConfig.CookieName; name != "" && name != c.CookieName {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"AuthHandlerConfig.CookieName %q does not match Config.CookieName %q — the auth handler must write the same cookie the session middleware reads; leave it empty to inherit",
			name,
			c.CookieName,
		)
	}

	return nil
}

// validateDisableService rejects shell-mode configs that still reference the
// usermgmt world. DisableService builds a bundle with no service (ADR-0054),
// so every field that describes, configures, or session-gates against one is
// a silent no-op or a latent panic — both are rejected instead. The two
// exceptions are EventStore/EventBus: in shell mode they are the bundle's
// OWN infrastructure inputs (defaults applied in New).
func (c Config) validateDisableService() error {
	if !c.DisableService {
		return nil
	}

	switch {
	case c.Service != nil:
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"DisableService builds a bundle without a usermgmt.Service, so Service must be nil — adopt with Service instead of DisableService, or drop Service for the shell",
		)
	case c.ServiceConfig != nil:
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"DisableService builds a bundle without a usermgmt.Service, so ServiceConfig must be nil — build one with ServiceConfig instead of DisableService, or drop ServiceConfig for the shell",
		)
	}

	var serviceFields []string
	for _, conflict := range c.serviceConstructionConflicts() {
		if conflict == "EventStore" || conflict == "EventBus" {
			continue // the shell's own inputs — consumed, not rejected
		}

		serviceFields = append(serviceFields, conflict)
	}

	if len(serviceFields) > 0 {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"DisableService builds a bundle without a usermgmt.Service, so service-construction fields are ignored — unset them: %s",
			strings.Join(serviceFields, ", "),
		)
	}

	if panels := c.disableServicePanelConflicts(); len(panels) > 0 {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"DisableService builds a bundle without a usermgmt.Service, so the panels cannot be built — set %s to true",
			strings.Join(panels, ", "),
		)
	}

	if gated := c.disableServiceGatedSurfaces(); len(gated) > 0 {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"DisableService builds a bundle whose session gate has no service to authenticate against (a cookie-carrying request would panic), so the session-gated surfaces cannot mount — unset %s; they return behind an injectable gate in a future release",
			strings.Join(gated, ", "),
		)
	}

	if c.AuthHandlerConfig != nil {
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"DisableService mounts no auth endpoints, so AuthHandlerConfig has nothing to configure — drop it",
		)
	}

	return nil
}

// disableServicePanelConflicts lists the panels a DisableService bundle
// cannot build (each needs the service for its data and mutations).
func (c Config) disableServicePanelConflicts() []string {
	var panels []string
	if !c.DisableAdmin {
		panels = append(panels, "DisableAdmin")
	}
	if !c.DisableDashboard {
		panels = append(panels, "DisableDashboard")
	}
	if !c.DisableLogin {
		panels = append(panels, "DisableLogin")
	}
	return panels
}

// disableServiceGatedSurfaces lists the session-gated surfaces whose gate
// would dereference the missing service — rejected at New so a half-mounted
// gated endpoint is never a latent panic on the first cookie.
func (c Config) disableServiceGatedSurfaces() []string {
	var gated []string
	for _, surface := range []struct{ name, path string }{
		{fieldSSEPath, c.SSEPath},
		{fieldDataStarPath, c.DataStarPath},
		{fieldEventCatalogPath, c.EventCatalogPath},
		{fieldProjectionStatusPath, c.ProjectionStatusPath},
		{fieldDebugPath, c.DebugPath},
		{fieldDocsPath, c.DocsPath},
	} {
		if surface.path != "" {
			gated = append(gated, surface.name)
		}
	}
	return gated
}

// validateDisableAuth rejects DisableAuth configs that keep surfaces which
// require the /auth/* routes. The login page's form posts to /auth/login —
// without the routes it renders fine and breaks on submit, the exact silent
// breakage class this validator family exists for (the cookie-name
// composition bug). AuthHandlerConfig without auth endpoints is a silent
// no-op the consumer would never discover.
func (c Config) validateDisableAuth() error {
	if !c.DisableAuth {
		return nil
	}

	if !c.DisableLogin {
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"DisableAuth mounts no /auth/* routes, so the login page's form would post into the void — set DisableLogin too, or drop DisableAuth",
		)
	}

	if c.AuthHandlerConfig != nil {
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"DisableAuth builds no auth handler, so AuthHandlerConfig has nothing to configure — drop it",
		)
	}

	return nil
}

// serviceConstructionConflicts lists the flattened service-construction fields
// that are set. Only meaningful when Service or ServiceConfig is also set.
func (c Config) serviceConstructionConflicts() []string {
	var conflicts []string

	for _, set := range []struct {
		name string
		set  bool
	}{
		{"EventStore", c.EventStore != nil},
		{"EventBus", c.EventBus != nil},
		{"ReadModelDB", c.ReadModelDB != nil},
		{"TOTP", c.TOTP != nil},
		{"WebAuthn", c.WebAuthn != nil},
		{"OAuth2", c.OAuth2 != nil},
		{"SessionTTL", c.SessionTTL != 0},
		{"Logger", c.Logger != nil},
		{"AsyncStartup", c.AsyncStartup},
		{"OnProjectionFailed", c.OnProjectionFailed != nil},
		{"Observability", c.Observability != nil},
	} {
		if set.set {
			conflicts = append(conflicts, set.name)
		}
	}

	return conflicts
}

// validateHealthChecks rejects consumer checks whose names collide with each
// other or with the built-ins ("projections", "sse-hub"). The readiness
// response keys results by name — a collision would silently drop one check's
// result, and a name squatting on a built-in would mask the real projection
// or hub state. Duplicate rejection is the fail-fast counterpart (mirrors the
// path-collision posture).
func (c Config) validateHealthChecks() error {
	const (
		builtinProjections = "projections"
		builtinHub         = "sse-hub"
	)

	seen := make(map[string]struct{}, len(c.HealthChecks)+2)
	seen[builtinProjections] = struct{}{}
	seen[builtinHub] = struct{}{}

	for _, check := range c.HealthChecks {
		if check.Name == "" {
			return errorfamily.NewRejection("setup.invalid_config",
				"HealthChecks entries must have a non-empty Name")
		}

		if check.Check == nil {
			return errorfamily.Newf(errorfamily.Rejection,
				"setup.invalid_config",
				"HealthChecks entry %q must have a non-nil Check", check.Name,
			)
		}

		if _, dup := seen[check.Name]; dup {
			return errorfamily.Newf(
				errorfamily.Rejection,
				"setup.invalid_config",
				"HealthChecks name %q collides with another check or a built-in (projections, sse-hub) — names must be unique",
				check.Name,
			)
		}

		seen[check.Name] = struct{}{}
	}

	return nil
}
