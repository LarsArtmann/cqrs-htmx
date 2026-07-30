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

//nolint:dupl // parallel cmd/query
func (d *Dashboard) commandsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Commands", "/commands", r)

	pageSize := parsePageSize(r, d.cfg.PageSize)
	after := id.CommandID(r.URL.Query().Get("after"))
	hasPrev := r.URL.Query().Get("after") != ""

	var cmds []*command.PersistedCommand
	var hasNext bool

	if d.cfg.CommandJournal != nil {
		var err error

		if seekable, ok := d.cfg.CommandJournal.(command.SeekableCommandJournal); ok {
			cmds, err = seekable.ReadFrom(r.Context(), after, pageSize+1)
		} else {
			cmds, err = d.cfg.CommandJournal.ReadAll(r.Context())
			if err == nil && len(cmds) > pageSize {
				cmds = cmds[:pageSize]
			}
		}

		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "failed to load commands")
			return
		}

		hasNext = len(cmds) > pageSize
		if hasNext {
			cmds = cmds[:pageSize]
		}
	}

	var nextCursor string
	if hasNext && len(cmds) > 0 {
		nextCursor = cmds[len(cmds)-1].ID().String()
	}

	html := d.renderCommands(p, cmds, paginationState{HasNext: hasNext, NextCursor: nextCursor, PageSize: pageSize, HasPrev: hasPrev})
	renderPage(w, r, html)
}

func (d *Dashboard) renderCommands(p pageData, cmds []*command.PersistedCommand, pg paginationState) string {
	return d.renderLayout(p, func() string {
		if len(cmds) == 0 {
			return emptyState("No commands recorded", "Commands will appear here as they are dispatched.")
		}

		var rows strings.Builder

		for _, cmd := range cmds {
			fmt.Fprintf(&rows, `<tr><td class="mono">%s</td><td><code>%s</code></td><td>%s</td><td class="mono">%s</td><td class="mono">%s</td></tr>`,
				esc(cmd.ReceivedAt().Format("2006-01-02 15:04:05")),
				esc(string(cmd.Type())),
				esc(string(cmd.StreamType())),
				esc(cmd.StreamID().String()),
				truncate(cmd.ID().String(), eventIDWidth))
		}

		var b strings.Builder
		fmt.Fprintf(&b, `<h3>Command Audit</h3><table class="data-table"><thead><tr><th scope="col">Received At</th><th scope="col">Type</th><th scope="col">Stream Type</th><th scope="col">Stream ID</th><th scope="col">Command ID</th></tr></thead><tbody>%s</tbody></table>`, rows.String())
		b.WriteString(renderPagination(p.BasePath, "/commands", pg, ""))

		return b.String()
	})
}

//nolint:dupl // parallel cmd/query
func (d *Dashboard) queriesIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Queries", "/queries", r)

	pageSize := parsePageSize(r, d.cfg.PageSize)
	after := id.RequestID(r.URL.Query().Get("after"))
	hasPrev := r.URL.Query().Get("after") != ""

	var queries []*query.PersistedQuery
	var hasNext bool

	if d.cfg.QueryJournal != nil {
		var err error

		if seekable, ok := d.cfg.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, err = seekable.ReadQueriesFrom(r.Context(), after, pageSize+1)
		} else {
			queries, err = d.cfg.QueryJournal.ReadAllQueries(r.Context())
			if err == nil && len(queries) > d.cfg.PageSize {
				queries = queries[:d.cfg.PageSize]
			}
		}

		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "failed to load queries")
			return
		}

		hasNext = len(queries) > pageSize
		if hasNext {
			queries = queries[:pageSize]
		}
	}

	var nextCursor string
	if hasNext && len(queries) > 0 {
		nextCursor = queries[len(queries)-1].ID().String()
	}

	html := d.renderQueries(p, queries, paginationState{HasNext: hasNext, NextCursor: nextCursor, PageSize: pageSize, HasPrev: hasPrev})
	renderPage(w, r, html)
}

func (d *Dashboard) renderQueries(p pageData, queries []*query.PersistedQuery, pg paginationState) string {
	return d.renderLayout(p, func() string {
		if len(queries) == 0 {
			return emptyState("No queries recorded", "Queries will appear here as they are executed.")
		}

		var rows strings.Builder

		for _, q := range queries {
			fmt.Fprintf(&rows, `<tr><td class="mono">%s</td><td><code>%s</code></td><td class="mono">%s</td></tr>`,
				esc(q.ReceivedAt().Format("2006-01-02 15:04:05")),
				esc(string(q.Type())),
				truncate(q.ID().String(), eventIDWidth))
		}

		var b strings.Builder
		fmt.Fprintf(&b, `<h3>Query Audit</h3><table class="data-table"><thead><tr><th scope="col">Received At</th><th scope="col">Type</th><th scope="col">Request ID</th></tr></thead><tbody>%s</tbody></table>`, rows.String())
		b.WriteString(renderPagination(p.BasePath, "/queries", pg, ""))

		return b.String()
	})
}
