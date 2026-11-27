// Package main is a runnable demo of cqrs-htmx client tracking on the setup
// bundle. The bundle serves the embedded tracker at /-/tracker.js, stores
// every page view, click and impression in its engine-owned signals journal,
// and /signals lists what was recorded, for everyone or for one subject.
//
// What to look for:
//
//   - Every page carries one script tag; nothing else is needed to collect.
//   - data-track names a click, data-track-impression names an element that
//     became visible. Element text and attribute values are never recorded.
//   - /dev-login signs the demo shopper in: their signals then carry the
//     user key next to the device key.
//   - /signals reads the journal back (All); ?user=<id> and ?device=<id>
//     read one subject's stream (ForUser, ForClient).
//
// Run: GOEXPERIMENT=jsonv2 go run . and open http://localhost:8094
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// defaultAddr avoids the ports the other examples document.
const defaultAddr = ":8094"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// demo is the composed application: the setup bundle (tracker script,
// endpoint, signals journal, session) plus the pages of the shop.
type demo struct {
	bundle  *setup.Bundle
	mux     *http.ServeMux
	shopper usermgmt.User
	token   string // the demo shopper's session token
}

// newDemo builds the composition main() serves.
func newDemo(ctx context.Context) (*demo, error) {
	// One call: the bundle mounts the tracker script and the endpoint and
	// stores signals in the journal over the given engine. An app whose
	// event store is the one go-cqrs-lite's system builds passes nothing
	// here: the journal then shares that store's engine.
	bundle, err := setup.New(setup.Config{ //nolint:exhaustruct // a demo takes the in-memory defaults
		Title:           "cqrs-htmx Tracking Demo",
		TrackingBackend: streamLog(metaengine.NewMemoryEngine()),
		// The demo owns its pages and its sign-in shortcut.
		DisableLogin:     true,
		DisableAdmin:     true,
		DisableDashboard: true,
	})
	if err != nil {
		return nil, fmt.Errorf("setup.New: %w", err)
	}

	shopper, err := bundle.Service.Register(ctx, usermgmt.RegisterRequest{
		ID: usermgmt.SyntheticUserID(shopperID), Email: shopperEmail, DisplayName: "Demo Shopper",
	})
	if err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("register the demo shopper: %w", err)
	}

	app := &demo{bundle: bundle, mux: http.NewServeMux(), shopper: *shopper.User, token: shopper.Session.Token}
	app.mux.HandleFunc("GET /{$}", app.page("Shop", shopBody))
	app.mux.HandleFunc("GET /checkout", app.page("Checkout", checkoutBody))
	app.mux.HandleFunc("GET /dev-login", app.devLogin)
	app.mux.HandleFunc("GET /signals", app.signals)

	return app, nil
}

// streamLog narrows an engine to the interface the signals journal writes
// through. metaengine.Engine does not name its stream-log methods, so the
// narrowing is the consumer's one line.
func streamLog(engine metaengine.Engine) metaengine.StreamLogBackend {
	backend, ok := engine.(metaengine.StreamLogBackend)
	if !ok {
		panic("tracking-demo: the engine cannot store streams")
	}

	return backend
}

func run() error {
	addr := envOr("TRACKING_DEMO_ADDR", defaultAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := newDemo(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("tracking-demo\nOpen http://localhost%s (the shop) and http://localhost%s/signals (what was recorded)\n", addr, addr)
	fmt.Println("Sign in as the demo shopper at /dev-login: their signals then carry the user key next to the device key.")

	return app.bundle.RunHandler(ctx, addr, app.bundle.Handler(app.mux))
}

// envOr reads an environment variable, or its fallback.
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
