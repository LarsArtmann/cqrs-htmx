// Package main demonstrates setup.New with AsyncStartup: the HTTP server
// binds immediately while projections replay the journal in the background.
// /health answers 503 (not ready) until every projection worker is live,
// then flips to 200 — the zero-downtime restart pattern from
// docs/guides/async-projection-startup.md.
//
// Run: go run . then poll http://localhost:8098/health
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

func main() {
	bundle, err := setup.New(setup.Config{
		Title:        "Async Startup Demo",
		AsyncStartup: true,
	})
	if err != nil {
		log.Fatalf("setup.New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.Printf("listening on :8098 — /health returns 503 while projections drain, then 200")
	if err := bundle.Run(ctx, ":8098"); err != nil {
		log.Fatal(err)
	}
}
