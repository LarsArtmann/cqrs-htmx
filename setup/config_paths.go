package setup

import (
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)


// Config-field display names used across the validation tables. They name
// the field (not the route), so a rejected config names the exact key to
// change; constants keep the three tables in sync (goconst).
const (
	fieldSSEPath              = "SSEPath"
	fieldDataStarPath         = "DataStarPath"
	fieldEventCatalogPath     = "EventCatalogPath"
	fieldProjectionStatusPath = "ProjectionStatusPath"
	fieldDebugPath            = "DebugPath"
	fieldDocsPath             = "DocsPath"
	fieldLivePath             = "LivePath"
	fieldSSEScriptPath        = "SSEScriptPath"
	fieldDataStarScriptPath   = "DataStarScriptPath"
)

// normalizeRoutePaths applies the route-shape rules every mount path shares:
// panel subtrees gain a trailing slash (the standard mux needs it for subtree
// matching), exact-match endpoints lose theirs (no ugly /health → /health/
// redirect), and the two SDK-script mounts default beside their feed when the
// feed is configured. "-" opt-outs pass through untouched (single chars are
// never trimmed).
func normalizeRoutePaths(cfg Config) Config {
	// The standard mux only treats patterns ending in "/" as subtree patterns:
	// without the slash, "/manage" matches exactly "/manage" and every panel
	// sub-route 404s. Normalize so consumers can pass either form.
	cfg.AdminPath = ensureTrailingSlash(cfg.AdminPath)
	cfg.DashboardPath = ensureTrailingSlash(cfg.DashboardPath)

	// Exact-match endpoints, unlike the panel subtrees.
	for _, path := range []*string{
		&cfg.HealthPath,
		&cfg.LivePath,
		&cfg.EventCatalogPath,
		&cfg.ProjectionStatusPath,
		&cfg.DebugPath,
		&cfg.SSEPath,
		&cfg.SSEScriptPath,
		&cfg.DataStarPath,
		&cfg.DataStarScriptPath,
	} {
		*path = trimTrailingSlash(*path)
	}

	if cfg.SSEPath != "" && cfg.SSEScriptPath == "" {
		cfg.SSEScriptPath = "/sse.js"
	}

	if cfg.DataStarPath != "" && cfg.DataStarScriptPath == "" {
		cfg.DataStarScriptPath = "/datastar.js"
	}

	return cfg
}

// ensureTrailingSlash appends "/" unless the path already ends with one.
func ensureTrailingSlash(s string) string {
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}

	return s
}

func trimTrailingSlash(s string) string {
	if len(s) > 1 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}

	return s
}


// validatePathShapes rejects paths that do not start with a slash (or, for
// LoginRedirect, a URL scheme).
func (c Config) validatePathShapes() error {
	if !startsWithSlash(c.AdminPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "AdminPath must start with %q (got %q)", "/", c.AdminPath)
	}

	if !startsWithSlash(c.DashboardPath) {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"DashboardPath must start with %q (got %q)",
			"/",
			c.DashboardPath,
		)
	}

	if c.HealthPath != "-" && !startsWithSlash(c.HealthPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config",
			"HealthPath must start with %q or be \"-\" to disable (got %q)", "/", c.HealthPath)
	}

	if c.LivePath != "" && !startsWithSlash(c.LivePath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "LivePath must start with %q (got %q)", "/", c.LivePath)
	}

	if err := c.validateOptionalFeedPaths(); err != nil {
		return err
	}

	if !startsWithSlash(c.LoginRedirect) && !startsWithScheme(c.LoginRedirect) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config",
			"LoginRedirect must start with %q or a URL scheme (got %q)", "/", c.LoginRedirect)
	}

	if c.CookieName == "" {
		return errorfamily.NewRejection("setup.invalid_config", "CookieName must not be empty")
	}

	return nil
}

// validateOptionalFeedPaths checks the optional SSE/DataStar/machine mount
// paths — empty means "feature disabled", which is always valid, and the two
// script mounts additionally accept "-" (serve the feed but not the script).
func (c Config) validateOptionalFeedPaths() error {
	for _, opt := range []struct {
		name, path string
		allowDash  bool
	}{
		{fieldEventCatalogPath, c.EventCatalogPath, false},
		{fieldProjectionStatusPath, c.ProjectionStatusPath, false},
		{fieldDebugPath, c.DebugPath, false},
		{fieldLivePath, c.LivePath, false},
		{fieldSSEPath, c.SSEPath, false},
		{fieldSSEScriptPath, c.SSEScriptPath, true},
		{fieldDataStarPath, c.DataStarPath, false},
		{fieldDataStarScriptPath, c.DataStarScriptPath, true},
		{fieldDocsPath, c.DocsPath, false},
	} {
		if opt.path == "" || (opt.allowDash && opt.path == "-") {
			continue
		}

		if !startsWithSlash(opt.path) {
			return errorfamily.Newf(errorfamily.Rejection,
				"setup.invalid_config", "%s must start with %q (got %q)", opt.name, "/", opt.path)
		}
	}

	if (c.DocsPath == "") != (c.DocsHandler == nil) {
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"DocsPath and DocsHandler must be set together — mount both or neither",
		)
	}

	return nil
}

// validatePathRoots rejects mounts on the site root: the login page owns "/"
// as its catch-all, so any panel or health endpoint there would collide at
// Mount time.
func (c Config) validatePathRoots() error {
	for _, p := range []struct{ name, path string }{
		{"AdminPath", c.AdminPath},
		{"DashboardPath", c.DashboardPath},
		{"HealthPath", c.HealthPath},
		{fieldSSEPath, c.SSEPath},
		{fieldSSEScriptPath, c.SSEScriptPath},
		{fieldDataStarPath, c.DataStarPath},
		{fieldDataStarScriptPath, c.DataStarScriptPath},
		{fieldLivePath, c.LivePath},
		{fieldEventCatalogPath, c.EventCatalogPath},
		{fieldProjectionStatusPath, c.ProjectionStatusPath},
		{fieldDebugPath, c.DebugPath},
		{fieldDocsPath, c.DocsPath},
	} {
		if p.path == "/" {
			return errorfamily.Newf(errorfamily.Rejection,
				"setup.invalid_config",
				"%s must not be %q — the site root is reserved for the login page", p.name, "/")
		}
	}

	return nil
}

// requireDistinctPaths rejects configs where two mount paths resolve to the
// same route after normalization. Equal paths would make http.ServeMux panic
// inside Mount ("conflicts with pattern"); rejecting here surfaces the
// misconfiguration at New, not at first request. Overlapping-but-distinct
// paths ("/app/" vs "/app/admin/") are fine: the mux resolves them by
// longest prefix.
func requireDistinctPaths(c Config) error {
	paths := []struct{ name, path string }{
		{"AdminPath", trimTrailingSlash(c.AdminPath)},
		{"DashboardPath", trimTrailingSlash(c.DashboardPath)},
		{"HealthPath", trimTrailingSlash(c.HealthPath)},
		{fieldSSEPath, trimTrailingSlash(c.SSEPath)},
		{fieldSSEScriptPath, trimTrailingSlash(c.SSEScriptPath)},
		{fieldDataStarPath, trimTrailingSlash(c.DataStarPath)},
		{fieldDataStarScriptPath, trimTrailingSlash(c.DataStarScriptPath)},
		{fieldLivePath, trimTrailingSlash(c.LivePath)},
		{fieldEventCatalogPath, trimTrailingSlash(c.EventCatalogPath)},
		{fieldProjectionStatusPath, trimTrailingSlash(c.ProjectionStatusPath)},
		{fieldDebugPath, trimTrailingSlash(c.DebugPath)},
		{fieldDocsPath, trimTrailingSlash(c.DocsPath)},
	}

	for i := range paths {
		for j := i + 1; j < len(paths); j++ {
			// Unset optional paths ("" and the "-" opt-out) never conflict
			// with each other — two disabled features are not a route collision.
			if paths[i].path == "" || paths[j].path == "" || paths[i].path == "-" || paths[j].path == "-" {
				continue
			}

			if paths[i].path == paths[j].path {
				return errorfamily.Newf(errorfamily.Rejection,
					"setup.invalid_config",
					"%s and %s both resolve to %q — routes would conflict",
					paths[i].name, paths[j].name, paths[i].path)
			}
		}
	}

	return nil
}

func startsWithSlash(s string) bool {
	return len(s) > 0 && s[0] == '/'
}

func startsWithScheme(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
