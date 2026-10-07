package core

import (
	"context"
	"time"
)

// Telemetry views: provider seams + plain data the dashboard renders
// read-only telemetry panels from. The views are deliberately dashboard-owned
// (not go-cqrs-lite system/metaengine types) so the rendering layer stays
// leaf-typed — an adapter maps a *system.System onto these.

// TopologyProvider renders the system topology panel: instances, buses, and
// the projection host shape.
type TopologyProvider func(ctx context.Context) (TopologyView, error)

// EngineHealthProvider feeds the healthz/readyz composition with per-engine
// health. Healthy=false engines fail readyz.
type EngineHealthProvider func(ctx context.Context) []EngineHealthView

// PlacementsProvider renders the query-placement table: which engine and ADT
// the planner assigned to each query, at what volume and latency estimate.
type PlacementsProvider func(ctx context.Context) ([]PlacementView, error)

// EngineStatsProvider renders the engine stat cards: live RTT EWMA/percentiles
// and sample counts per engine.
type EngineStatsProvider func(ctx context.Context) []EngineStatsView

// TopologyView is the read-only system shape: who holds what, connected how.
type TopologyView struct {
	Instances      []InstanceView
	Buses          []BusView
	ProjectionHost *ProjectionHostView
}

// InstanceView is one deployed instance (role + engine + health).
type InstanceView struct {
	Name         string
	Role         string
	EngineName   string
	DriverName   string
	Durability   string
	HealthStatus string
	Collections  []string
}

// BusView is one event bus in the topology.
type BusView struct {
	Name   string
	Driver string
	Mode   string
}

// ProjectionHostView summarizes the projection host. Nil means the topology
// carries no host.
type ProjectionHostView struct {
	Started bool
	Workers int
}

// EngineHealthView is one engine's health for probe composition.
type EngineHealthView struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitzero"`
}

// PlacementView is one row of the query-placement table.
type PlacementView struct {
	Query        string
	Engine       string
	ADT          string
	Volume       int64
	EstLatencyMs float64
}

// EngineStatsView is one engine's live latency snapshot. Zero Samples with
// HasRTT means the tracker is installed but has not observed traffic yet.
type EngineStatsView struct {
	Name       string
	HasRTT     bool
	Samples    int
	EWMA       time.Duration
	P95        time.Duration
	LastSample time.Time
}

// Stale reports whether the last RTT sample is older than the threshold —
// a quiet engine is not necessarily a dead one, but an OLD sample is worth
// flagging next to the numbers.
func (v EngineStatsView) Stale(threshold time.Duration) bool {
	return v.LastSample.IsZero() || time.Since(v.LastSample) > threshold
}
