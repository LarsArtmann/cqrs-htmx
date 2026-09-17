package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/icons"
)

// ===== Command/Query Audit =====

//
//nolint:cyclop // export branching adds complexity
func (d *Dashboard) commandsIndexHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	p := d.page("Commands", "/commands", r)

	if fmt := parseFormat(r); fmt != formatHTML {
		var cmds []*command.PersistedCommand

		if seekable, ok := d.config.CommandJournal.(command.SeekableCommandJournal); ok {
			cmds, _ = seekable.ReadFrom(r.Context(), id.CommandID{}, exportLimit)
		} else if d.config.CommandJournal != nil {
			cmds, _ = d.config.CommandJournal.ReadAll(r.Context())
		}

		switch fmt {
		case formatCSV:
			exportCommandsCSV(w, cmds)
		case formatJSON:
			exportCommandsJSON(w, cmds)
		case formatHTML:
			// handled below
		}

		return
	}

	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)
	after, _ := id.ParseCommandID(afterCursor)

	var (
		cmds    []*command.PersistedCommand
		hasNext bool
	)

	if d.config.CommandJournal != nil { //nolint:nestif // optional data source branching
		var err error

		if seekable, ok := d.config.CommandJournal.(command.SeekableCommandJournal); ok {
			cmds, err = seekable.ReadFrom(r.Context(), after, pageSize+1)
		} else {
			cmds, err = d.config.CommandJournal.ReadAll(r.Context())
			if err == nil && len(cmds) > pageSize {
				cmds = cmds[:pageSize]
			}
		}

		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to load commands")

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

	html := d.renderCommands(
		r.Context(),
		p,
		cmds,
		paginationState{
			HasNext: hasNext, NextCursor: nextCursor, PageSize: pageSize, HasPrev: hasPrev,
			After: afterCursor, PrevHistory: prevHistory,
		}.WithCountInfo(len(cmds)),
	)
	renderPage(w, r, html)
}

func (d *Dashboard) renderCommands(
	ctx context.Context,
	p pageData,
	cmds []*command.PersistedCommand,
	page paginationState,
) string {
	return d.renderLayout(ctx, p, func() string {
		if len(cmds) == 0 {
			return emptyStateIcon(
				ctx,
				icons.Clipboard,
				"No commands recorded",
				"Commands will appear here as they are dispatched.",
			)
		}

		headers := plainHeaders("Received At", "Type", "Stream Type", "Stream ID", "Command ID", "")

		rows := make([]display.TableRow, 0, len(cmds))

		for _, cmd := range cmds {
			streamCell := fmt.Sprintf(
				`<span class="mono">%s</span> %s`,
				esc(truncate(cmd.StreamID().String(), listIDWidth)),
				copyButtonHTML(ctx, cmd.StreamID().String(), ""),
			)
			idCell := fmt.Sprintf(
				`<span class="mono">%s</span> %s`,
				esc(truncate(cmd.ID().String(), eventIDWidth)),
				copyButtonHTML(ctx, cmd.ID().String(), ""),
			)
			typeCell := fmt.Sprintf(`<code>%s</code>`, esc(string(cmd.Type())))
			viewCell := buttonLink(
				ctx,
				"View",
				p.BasePath+"/commands/"+esc(cmd.ID().String()),
				"",
				display.ButtonSecondary,
				false,
			)

			rows = append(rows, display.TableRow{
				Cells: []display.TableCell{
					textCell(cmd.ReceivedAt().Format("2006-01-02 15:04:05")),
					rawCell(typeCell),
					textCell(string(cmd.StreamType())),
					rawCell(streamCell),
					rawCell(idCell),
					rawCell(viewCell),
				},
				Href: "",
			})
		}

		var b strings.Builder
		b.WriteString(`<h2>Command Audit</h2>`)
		b.WriteString(tableHTML(ctx, headers, rows, ""))
		b.WriteString(renderPagination(ctx, p.BasePath, "/commands", page, ""))
		b.WriteString(formatLinks(ctx, p.BasePath, "/commands"))

		return b.String()
	})
}

//
//nolint:cyclop // export branching adds complexity
func (d *Dashboard) queriesIndexHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	p := d.page("Queries", "/queries", r)

	if fmt := parseFormat(r); fmt != formatHTML {
		var queries []*query.PersistedQuery

		if seekable, ok := d.config.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, _ = seekable.ReadQueriesFrom(r.Context(), id.RequestID{}, exportLimit)
		} else if d.config.QueryJournal != nil {
			queries, _ = d.config.QueryJournal.ReadAllQueries(r.Context())
		}

		switch fmt {
		case formatCSV:
			exportQueriesCSV(w, queries)
		case formatJSON:
			exportQueriesJSON(w, queries)
		case formatHTML:
			// handled below
		}

		return
	}

	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)
	after, _ := id.ParseRequestID(afterCursor)

	var (
		queries []*query.PersistedQuery
		hasNext bool
	)

	if d.config.QueryJournal != nil { //nolint:nestif // optional data source branching
		var err error

		if seekable, ok := d.config.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, err = seekable.ReadQueriesFrom(r.Context(), after, pageSize+1)
		} else {
			queries, err = d.config.QueryJournal.ReadAllQueries(r.Context())
			if err == nil && len(queries) > d.config.PageSize {
				queries = queries[:d.config.PageSize]
			}
		}

		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to load queries")

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

	html := d.renderQueries(
		r.Context(),
		p,
		queries,
		paginationState{
			HasNext: hasNext, NextCursor: nextCursor, PageSize: pageSize, HasPrev: hasPrev,
			After: afterCursor, PrevHistory: prevHistory,
		}.WithCountInfo(len(queries)),
	)
	renderPage(w, r, html)
}

func (d *Dashboard) renderQueries(
	ctx context.Context,
	p pageData,
	queries []*query.PersistedQuery,
	page paginationState,
) string {
	return d.renderLayout(ctx, p, func() string {
		if len(queries) == 0 {
			return emptyStateIcon(
				ctx,
				icons.Search,
				"No queries recorded",
				"Queries will appear here as they are executed.",
			)
		}

		headers := plainHeaders("Received At", "Type", "Request ID", "")

		rows := make([]display.TableRow, 0, len(queries))

		for _, q := range queries {
			idCell := fmt.Sprintf(
				`<span class="mono">%s</span> %s`,
				esc(truncate(q.ID().String(), eventIDWidth)),
				copyButtonHTML(ctx, q.ID().String(), ""),
			)
			typeCell := fmt.Sprintf(`<code>%s</code>`, esc(string(q.Type())))
			viewCell := buttonLink(
				ctx,
				"View",
				p.BasePath+"/queries/"+esc(q.ID().String()),
				"",
				display.ButtonSecondary,
				false,
			)

			rows = append(rows, display.TableRow{
				Cells: []display.TableCell{
					textCell(q.ReceivedAt().Format("2006-01-02 15:04:05")),
					rawCell(typeCell),
					rawCell(idCell),
					rawCell(viewCell),
				},
				Href: "",
			})
		}

		var b strings.Builder
		b.WriteString(`<h2>Query Audit</h2>`)
		b.WriteString(tableHTML(ctx, headers, rows, ""))
		b.WriteString(renderPagination(ctx, p.BasePath, "/queries", page, ""))
		b.WriteString(formatLinks(ctx, p.BasePath, "/queries"))

		return b.String()
	})
}

// ===== Command Detail =====

func (d *Dashboard) commandDetailHandler(w http.ResponseWriter, r *http.Request) {
	cmdIDStr := r.PathValue("id")

	cmdID, err := id.ParseCommandID(cmdIDStr)
	if err != nil {
		d.renderError(w, r, http.StatusBadRequest, "invalid command ID")

		return
	}

	cmd, err := d.loadCommandByID(r.Context(), cmdID)
	if err != nil {
		d.renderError(w, r, http.StatusNotFound, "command not found")

		return
	}

	p := d.page("Command: "+truncate(string(cmd.Type()), eventTypeWidth), "/commands", r)
	html := d.renderCommandDetail(r.Context(), p, cmd)
	renderPage(w, r, html)
}

// loadCommandByID scans the command journal for a specific command.
//
//nolint:dupl // structurally mirrors loadQueryByID, different types
func (d *Dashboard) loadCommandByID(
	ctx context.Context,
	cmdID id.CommandID,
) (*command.PersistedCommand, error) {
	if seekable, ok := d.config.CommandJournal.(command.SeekableCommandJournal); ok {
		return scanJournalByID(ctx,
			func(c context.Context, after string, limit int) ([]*command.PersistedCommand, error) {
				cursor, _ := id.ParseCommandID(after)

				return seekable.ReadFrom(c, cursor, limit)
			},
			func(cmd *command.PersistedCommand) string { return cmd.ID().String() },
			cmdID.String(),
			fmt.Sprintf("command %s not found", cmdID),
			"dashboardui.command_detail.scan_failed", "scan command journal",
		)
	}

	return findInAll(ctx,
		d.config.CommandJournal.ReadAll,
		func(cmd *command.PersistedCommand) string { return cmd.ID().String() },
		cmdID.String(),
		fmt.Sprintf("command %s not found", cmdID),
		"dashboardui.command_detail.read_failed", "read command journal",
	)
}

func (d *Dashboard) renderCommandDetail(ctx context.Context, p pageData, cmd *command.PersistedCommand) string {
	return d.renderLayout(ctx, p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2><code>%s</code></h2>`, esc(string(cmd.Type())))
		fmt.Fprintf(
			&b,
			`<div class="page-subtitle mono">%s %s</div>`,
			esc(cmd.ID().String()),
			copyButtonHTML(ctx, cmd.ID().String(), ""),
		)
		b.WriteString(`</div>`)

		b.WriteString(`<div class="two-col-grid">`)

		mono := func(s string) string { return "<span class=\"mono\">" + s + "</span>" }

		items := []display.DefinitionItem{
			defItem("Command Type", string(cmd.Type())),
			defItem("Stream Type", string(cmd.StreamType())),
			defItemCopy("Stream ID", mono(esc(cmd.StreamID().String())), cmd.StreamID().String()),
			defItem("Received At", cmd.ReceivedAt().Format("2006-01-02 15:04:05")),
			defItemCopy("Command ID", mono(esc(cmd.ID().String())), cmd.ID().String()),
		}

		meta := cmd.Metadata()
		if corrID := meta.CorrelationID.String(); corrID != "" {
			items = append(items, defItemCopy("Correlation ID", mono(esc(corrID)), corrID))
		}

		if causID := meta.CausationID.String(); causID != "" {
			items = append(items, defItemCopy("Causation ID", mono(esc(causID)), causID))
		}

		if actorID := meta.ActorID; !actorID.IsZero() {
			actorPrefixed := actorID.PrefixedString()
			items = append(items, defItemCopy("Actor ID", mono(esc(actorPrefixed)), actorPrefixed))
		}

		b.WriteString(`<div><h3>Metadata</h3>`)
		b.WriteString(definitionListHTML(ctx, items))

		b.WriteString(`<div><h3>Payload</h3>`)

		payload := cmd.Payload()
		if len(payload) > 0 {
			pretty := prettyJSON(payload)
			fmt.Fprintf(&b, `<pre class="code-block"><code>%s</code></pre>`, esc(pretty))
		} else {
			b.WriteString(`<p class="muted">No payload</p>`)
		}

		b.WriteString(`</div></div>`)

		return b.String()
	})
}

// ===== Query Detail =====

func (d *Dashboard) queryDetailHandler(w http.ResponseWriter, r *http.Request) {
	queryIDStr := r.PathValue("id")

	queryID, err := id.ParseRequestID(queryIDStr)
	if err != nil {
		d.renderError(w, r, http.StatusBadRequest, "invalid query ID")

		return
	}

	q, err := d.loadQueryByID(r.Context(), queryID)
	if err != nil {
		d.renderError(w, r, http.StatusNotFound, "query not found")

		return
	}

	p := d.page("Query: "+truncate(string(q.Type()), eventTypeWidth), "/queries", r)
	html := d.renderQueryDetail(r.Context(), p, q)
	renderPage(w, r, html)
}

// loadQueryByID scans the query journal for a specific query.
//
//nolint:dupl // structurally mirrors loadCommandByID, different types
func (d *Dashboard) loadQueryByID(
	ctx context.Context,
	queryID id.RequestID,
) (*query.PersistedQuery, error) {
	if seekable, ok := d.config.QueryJournal.(query.SeekableQueryJournal); ok {
		return scanJournalByID(ctx,
			func(c context.Context, after string, limit int) ([]*query.PersistedQuery, error) {
				cursor, _ := id.ParseRequestID(after)

				return seekable.ReadQueriesFrom(c, cursor, limit)
			},
			func(q *query.PersistedQuery) string { return q.ID().String() },
			queryID.String(),
			fmt.Sprintf("query %s not found", queryID),
			"dashboardui.query_detail.scan_failed", "scan query journal",
		)
	}

	return findInAll(ctx,
		d.config.QueryJournal.ReadAllQueries,
		func(q *query.PersistedQuery) string { return q.ID().String() },
		queryID.String(),
		fmt.Sprintf("query %s not found", queryID),
		"dashboardui.query_detail.read_failed", "read query journal",
	)
}

func (d *Dashboard) renderQueryDetail(ctx context.Context, p pageData, q *query.PersistedQuery) string {
	return d.renderLayout(ctx, p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2><code>%s</code></h2>`, esc(string(q.Type())))
		fmt.Fprintf(
			&b,
			`<div class="page-subtitle mono">%s %s</div>`,
			esc(q.ID().String()),
			copyButtonHTML(ctx, q.ID().String(), ""),
		)
		b.WriteString(`</div>`)

		b.WriteString(`<div class="two-col-grid">`)

		mono := func(s string) string { return "<span class=\"mono\">" + s + "</span>" }

		items := []display.DefinitionItem{
			defItem("Query Type", string(q.Type())),
			defItem("Received At", q.ReceivedAt().Format("2006-01-02 15:04:05")),
			defItemCopy("Request ID", mono(esc(q.ID().String())), q.ID().String()),
		}

		meta := q.Metadata()
		if corrID := meta.CorrelationID.String(); corrID != "" {
			items = append(items, defItemCopy("Correlation ID", mono(esc(corrID)), corrID))
		}

		if causID := meta.CausationID.String(); causID != "" {
			items = append(items, defItemCopy("Causation ID", mono(esc(causID)), causID))
		}

		if actorID := meta.ActorID; !actorID.IsZero() {
			actorPrefixed := actorID.PrefixedString()
			items = append(items, defItemCopy("Actor ID", mono(esc(actorPrefixed)), actorPrefixed))
		}

		b.WriteString(`<div><h3>Metadata</h3>`)
		b.WriteString(definitionListHTML(ctx, items))

		b.WriteString(`<div><h3>Payload</h3>`)

		payload := q.Payload()
		if len(payload) > 0 {
			pretty := prettyJSON(payload)
			fmt.Fprintf(&b, `<pre class="code-block"><code>%s</code></pre>`, esc(pretty))
		} else {
			b.WriteString(`<p class="muted">No payload</p>`)
		}

		b.WriteString(`</div></div>`)

		return b.String()
	})
}

// scanJournalByID scans a seekable journal in batches looking for an entry
// whose ID matches targetID. Returns a rejection error if not found.
func scanJournalByID[T any](
	ctx context.Context,
	read func(ctx context.Context, after string, limit int) ([]T, error),
	idOf func(T) string,
	targetID string,
	notFoundMsg string,
	errCode, errDesc string,
) (T, error) {
	const scanLimit = 5000

	var after string

	for {
		batch, err := read(ctx, after, scanLimit)
		if err != nil {
			var zero T

			return zero, errorfamily.WrapInfrastructure(err, errCode, errDesc)
		}

		for _, item := range batch {
			if idOf(item) == targetID {
				return item, nil
			}
		}

		if len(batch) < scanLimit {
			break
		}

		after = idOf(batch[len(batch)-1])
	}

	var zero T

	return zero, errorfamily.NewRejection(
		"dashboardui.detail.not_found", notFoundMsg)
}

// findInAll loads all entries and searches linearly for one whose ID matches.
func findInAll[T any](
	ctx context.Context,
	readAll func(ctx context.Context) ([]T, error),
	idOf func(T) string,
	targetID string,
	notFoundMsg string,
	errCode, errDesc string,
) (T, error) {
	all, err := readAll(ctx)
	if err != nil {
		var zero T

		return zero, errorfamily.WrapInfrastructure(err, errCode, errDesc)
	}

	for _, item := range all {
		if idOf(item) == targetID {
			return item, nil
		}
	}

	var zero T

	return zero, errorfamily.NewRejection(
		"dashboardui.detail.not_found", notFoundMsg)
}
