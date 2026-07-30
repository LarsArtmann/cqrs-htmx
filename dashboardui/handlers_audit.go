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

	var cmds []*command.PersistedCommand

	if d.cfg.CommandJournal != nil {
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
			renderError(w, r, http.StatusInternalServerError, "failed to load commands")
			return
		}
	}

	html := d.renderCommands(p, cmds)
	renderPage(w, r, html)
}

func (d *Dashboard) renderCommands(p pageData, cmds []*command.PersistedCommand) string {
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

		return fmt.Sprintf(`<h3>Command Audit</h3><table class="data-table"><thead><tr><th scope="col">Received At</th><th scope="col">Type</th><th scope="col">Stream Type</th><th scope="col">Stream ID</th><th scope="col">Command ID</th></tr></thead><tbody>%s</tbody></table>`, rows.String())
	})
}

//nolint:dupl // parallel cmd/query
func (d *Dashboard) queriesIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Queries", "/queries", r)

	var queries []*query.PersistedQuery

	if d.cfg.QueryJournal != nil {
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
			renderError(w, r, http.StatusInternalServerError, "failed to load queries")
			return
		}
	}

	html := d.renderQueries(p, queries)
	renderPage(w, r, html)
}

func (d *Dashboard) renderQueries(p pageData, queries []*query.PersistedQuery) string {
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

		return fmt.Sprintf(`<h3>Query Audit</h3><table class="data-table"><thead><tr><th scope="col">Received At</th><th scope="col">Type</th><th scope="col">Request ID</th></tr></thead><tbody>%s</tbody></table>`, rows.String())
	})
}
