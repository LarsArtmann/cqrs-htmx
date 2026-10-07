package core

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
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

	// RecentEventsLimit is the number of recent events shown on the overview.
	RecentEventsLimit = 5
	// OverviewCountLimit is the scan cap for event/aggregate totals.
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

// HubHealth is a display DTO for the SSE hub (broadcaster) state on the
// overview page. Nil on Overview means the dashboard has no SSE hub (no
// EventBus configured), so no hub card renders.
type HubHealth struct {
	Subscribers int
	BufferSize  int
	Closed      bool
	Draining    bool
}

// Overview aggregates the top-level dashboard stats: event count, aggregate
// count, projection health, DLQ count, recent events, and SSE hub state.
type Overview struct {
	TotalAggregates string
	TotalEvents     string
	Projections     []ProjectionStat
	DLQCount        string
	RecentEvents    []RecentEvent
	HealthStatus    string
	HealthKind      string
	SSEHub          *HubHealth
}

// ProjectionStatusKind maps a raw projection status string to a semantic
// status kind (StatusGood, StatusWarn, StatusBad, StatusNeutral).
// "stopped" is a healthy terminal state: journal-only hosts drain to
// "stopped" once fully caught up, and the root library's readiness gate
// (ProjectionReadinessCheck) treats live/stopped as ready. Only "failed"
// marks an unhealthy worker.
func ProjectionStatusKind(status string) string {
	switch strings.ToLower(status) {
	case statusRunning, "live", "stopped":
		return StatusGood
	case "idle", "backoff", "draining":
		return StatusWarn
	case statusFailed:
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
func FetchOverview(
	ctx context.Context,
	cfg Config,
) Overview {
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

	if cfg.SeekableJournal != nil {
		events, err := cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, OverviewCountLimit)
		if err == nil {
			stats.RecentEvents = recentEventsFrom(events)

			stats.TotalEvents = strconv.Itoa(len(events))
			if len(events) >= OverviewCountLimit {
				stats.TotalEvents += "+"
			}
		}
	} else if cfg.Journal != nil {
		events, err := cfg.Journal.ReadAll(ctx)
		if err == nil {
			stats.TotalEvents = strconv.Itoa(len(events))
			stats.RecentEvents = recentEventsFrom(events)
		}
	}

	if cfg.ProjectionHost != nil {
		stats.Projections = ProjectionStats(cfg.ProjectionHost)
		stats.DLQCount, stats.HealthStatus, stats.HealthKind = classifyProjectionHealth(stats.Projections)
	}

	return stats
}

// recentEventsFrom projects at most RecentEventsLimit committed events into
// the display shape used by the overview card.
func recentEventsFrom(events []event.Event) []RecentEvent {
	recent := make([]RecentEvent, 0, min(len(events), RecentEventsLimit))
	for i, evt := range events {
		if i >= RecentEventsLimit {
			break
		}

		recent = append(recent, RecentEvent{
			Time:       evt.OccurredAt().Format(time.RFC3339),
			Type:       string(evt.Type()),
			StreamID:   evt.StreamID().String(),
			StreamType: string(evt.StreamType()),
			Version:    evt.Version().String(),
			EventID:    evt.ID().String(),
			OccurredAt: evt.OccurredAt(),
		})
	}

	return recent
}

// classifyProjectionHealth sums DLQ counts and derives the overall health
// word and kind from per-projection status kinds.
func classifyProjectionHealth(projs []ProjectionStat) (string, string, string) {
	totalErrors := int64(0)
	anyBad := false
	anyWarn := false

	for _, pr := range projs {
		totalErrors += pr.Errors

		switch pr.StatusKind {
		case StatusBad:
			anyBad = true
		case StatusWarn:
			anyWarn = true
		}
	}

	var dlqCount string

	if totalErrors > 0 {
		dlqCount = strconv.FormatInt(totalErrors, 10)
	}

	switch {
	case anyBad:
		return dlqCount, "Unhealthy", StatusBad
	case anyWarn:
		return dlqCount, "Degraded", StatusWarn
	case len(projs) > 0:
		return dlqCount, "Healthy", StatusGood
	default:
		return "", "", ""
	}
}
