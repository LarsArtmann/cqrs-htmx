package core

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// Status kinds drive the color-coding of projection health in the UI.
const (
	StatusGood    = "good"
	StatusWarn    = "warn"
	StatusBad     = "bad"
	StatusNeutral = "neutral"
)

const (
	statusRunning = "running"
	statusFailed  = "failed"

	RecentEventsLimit  = 5
	OverviewCountLimit = 500
)

// ProjectionStat is the display representation of a projection worker's
// status, suitable for rendering in any UI (table, card, JSON API).
type ProjectionStat struct {
	Name       string
	Status     string
	Lag        string
	Processed  int64
	Errors     int64
	StatusKind string
	Restarts   int
	Checkpoint string
	LastError  string
}

// RecentEvent is a display DTO for the overview's recent events list.
type RecentEvent struct {
	Time       string    `json:"time"`
	Type       string    `json:"type"`
	StreamID   string    `json:"streamId"`
	StreamType string    `json:"streamType"`
	Version    string    `json:"version"`
	EventID    string    `json:"eventId"`
	OccurredAt time.Time `json:"-"`
}

// Overview aggregates the top-level dashboard stats: event count, aggregate
// count, projection health, DLQ count, and recent events.
type Overview struct {
	TotalAggregates string
	TotalEvents     string
	Projections     []ProjectionStat
	DLQCount        string
	RecentEvents    []RecentEvent
	HealthStatus    string
	HealthKind      string
}

// ProjectionStatusKind maps a raw projection status string to a semantic
// status kind (StatusGood, StatusWarn, StatusBad, StatusNeutral).
func ProjectionStatusKind(status string) string {
	switch strings.ToLower(status) {
	case statusRunning, "live":
		return StatusGood
	case "idle", "backoff", "draining":
		return StatusWarn
	case "stopped", statusFailed:
		return StatusBad
	default:
		return StatusNeutral
	}
}

// ProjectionStats converts the projection host's WorkerState slice into
// ProjectionStat entries with status classification, lag, and additional
// fields (restarts, checkpoint, last error).
func ProjectionStats(host *projectionhost.Host) []ProjectionStat {
	if host == nil {
		return nil
	}

	lagPerProj := host.LagPerProjection()

	var stats []ProjectionStat

	for _, ws := range host.Status() {
		lag := lagPerProj[ws.Name]
		stats = append(stats, ProjectionStat{
			Name:       ws.Name,
			Status:     string(ws.Status),
			Lag:        lag.String(),
			Processed:  ws.Processed,
			Errors:     ws.Errors,
			StatusKind: ProjectionStatusKind(string(ws.Status)),
			Restarts:   ws.Restarts,
			Checkpoint: ws.Checkpoint,
			LastError:  ws.LastError,
		})
	}

	return stats
}

// FetchOverview aggregates events count, aggregates count, projection health,
// DLQ count, and recent events into a single Overview struct.
func FetchOverview(ctx context.Context, cfg Config) Overview { //nolint:cyclop,gocognit // multi-source aggregation
	stats := Overview{
		TotalAggregates: "0",
		TotalEvents:     "0",
	}

	if cfg.StreamReader != nil {
		page, err := cfg.StreamReader.List(ctx, listing.ListOptions{Limit: uint(cfg.PageSize)})
		if err == nil && page != nil {
			stats.TotalAggregates = strconv.Itoa(len(page.Items))
			if page.HasMore {
				stats.TotalAggregates += "+"
			}
		}
	}

	if cfg.SeekableJournal != nil { //nolint:nestif // optional data source branching
		events, err := cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, OverviewCountLimit)
		if err == nil {
			for i, evt := range events {
				if i >= RecentEventsLimit {
					break
				}

				stats.RecentEvents = append(stats.RecentEvents, RecentEvent{
					Time:       evt.OccurredAt().Format(time.RFC3339),
					Type:       string(evt.Type()),
					StreamID:   evt.StreamID().String(),
					StreamType: string(evt.StreamType()),
					Version:    evt.Version().String(),
					EventID:    evt.ID().String(),
					OccurredAt: evt.OccurredAt(),
				})
			}

			stats.TotalEvents = strconv.Itoa(len(events))
			if len(events) >= OverviewCountLimit {
				stats.TotalEvents += "+"
			}
		}
	} else if cfg.Journal != nil {
		events, err := cfg.Journal.ReadAll(ctx)
		if err == nil {
			stats.TotalEvents = strconv.Itoa(len(events))
			for i, evt := range events {
				if i >= RecentEventsLimit {
					break
				}

				stats.RecentEvents = append(stats.RecentEvents, RecentEvent{
					Time:       evt.OccurredAt().Format(time.RFC3339),
					Type:       string(evt.Type()),
					StreamID:   evt.StreamID().String(),
					StreamType: string(evt.StreamType()),
					Version:    evt.Version().String(),
					EventID:    evt.ID().String(),
					OccurredAt: evt.OccurredAt(),
				})
			}
		}
	}

	if cfg.ProjectionHost != nil {
		stats.Projections = ProjectionStats(cfg.ProjectionHost)

		totalErrors := int64(0)
		anyBad := false
		anyWarn := false

		for _, pr := range stats.Projections {
			totalErrors += pr.Errors
			switch pr.StatusKind {
			case StatusBad:
				anyBad = true
			case StatusWarn:
				anyWarn = true
			}
		}

		if totalErrors > 0 {
			stats.DLQCount = strconv.FormatInt(totalErrors, 10)
		}

		switch {
		case anyBad:
			stats.HealthStatus = "Unhealthy"
			stats.HealthKind = StatusBad
		case anyWarn:
			stats.HealthStatus = "Degraded"
			stats.HealthKind = StatusWarn
		case len(stats.Projections) > 0:
			stats.HealthStatus = "Healthy"
			stats.HealthKind = StatusGood
		}
	}

	return stats
}
