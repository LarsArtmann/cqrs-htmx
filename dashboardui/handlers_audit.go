package dashboardui

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// ===== Command/Query Audit =====

func (d *Dashboard) commandsIndexHandler(w http.ResponseWriter, r *http.Request) { //nolint:dupl,nestif // parallel command/query structure; types differ so cannot share
	p := d.page("Commands", "/commands", r)

	var cmds []*command.PersistedCommand

	if d.cfg.CommandJournal != nil { //nolint:nestif // journal-fallback branching is inherently nested
		var err error

		if seekable, ok := d.cfg.CommandJournal.(command.SeekableCommandJournal); ok {
			cmds, err = seekable.ReadFrom(r.Context(), id.CommandID{}, d.cfg.PageSize)
		} else {
			cmds, err = d.cfg.CommandJournal.ReadAll(r.Context())
			if err == nil && len(cmds) > d.cfg.PageSize {
				cmds = cmds[:d.cfg.PageSize]
			}
		}

		if err != nil {
			http.Error(w, "failed to load commands: "+err.Error(), http.StatusInternalServerError)

			return
		}
	}

	html := d.renderCommands(p, cmds)
	renderPage(w, r, html)
}

func (d *Dashboard) renderCommands(p pageData, cmds []*command.PersistedCommand) string {
	return d.renderLayout(p, func() string {
		if len(cmds) == 0 {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No commands recorded</h3><p>Commands will appear here as they are dispatched.</p></div>`
		}

		var rows strings.Builder

		for _, cmd := range cmds {
			fmt.Fprintf(
				&rows, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px"><code>%s</code></td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.8em;color:var(--muted)">%s</td>
			</tr>`,
				esc(cmd.ReceivedAt().Format("2006-01-02 15:04:05")),
				esc(string(cmd.Type())),
				esc(string(cmd.StreamType())),
				esc(cmd.StreamID().String()),
				truncate(cmd.ID().String(), eventIDWidth),
			)
		}

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Command Audit</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">Received At</th>
					<th style="padding:8px">Type</th>
					<th style="padding:8px">Stream Type</th>
					<th style="padding:8px">Stream ID</th>
					<th style="padding:8px">Command ID</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows.String())
	})
}

func (d *Dashboard) queriesIndexHandler(w http.ResponseWriter, r *http.Request) { //nolint:dupl,nestif // parallel command/query structure; types differ so cannot share
	p := d.page("Queries", "/queries", r)

	var queries []*query.PersistedQuery

	if d.cfg.QueryJournal != nil { //nolint:nestif // journal-fallback branching is inherently nested
		var err error

		if seekable, ok := d.cfg.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, err = seekable.ReadQueriesFrom(r.Context(), id.RequestID{}, d.cfg.PageSize)
		} else {
			queries, err = d.cfg.QueryJournal.ReadAllQueries(r.Context())
			if err == nil && len(queries) > d.cfg.PageSize {
				queries = queries[:d.cfg.PageSize]
			}
		}

		if err != nil {
			http.Error(w, "failed to load queries: "+err.Error(), http.StatusInternalServerError)

			return
		}
	}

	html := d.renderQueries(p, queries)
	renderPage(w, r, html)
}

func (d *Dashboard) renderQueries(p pageData, queries []*query.PersistedQuery) string {
	return d.renderLayout(p, func() string {
		if len(queries) == 0 {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No queries recorded</h3><p>Queries will appear here as they are executed.</p></div>`
		}

		var rows strings.Builder

		for _, q := range queries {
			fmt.Fprintf(
				&rows, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px"><code>%s</code></td>
				<td style="padding:8px;font-family:monospace;font-size:0.8em;color:var(--muted)">%s</td>
			</tr>`,
				esc(q.ReceivedAt().Format("2006-01-02 15:04:05")),
				esc(string(q.Type())),
				truncate(q.ID().String(), eventIDWidth),
			)
		}

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Query Audit</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">Received At</th>
					<th style="padding:8px">Type</th>
					<th style="padding:8px">Request ID</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows.String())
	})
}
