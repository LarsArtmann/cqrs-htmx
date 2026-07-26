package dashboardui

import (
	"fmt"
	"net/http"
	"sync"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
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
	cfg         Config
	caps        Capabilities
	nav         []navItem
	broadcaster *cqrshtmx.Broadcaster
	sseStore    cqrshtmx.SSEEventStore
	done        chan struct{}
	closeOnce   sync.Once
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

	d := &Dashboard{
		cfg:  cfg,
		caps: caps,
		nav:  buildNav(caps, cfg.BasePath),
		done: make(chan struct{}),
	}

	if caps.EventBus {
		d.broadcaster = cqrshtmx.NewBroadcaster()
		d.startEventBridge()

		// Build SSE replay store from the configured journal. Enables reconnect
		// replay (Last-Event-ID) and initial backfill of recent events.
		if journal := cfg.journalForReplay(); journal != nil {
			d.sseStore = cqrshtmx.NewJournalSSEStore(journal, newSSEEvent)
		}
	}

	return d, nil
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

// Close releases dashboard resources. Safe to call multiple times.
// Signals the event-bus handler to stop, then closes the SSE broadcaster,
// which disconnects all connected SSE clients.
// Call this during application shutdown.
func (d *Dashboard) Close() {
	d.closeOnce.Do(func() {
		if d.done != nil {
			close(d.done)
		}
		if d.broadcaster != nil {
			d.broadcaster.Close()
		}
	})
}

// errConfig constructs a configuration validation error.
func errConfig(msg string) error {
	return errorfamily.NewRejection("dashboardui.config", msg)
}
