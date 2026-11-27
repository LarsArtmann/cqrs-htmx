package main

// Client-tracking surfaces for the Playwright tracker specs (T18 of the
// native query log plan): the REAL endpoint, the REAL embedded tracker, and
// the REAL session attribution, over a test-only in-memory sink the specs
// read back. Nothing here is a re-implementation: the page is the smallest
// host that carries the three hooks the tracker honors (the script tag, a
// data-track element, a data-track-impression element).

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

const (
	// trackingCookie is the session cookie of the tracking stack's own
	// usermgmt service (separate from the WebAuthn stack's "session_token",
	// so the two stacks never read each other's cookies).
	trackingCookie = "e2e_tracking_session"

	// trackingUser names the one user the dev-login endpoint signs in.
	trackingUser = "e2e-tracker"
)

// trackingSession is the one session the dev-login endpoint hands out. It is
// built on first use and reused: the battery shares one server across
// projects and reuses a running one locally, and registering the same user
// twice is rejected.
var trackingSession = struct {
	once   sync.Once
	userID string
	token  string
	err    error
}{}

// signalStore is the test-only sink: every validated signal the endpoint
// recorded, in arrival order.
type signalStore struct {
	mu      sync.Mutex
	signals []cqrshtmx.TrackedEvent
}

var tracked = &signalStore{}

func (s *signalStore) Record(_ context.Context, events []cqrshtmx.TrackedEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.signals = append(s.signals, events...)

	return nil
}

func (s *signalStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.signals = nil
}

func (s *signalStore) all() []cqrshtmx.TrackedEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]cqrshtmx.TrackedEvent(nil), s.signals...)
}

// mountTracking wires the tracking stack onto mux: the tracker script, the
// endpoint behind the session middleware (so a signed-in visitor's signals
// carry the user key), the page under test, and the debug endpoints the
// specs drive and read.
func mountTracking(mux *http.ServeMux) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{}) //nolint:exhaustruct_v5 // e2e defaults
	if err != nil {
		log.Fatalf("tracking usermgmt service: %v", err)
	}

	enrich := cqrshtmx.ContextEnrichmentMiddleware(func(r *http.Request) (cqrshtmx.UserID, error) {
		user, _ := usermgmt.UserFromContext(r.Context())
		if user == nil {
			return cqrshtmx.UserID{}, nil
		}

		return user.ID, nil
	})

	session := usermgmt.NewSessionMiddleware(svc, trackingCookie)

	mux.Handle("GET /-/tracker.js", cqrshtmx.TrackerScriptHandler())
	mux.Handle("POST /-/track", session(enrich(cqrshtmx.TrackHandler(tracked))))

	mux.HandleFunc("GET /tracking", trackingPage(false))
	mux.HandleFunc("GET /tracking/off", trackingPage(true))

	// TEST-ONLY: signs the tracking user in and reports the user key the
	// spec expects on every signal. Same never-ship-this rule as the other
	// debug endpoints.
	mux.HandleFunc("POST /api/debug/tracking-login", func(w http.ResponseWriter, r *http.Request) {
		trackingSession.once.Do(func() {
			registered, err := svc.Register(r.Context(), usermgmt.RegisterRequest{
				ID: usermgmt.SyntheticUserID(trackingUser), Email: trackingUser + "@e2e.test",
			})
			if err != nil {
				trackingSession.err = err

				return
			}

			trackingSession.userID = registered.User.ID.String()
			trackingSession.token = registered.Session.Token
		})

		if trackingSession.err != nil {
			http.Error(w, trackingSession.err.Error(), http.StatusInternalServerError)

			return
		}

		http.SetCookie(w, &http.Cookie{ //nolint:exhaustruct // e2e cookie
			Name: trackingCookie, Value: trackingSession.token, Path: "/",
		})

		cqrshtmx.WriteJSON(w, http.StatusOK, map[string]string{"userId": trackingSession.userID})
	})

	mux.HandleFunc("GET /api/debug/signals", func(w http.ResponseWriter, _ *http.Request) {
		cqrshtmx.WriteJSON(w, http.StatusOK, tracked.all())
	})

	mux.HandleFunc("POST /api/debug/signals/reset", func(w http.ResponseWriter, _ *http.Request) {
		tracked.reset()
		w.WriteHeader(http.StatusNoContent)
	})
}

// trackingPage is the smallest host page for the tracker. off sets the
// page-level opt-out the tracker honors on <html data-tracking="off">.
func trackingPage(off bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		attr := ""
		if off {
			attr = ` data-tracking="off"`
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html lang="en"` + attr + `>
<head><meta charset="utf-8"><title>Tracking E2E</title></head>
<body>
	<h1>Tracking E2E</h1>
	<button id="buy" data-track="buy">Buy</button>
	<div id="promo" data-track-impression="promo">Promotion</div>
	<script src="/-/tracker.js" defer></script>
</body>
</html>
`))
	}
}
