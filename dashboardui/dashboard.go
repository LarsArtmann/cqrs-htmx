package dashboardui

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/transport"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-sse"
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
	config      Config
	caps        Capabilities
	nav         []navItem
	broadcaster *cqrshtmx.Broadcaster
	sseStore    sse.EventStore
	done        chan struct{}
	closeOnce   sync.Once
}

// New builds a dashboard from config, applying defaults and validating
// the result. Returns an error only for invalid configuration (e.g. no
// event interfaces provided).
func New(config Config) (*Dashboard, error) {
	config, err := config.withDefaults()
	if err != nil {
		return nil, err
	}

	caps := config.capabilities()

	if !config.ReadOnly && config.Authorizer == nil {
		slog.Warn("dashboardui: write operations are enabled (ReadOnly=false) but no Authorizer is configured; " +
			"anyone with network access can reset projections, replay/delete dead letters, and delete snapshots. " +
			"Set Config.Authorizer or wrap the dashboard with authentication middleware.")
	}

	d := &Dashboard{
		config: config,
		caps:   caps,
		nav:    buildNav(caps),
		done:   make(chan struct{}),
	}

	if caps.EventBus {
		d.broadcaster = cqrshtmx.NewBroadcaster()
		d.startEventBridge()

		// Build SSE replay store from the configured journal. Enables reconnect
		// replay (Last-Event-ID) and initial backfill of recent events.
		if journal := config.journalForReplay(); journal != nil {
			d.sseStore = cqrshtmx.NewJournalSSEStore(journal, transport.DomainEventToSSE)
		}
	}

	return d, nil
}

// MustNew is like [New] but panics on error. For init-time setup.
func MustNew(config Config) *Dashboard {
	d, err := New(config)
	if err != nil {
		panic(
			fmt.Sprintf("dashboardui: %v", err),
		)
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
		BasePath:  d.config.BasePath,
		Accent:    d.config.AccentColor,
		Brand:     d.config.Title,
		Nav:       nav,
		LogoutURL: d.config.LogoutURL,
		CSRFToken: csrfToken(r),
		ReadOnly:  d.config.ReadOnly,
		Caps:      d.caps,
		HTMX:      isHTMXRequest(r),
	}
}

// guard wraps a handler with authorization. If Config.Authorizer is nil,
// all requests are allowed (the consumer must wrap with their own middleware).
func (d *Dashboard) guard(fn http.HandlerFunc) http.HandlerFunc {
	if d.config.Authorizer == nil {
		return fn
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if err := d.config.Authorizer(r); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)

			return
		}

		fn(w, r)
	}
}

// Capabilities returns which panels are active.
func (d *Dashboard) Capabilities() Capabilities { return d.caps }

// Config returns the resolved configuration (with defaults applied).
func (d *Dashboard) Config() Config { return d.config }

// Close releases dashboard resources. Safe to call multiple times and on a
// nil dashboard (a no-op), so shutdown hooks can run even if construction
// failed partway.
// Signals the event-bus handler to stop, then closes the SSE broadcaster,
// which disconnects all connected SSE clients.
// Call this during application shutdown.
func (d *Dashboard) Close() {
	if d == nil {
		return
	}

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
