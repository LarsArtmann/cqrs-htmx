package setup

import (
	"log/slog"
	"net/http"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TrackingJournal returns the signals journal the bundle stores client
// signals in. It is nil when signals go to a [Config.TrackingSink] or when
// tracking has no durable home (see [Config.TrackingBackend]); the read
// methods of a nil journal return an error instead of panicking.
func (b *Bundle) TrackingJournal() *SignalsJournal {
	return b.tracking
}

// attachTracking resolves where signals are stored and builds what the
// bundle mounts: the endpoint that feeds it, wrapped so a signal is
// attributed to the session the server saw. A config that names no durable
// home mounts nothing and says why once at debug level: tracking is opt-in,
// so a bundle that was never offered a home has nothing to warn about, and
// an app that mounted the tracker anyway sees the script answer 404.
func (b *Bundle) attachTracking() {
	sink, journal := b.config.resolveSignalsHome(b.Stores.EventStore)

	b.tracking = journal

	if sink == nil {
		if b.config.TrackingPath != "-" {
			b.trackingLogger().Debug("setup: client tracking is off",
				"tracking_path", b.config.TrackingPath,
				"reason", "no durable home for signals: the event store is not the one go-cqrs-lite's system builds",
				"fix", "set Config.TrackingBackend (any engine stream-log backend), or Config.TrackingSink, or pass system.EventStore() as EventStore",
			)
		}

		return
	}

	if b.config.TrackingPath == "-" {
		return
	}

	b.trackingHandler = trackEnrichment(b)(cqrshtmx.TrackHandler(sink))
}

// trackingLogger is the bundle's configured logger, or the process default.
func (b *Bundle) trackingLogger() *slog.Logger {
	if b.config.Logger != nil {
		return b.config.Logger
	}

	return slog.Default()
}

// resolveSignalsHome names where signals are stored, in precedence order:
// the consumer's sink, the consumer's backend, the system event adapter's
// backend. No sink means tracking stays off; the journal is set exactly when
// the bundle built it (validateTracking already refused the two-homes case).
func (c Config) resolveSignalsHome(store event.Store) (cqrshtmx.TrackingSink, *SignalsJournal) {
	switch {
	case c.TrackingSink != nil:
		return c.TrackingSink, nil
	case c.TrackingBackend != nil:
		journal := newSignalsJournal(c.TrackingBackend)

		return journal, journal
	}

	if adapter, ok := store.(*system.EventAdapter); ok && adapter.Backend != nil {
		journal := newSignalsJournal(adapter.Backend)

		return journal, journal
	}

	return nil, nil
}

// trackEnrichment attributes a signal to the visitor the server saw: the
// session middleware reads the session cookie into the context, and the
// enrichment copies the user, the device key (X-Client-Id) and a request ID
// from there. The session middleware only enriches, it never blocks, so an
// anonymous visitor's beacon passes exactly like a signed-in one's.
func trackEnrichment(b *Bundle) func(http.Handler) http.Handler {
	enrich := cqrshtmx.ContextEnrichmentMiddleware(sessionUserID)

	if b.Service == nil {
		return enrich
	}

	session := usermgmt.NewSessionMiddleware(b.Service, b.config.CookieName)

	return func(next http.Handler) http.Handler { return session(enrich(next)) }
}

// sessionUserID names the user of an authenticated request, and nobody for
// the rest.
func sessionUserID(r *http.Request) (cqrshtmx.UserID, error) {
	user, _ := usermgmt.UserFromContext(r.Context())
	if user == nil {
		return cqrshtmx.UserID{}, nil
	}

	return user.ID, nil
}

// mountTracking registers the tracker script and the endpoint. Both are
// public: the script is a static file, and the endpoint proves first-party
// origin itself (Sec-Fetch-Site, Origin) and caps every field, so it needs
// neither a session nor a CSRF token.
func (b *Bundle) mountTracking(mux *http.ServeMux) {
	if b.trackingHandler == nil {
		return
	}

	if path := b.config.TrackerScriptPath; path != "" && path != "-" {
		mux.Handle(path, cqrshtmx.TrackerScriptHandler())
	}

	mux.Handle(b.config.TrackingPath, b.trackingHandler)
}

// trackingPostures manifests the tracking endpoint and the tracker script,
// in the order Mount registers them.
func (b *Bundle) trackingPostures() []cqrshtmx.RoutePosture {
	if b.trackingHandler == nil {
		return nil
	}

	rows := []cqrshtmx.RoutePosture{
		{
			Method:  http.MethodPost,
			Pattern: b.config.TrackingPath,
			Posture: cqrshtmx.PosturePublic,
			Why:     "collects signals from anyone: first-party origin is proved by Sec-Fetch-Site/Origin, every field is capped, and a session cookie only attributes",
		},
	}

	if path := b.config.TrackerScriptPath; path != "" && path != "-" {
		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  http.MethodGet,
			Pattern: path,
			Posture: cqrshtmx.PosturePublic,
			Why:     "static tracker script, no data",
		})
	}

	return rows
}
