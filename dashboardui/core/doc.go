// Package core contains the pure data layer for the CQRS dashboard.
//
// It provides typed data-fetching functions, capability detection,
// pagination math, and payload rendering — all with zero HTML generation
// and zero rendering dependencies. Any consumer (templ app, CLI tool,
// metrics exporter) can import core to inspect a go-cqrs-lite system
// without pulling in the dashboard's rendering layer.
//
// The standalone [Dashboard] in the parent package is built on top of core:
// it fetches data via core functions and renders it via fmt.Fprintf-based
// handlers. Future consumers can use core directly with their own rendering.
package core
