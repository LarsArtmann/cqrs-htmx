// Command catalog-demo starts an HTTP server that serves live API documentation
// (OpenAPI, AsyncAPI, D2) generated from Go types via the cqrs-htmx catalog module.
//
// Run it and visit:
//
//	http://localhost:8080/openapi.json       — OpenAPI 3.0 spec
//	http://localhost:8080/asyncapi.json      — AsyncAPI 3.0 spec
//	http://localhost:8080/diagram.d2         — D2 architecture diagram
//	http://localhost:8080/health             — liveness probe
//	http://localhost:8080/openapi.yaml       — same spec as YAML (add ?format=yaml)
//
// It also writes an EventCatalog MDX file tree to ./eventcatalog on startup,
// demonstrating build-time doc generation.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/docserver"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
)

// --- Domain message types (these drive all schema generation) ---

// CreateOrderCommand is the HTTP request body for placing an order.
type CreateOrderCommand struct {
	CustomerID string      `json:"customer_id" doc:"Customer placing the order" example:"cust_123"`
	Items      []OrderItem `json:"items"       doc:"Line items in the order"`
	Currency   string      `json:"currency"    doc:"ISO 4217 currency code"     example:"USD"      enum:"USD,EUR,GBP"`
}

type OrderItem struct {
	SKU      string `json:"sku"      doc:"Product identifier" example:"sku_abc"`
	Quantity int    `json:"quantity" doc:"Units ordered"      example:"2"`
}

// OrderCreatedEvent is persisted to the event store when an order is placed.
//cqrs-lint:ignore(A011) all keys are lowercase snake_case; single-word keys like "total"/"items" are valid snake_case
type OrderCreatedEvent struct {
	OrderID    string      `json:"order_id"    doc:"The new order identifier" example:"ord_456"`
	CustomerID string      `json:"customer_id" doc:"Owning customer"          example:"cust_123"`
	Total      int         `json:"total"       doc:"Total in minor units"     example:"4999"`
	Items      []OrderItem `json:"items"       doc:"Ordered line items"`
}

// OrderCancelledEvent is published when an order is cancelled.
//cqrs-lint:ignore(A011) all keys are lowercase snake_case; single-word keys like "reason" are valid snake_case
type OrderCancelledEvent struct {
	OrderID string `json:"order_id" doc:"The cancelled order" example:"ord_456"`
	Reason  string `json:"reason"   doc:"Cancellation reason" example:"customer_request"`
}

// GetOrderQuery retrieves a single order by ID.
type GetOrderQuery struct {
	ID string `json:"id" doc:"Order identifier to look up" example:"ord_456"`
}

// buildCatalog wires up the documentation for the Order Service.
func buildCatalog() *catalog.Catalog {
	b := simple.New(
		"Order Service", "1.0.0",
		simple.WithServiceSummary("Example service demonstrating the catalog module"),
	)

	// Commands — describe the HTTP request shapes.
	simple.Command[CreateOrderCommand](
		b, "create-order",
		simple.WithOperation("POST", "/orders"),
		catalog.WithSummary("Place a new order"),
	)

	// Events — the persisted payloads are the real contract.
	simple.Event[OrderCreatedEvent](
		b, "order.created", catalog.Sends,
		catalog.WithSummary("An order was placed"),
	)
	simple.Event[OrderCancelledEvent](
		b, "order.cancelled", catalog.Sends,
		catalog.WithSummary("An order was cancelled"),
	)

	// Queries — read-side request shapes.
	simple.Query[GetOrderQuery](
		b, "get-order",
		simple.WithOperation("GET", "/orders/{id}"),
		catalog.WithSummary("Retrieve an order by ID"),
	)

	return b.Build()
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	eventCatalogDir := flag.String(
		"eventcatalog",
		"./eventcatalog",
		"directory for generated EventCatalog MDX files (empty to skip)",
	)
	flag.Parse()

	cat := buildCatalog()

	// Build-time artifact: generate the EventCatalog MDX file tree.
	if *eventCatalogDir != "" {
		if err := docserver.GenerateEventCatalog(cat, *eventCatalogDir); err != nil {
			log.Fatalf("generate event catalog: %v", err)
		}
		log.Printf("wrote EventCatalog files to %s", *eventCatalogDir)
	}

	// DocsServer serves OpenAPI/AsyncAPI specs (JSON) + HTML UIs.
	ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
		ServiceName: "Order Service",
		Version:     "1.0.0",
	})

	mux := http.NewServeMux()
	mux.Handle("/openapi.json", ds.OpenAPISpec())
	mux.Handle("/asyncapi.json", ds.AsyncAPISpec())
	mux.Handle("/diagram.d2", docserver.D2Handler(cat))
	mux.Handle("/health", docserver.HealthCheckHandler(cat))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		log.Printf("catalog-demo serving API docs on http://localhost%s", *addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	//cqrs-lint:ignore(E010) example: minimal signal handling for demo
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Print("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
