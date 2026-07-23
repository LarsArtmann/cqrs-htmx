package dashboardui

import (
	"fmt"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Dashboard is the CQRS/ES observability panel. Build it with [New] or
// [FromBundle], then register it on a router with [Dashboard.Mount] or
// [Dashboard.Handler].
//
// The dashboard reads from go-cqrs-lite introspection interfaces.
// Panels are conditionally active based on which interfaces the consumer
// provides. The dashboard auto-detects capabilities and shows only
// relevant panels.
type Dashboard struct {
	cfg  Config
	caps Capabilities
	nav  []navItem
}

// New builds a dashboard from cfg, applying defaults and validating
// the result. Returns an error only for invalid configuration (e.g. no
// event interfaces provided).
func New(cfg Config) (*Dashboard, error) {
	cfg, err := cfg.withDefaults()
	if err != nil {
		return nil, err
	}

	caps := cfg.capabilities()

	return &Dashboard{
		cfg:  cfg,
		caps: caps,
		nav:  buildNav(caps, cfg.BasePath),
	}, nil
}

// MustNew is like [New] but panics on error. For init-time setup.
func MustNew(cfg Config) *Dashboard {
	d, err := New(cfg)
	if err != nil {
		panic(fmt.Sprintf("dashboardui: %v", err))
	}

	return d
}

// page builds the pageData shell for a page, marking the nav item whose
// Href matches active.
func (d *Dashboard) page(title, active string, r *http.Request) pageData {
	nav := make([]navItem, len(d.nav))
	for i, n := range d.nav {
		n.Active = n.Href == active
		nav[i] = n
	}

	return pageData{
		Title:     title,
		BasePath:  d.cfg.BasePath,
		Accent:    d.cfg.AccentColor,
		Brand:     d.cfg.Title,
		Nav:       nav,
		LogoutURL: "",
		CSRFToken: csrfToken(r),
		CSRFMeta:  csrfMeta(r),
		ReadOnly:  d.cfg.ReadOnly,
		Caps:      d.caps,
	}
}

// guard wraps a handler with authorization. If Config.Authorizer is nil,
// all requests are allowed (the consumer must wrap with their own middleware).
func (d *Dashboard) guard(fn http.HandlerFunc) http.HandlerFunc {
	if d.cfg.Authorizer == nil {
		return fn
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if err := d.cfg.Authorizer(r); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)

			return
		}

		fn(w, r)
	}
}

// Capabilities returns which panels are active.
func (d *Dashboard) Capabilities() Capabilities { return d.caps }

// Config returns the resolved configuration (with defaults applied).
func (d *Dashboard) Config() Config { return d.cfg }

// errConfig constructs a configuration validation error.
func errConfig(msg string) error {
	return errorfamily.NewRejection("dashboardui.config", msg)
}
