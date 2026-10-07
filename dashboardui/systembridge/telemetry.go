// Package systembridge maps a live go-cqrs-lite system onto the dashboard's
// optional capability and telemetry seams. It is the only dashboardui package
// importing go-cqrs-lite's system and metaengine packages — consumers that
// never import it carry none of that dependency weight.
package systembridge

import (
	"context"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// WireTelemetry maps a system's read-only introspection surface onto the
// dashboard's telemetry providers: the topology snapshot feeds the topology
// table, HealthCheckDetailed feeds the healthz/readyz engine composition,
// and the metaengine's plan + engine stats feed the placement table and
// latency cards. Combine with [dashboardui.FromSystem]:
//
//	cfg := dashboardui.FromSystem(sys)
//	systembridge.WireTelemetry(&cfg, sys)
//	dash := dashboardui.MustNew(cfg)
//
// This package is the one place dashboardui touches the system package —
// consumers who never import it carry no system/metaengine dependency.
// Calling it twice replaces the providers; a nil MetaEngine simply leaves
// the placements/stats providers emitting empty views.
func WireTelemetry(cfg *dashboardui.Config, sys *system.System) {
	cfg.Topology = func(ctx context.Context) (core.TopologyView, error) {
		topology, err := sys.Snapshot(ctx)
		if err != nil {
			return core.TopologyView{}, errorfamily.WrapInfrastructure(
				err, "systemadapter.topology_snapshot", "system topology snapshot failed",
			)
		}

		return topologyViewFrom(topology), nil
	}

	cfg.EngineHealths = func(ctx context.Context) []core.EngineHealthView {
		return engineHealthViews(sys.HealthCheckDetailed(ctx))
	}

	cfg.Placements = func(context.Context) ([]core.PlacementView, error) {
		return placementViews(sys), nil
	}

	cfg.EngineStats = func(ctx context.Context) []core.EngineStatsView {
		return engineStatsViews(sys, ctx)
	}
}

func topologyViewFrom(topology *system.Topology) core.TopologyView {
	var projectionHost *core.ProjectionHostView
	if topology.ProjectionHost != nil {
		projectionHost = &core.ProjectionHostView{
			Started: topology.ProjectionHost.Started,
			Workers: topology.ProjectionHost.Workers,
		}
	}

	view := core.TopologyView{
		Instances:      make([]core.InstanceView, 0, len(topology.Instances)),
		Buses:          make([]core.BusView, 0, len(topology.Buses)),
		ProjectionHost: projectionHost,
	}

	for _, instance := range topology.Instances {
		view.Instances = append(view.Instances, core.InstanceView{
			Name:         instance.Name,
			Role:         string(instance.Role),
			EngineName:   instance.EngineName,
			DriverName:   instance.DriverName,
			Durability:   string(instance.Durability),
			HealthStatus: instance.HealthStatus,
			Collections:  instance.Collections,
		})
	}

	for _, bus := range topology.Buses {
		view.Buses = append(view.Buses, core.BusView{
			Name:   bus.Name,
			Driver: bus.Driver,
			Mode:   bus.Mode,
		})
	}

	return view
}

func engineHealthViews(healths []system.EngineHealth) []core.EngineHealthView {
	views := make([]core.EngineHealthView, 0, len(healths))

	for _, health := range healths {
		var engineErr string
		if health.Error != nil {
			engineErr = health.Error.Error()
		}

		views = append(views, core.EngineHealthView{
			Name:    health.Name,
			Healthy: health.Error == nil,
			Error:   engineErr,
		})
	}

	return views
}

func placementViews(sys *system.System) []core.PlacementView {
	store := sys.MetaEngine()
	if store == nil {
		return nil
	}

	plan := store.Plan()
	if plan == nil {
		return nil
	}

	views := make([]core.PlacementView, 0, len(plan.Queries))

	for _, assignment := range plan.Queries {
		views = append(views, core.PlacementView{
			Query:        assignment.QueryName,
			Engine:       assignment.EngineName,
			ADT:          string(assignment.ADT),
			Volume:       assignment.Cost.Volume,
			EstLatencyMs: assignment.Cost.EstimatedLatencyMs,
		})
	}

	return views
}

func engineStatsViews(sys *system.System, ctx context.Context) []core.EngineStatsView {
	store := sys.MetaEngine()
	if store == nil {
		return nil
	}

	stats := store.GetEngineStats(ctx)
	views := make([]core.EngineStatsView, 0, len(stats))

	for _, stat := range stats {
		views = append(views, core.EngineStatsView{
			Name:       stat.Name,
			HasRTT:     stat.HasLiveRTT,
			Samples:    stat.MeasuredRTT.Samples,
			EWMA:       stat.MeasuredRTT.EWMA,
			P95:        stat.MeasuredRTT.P95,
			LastSample: stat.MeasuredRTT.LastSample,
		})
	}

	return views
}
