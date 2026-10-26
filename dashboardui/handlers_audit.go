package dashboardui

import (
	"context"
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// ===== Command/Query Audit =====

// commandsAfter seeks a ReadAll result past the cursor. The non-seekable
// fallback must honor the after cursor or the Next link re-serves page one
// forever; a cursor that no longer exists restarts from the head.
func commandsAfter(cmds []*command.PersistedCommand, after id.CommandID) []*command.PersistedCommand {
	if after == (id.CommandID{}) {
		return cmds
	}

	for i, cmd := range cmds {
		if cmd.ID() == after {
			return cmds[i+1:]
		}
	}

	return cmds
}

// queriesAfter is the query-journal twin of commandsAfter.
func queriesAfter(queries []*query.PersistedQuery, after id.RequestID) []*query.PersistedQuery {
	if after == (id.RequestID{}) {
		return queries
	}

	for i, qry := range queries {
		if qry.ID() == after {
			return queries[i+1:]
		}
	}

	return queries
}

// exportBatch is the page size when walking a seekable journal for an
// export: small enough to bound each read, large enough to keep the walk
// cheap.
const exportBatch = 500

// loadAllCommands walks the command journal from the head to the tail. A
// seekable journal is paged by cursor (a single zero-cursor ReadFrom returns
// the OLDEST batch, silently truncating the export); a non-seekable journal
// falls back to one ReadAll, the only shape it offers. The export is the
// complete audit trail — truncation is the bug this replaces.
func loadAllCommands(
	ctx context.Context,
	journal command.CommandJournal,
) ([]*command.PersistedCommand, error) {
	seekable, ok := journal.(command.SeekableCommandJournal)
	if !ok {
		return journal.ReadAll(ctx)
	}

	var (
		all   []*command.PersistedCommand
		after id.CommandID
	)
	for {
		batch, err := seekable.ReadFrom(ctx, after, exportBatch)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.audit.commands_read_failed", "read command journal for export").
				WithContextAny("batch_size", exportBatch)
		}

		all = append(all, batch...)

		if len(batch) < exportBatch {
			break
		}

		after = batch[len(batch)-1].ID()
	}

	return all, nil
}

// loadAllQueries is the query-journal twin of loadAllCommands.
func loadAllQueries(
	ctx context.Context,
	journal query.QueryJournal,
) ([]*query.PersistedQuery, error) {
	seekable, ok := journal.(query.SeekableQueryJournal)
	if !ok {
		return journal.ReadAllQueries(ctx)
	}

	var (
		all   []*query.PersistedQuery
		after id.RequestID
	)
	for {
		batch, err := seekable.ReadQueriesFrom(ctx, after, exportBatch)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.audit.queries_read_failed", "read query journal for export").
				WithContextAny("batch_size", exportBatch)
		}

		all = append(all, batch...)

		if len(batch) < exportBatch {
			break
		}

		after = batch[len(batch)-1].ID()
	}

	return all, nil
}

//
//nolint:cyclop,dupl // export branching; deliberate parallel twin — see queriesIndexHandler
func (d *Dashboard) commandsIndexHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	p := d.page("Commands", "/commands", r)

	if format := parseFormat(r); format != formatHTML {
		if d.config.CommandJournal == nil {
			d.renderError(w, r, http.StatusInternalServerError, "no command journal configured")

			return
		}

		cmds, err := loadAllCommands(r.Context(), d.config.CommandJournal)
		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to load commands for export")

			return
		}

		switch format {
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

	after, okCursor := parseAfterCursor(d, w, r, afterCursor, id.ParseCommandID)
	if !okCursor {
		return
	}

	var (
		cmds    []*command.PersistedCommand
		hasNext bool
	)

	if d.config.CommandJournal != nil {
		var err error

		if seekable, ok := d.config.CommandJournal.(command.SeekableCommandJournal); ok {
			cmds, err = seekable.ReadFrom(r.Context(), after, pageSize+1)
		} else {
			cmds, err = d.config.CommandJournal.ReadAll(r.Context())
			cmds = commandsAfter(cmds, after)
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

	renderPage(w, r, commandsPage(p, cmds, paginationState{
		HasNext: hasNext, NextCursor: nextCursor, PageSize: pageSize, HasPrev: hasPrev,
		After: afterCursor, PrevHistory: prevHistory,
	}.WithCountInfo(len(cmds))))
}

// queriesIndexHandler is a deliberate parallel twin of commandsIndexHandler
// (accepted duplication, art-dupl round 2026-10-07): the journal interfaces
// diverge (SeekableCommandJournal.ReadFrom + CommandJournal.ReadAll +
// id.CommandID vs their query equivalents), as do the persisted types, the
// exporters (exportCommands*/exportQueries*), and the ID parsers
// (id.ParseCommandID vs id.ParseRequestID) — a shared template would need
// ~6 callbacks plus generics over two unrelated persisted types, more
// machinery than the mirrored lines save. dupl is suppressed on both
// handlers; change one, then mirror the change deliberately.
//
//nolint:cyclop,dupl // export branching; deliberate parallel twin — see rationale above
func (d *Dashboard) queriesIndexHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	p := d.page("Queries", "/queries", r)

	if format := parseFormat(r); format != formatHTML {
		var queries []*query.PersistedQuery

		var err error

		if seekable, ok := d.config.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, err = seekable.ReadQueriesFrom(r.Context(), id.RequestID{}, exportLimit)
		} else if d.config.QueryJournal != nil {
			queries, err = d.config.QueryJournal.ReadAllQueries(r.Context())
		}

		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to load queries for export")

			return
		}

		switch format {
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

	after, okCursor := parseAfterCursor(d, w, r, afterCursor, id.ParseRequestID)
	if !okCursor {
		return
	}

	var (
		queries []*query.PersistedQuery
		hasNext bool
	)

	if d.config.QueryJournal != nil {
		var err error

		if seekable, ok := d.config.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, err = seekable.ReadQueriesFrom(r.Context(), after, pageSize+1)
		} else {
			queries, err = d.config.QueryJournal.ReadAllQueries(r.Context())
			queries = queriesAfter(queries, after)
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

	renderPage(w, r, queriesPage(p, queries, paginationState{
		HasNext: hasNext, NextCursor: nextCursor, PageSize: pageSize, HasPrev: hasPrev,
		After: afterCursor, PrevHistory: prevHistory,
	}.WithCountInfo(len(queries))))
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
		d.renderLoadError(w, r, "command", err)

		return
	}

	p := d.page("Command: "+truncate(string(cmd.Type()), eventTypeWidth), "/commands", r)
	renderPage(w, r, commandDetailPage(p, cmd))
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
		d.renderLoadError(w, r, "query", err)

		return
	}

	p := d.page("Query: "+truncate(string(q.Type()), eventTypeWidth), "/queries", r)
	renderPage(w, r, queryDetailPage(p, q))
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

			return zero, errorfamily.WrapInfrastructure(err, errCode, errDesc).
				WithContext("target_id", targetID)
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
		"dashboardui.detail.not_found", notFoundMsg).WithContext("target_id", targetID)
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

		return zero, errorfamily.WrapInfrastructure(err, errCode, errDesc).
			WithContext("target_id", targetID)
	}

	for _, item := range all {
		if idOf(item) == targetID {
			return item, nil
		}
	}

	var zero T

	return zero, errorfamily.NewRejection(
		"dashboardui.detail.not_found", notFoundMsg).WithContext("target_id", targetID)
}
