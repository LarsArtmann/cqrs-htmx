package dashboardui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

const exportLimit = 10000

const (
	formatCSVParam  = "csv"
	formatJSONParam = "json"
	formatParam     = "format"
)

type responseFormat int

const (
	formatHTML responseFormat = iota
	formatCSV
	formatJSON
)

func parseFormat(r *http.Request) responseFormat {
	switch r.URL.Query().Get(formatParam) {
	case formatCSVParam:
		return formatCSV
	case formatJSONParam:
		return formatJSON
	default:
		return formatHTML
	}
}

func writeCSV(w http.ResponseWriter, filename string, headers []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)
	_ = writer.Write(headers)

	for _, row := range rows {
		_ = writer.Write(row)
	}

	writer.Flush()
}

func writeJSONResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	_ = encoder.Encode(data) //nolint:errchkjson // dynamic export data
}

// ===== Events =====

type eventExportRow struct {
	EventID    string `json:"eventId"`
	Type       string `json:"type"`
	StreamType string `json:"streamType"`
	StreamID   string `json:"streamId"`
	Version    string `json:"version"`
	OccurredAt string `json:"occurredAt"`
}

func exportEventsCSV(w http.ResponseWriter, events []event.Event) {
	headers := []string{"Event ID", "Type", "Stream Type", "Stream ID", "Version", "Occurred At"}
	rows := make([][]string, len(events))

	for i, evt := range events {
		rows[i] = []string{
			evt.ID().String(),
			string(evt.Type()),
			string(evt.StreamType()),
			evt.StreamID().String(),
			evt.Version().String(),
			evt.OccurredAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	writeCSV(w, "events.csv", headers, rows)
}

func exportEventsJSON(w http.ResponseWriter, events []event.Event) {
	rows := make([]eventExportRow, len(events))

	for i, evt := range events {
		rows[i] = eventExportRow{
			EventID:    evt.ID().String(),
			Type:       string(evt.Type()),
			StreamType: string(evt.StreamType()),
			StreamID:   evt.StreamID().String(),
			Version:    evt.Version().String(),
			OccurredAt: evt.OccurredAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	writeJSONResponse(w, rows)
}

// ===== Commands =====

type commandExportRow struct {
	CommandID  string `json:"commandId"`
	Type       string `json:"type"`
	StreamType string `json:"streamType"`
	StreamID   string `json:"streamId"`
	ReceivedAt string `json:"receivedAt"`
}

func exportCommandsCSV(w http.ResponseWriter, cmds []*command.PersistedCommand) {
	rows := make([][]string, len(cmds))

	for i, cmd := range cmds {
		rows[i] = []string{
			cmd.ID().String(),
			string(cmd.Type()),
			string(cmd.StreamType()),
			cmd.StreamID().String(),
			cmd.ReceivedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	writeCSV(w, "commands.csv", []string{"Command ID", "Type", "Stream Type", "Stream ID", "Received At"}, rows)
}

func exportCommandsJSON(w http.ResponseWriter, cmds []*command.PersistedCommand) {
	rows := make([]commandExportRow, len(cmds))

	for i, cmd := range cmds {
		rows[i] = commandExportRow{
			CommandID:  cmd.ID().String(),
			Type:       string(cmd.Type()),
			StreamType: string(cmd.StreamType()),
			StreamID:   cmd.StreamID().String(),
			ReceivedAt: cmd.ReceivedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	writeJSONResponse(w, rows)
}

// ===== Queries =====

type queryExportRow struct {
	RequestID  string `json:"requestId"`
	Type       string `json:"type"`
	ReceivedAt string `json:"receivedAt"`
}

func exportQueriesCSV(w http.ResponseWriter, queries []*query.PersistedQuery) {
	rows := make([][]string, len(queries))

	for i, q := range queries {
		rows[i] = []string{
			q.ID().String(),
			string(q.Type()),
			q.ReceivedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	writeCSV(w, "queries.csv", []string{"Request ID", "Type", "Received At"}, rows)
}

func exportQueriesJSON(w http.ResponseWriter, queries []*query.PersistedQuery) {
	rows := make([]queryExportRow, len(queries))

	for i, q := range queries {
		rows[i] = queryExportRow{
			RequestID:  q.ID().String(),
			Type:       string(q.Type()),
			ReceivedAt: q.ReceivedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	writeJSONResponse(w, rows)
}

// formatLinks renders CSV and JSON export links for the given page path.
func formatLinks(basePath, path string) string {
	return fmt.Sprintf(
		`<div class="filter-bar"><span class="muted">Export:</span><a href="%s%s?format=csv" class="btn">CSV</a><a href="%s%s?format=json" class="btn">JSON</a></div>`,
		basePath,
		path,
		basePath,
		path,
	)
}
