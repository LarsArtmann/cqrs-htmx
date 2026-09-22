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

//
//nolint:cyclop // export branching adds complexity
func (d *Dashboard) commandsIndexHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	p := d.page("Commands", "/commands", r)

	if format := parseFormat(r); format != formatHTML {
		var cmds []*command.PersistedCommand

		if seekable, ok := d.config.CommandJournal.(command.SeekableCommandJournal); ok {
			cmds, _ = seekable.ReadFrom(r.Context(), id.CommandID{}, exportLimit)
		} else if d.config.CommandJournal != nil {
			cmds, _ = d.config.CommandJournal.ReadAll(r.Context())
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

	renderPage(w, r, commandsPage(p, cmds, paginationState{
		HasNext: hasNext, NextCursor: nextCursor, PageSize: pageSize, HasPrev: hasPrev,
		After: afterCursor, PrevHistory: prevHistory,
	}.WithCountInfo(len(cmds))))
}

//
//nolint:cyclop // export branching adds complexity
func (d *Dashboard) queriesIndexHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	p := d.page("Queries", "/queries", r)

	if format := parseFormat(r); format != formatHTML {
		var queries []*query.PersistedQuery

		if seekable, ok := d.config.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, _ = seekable.ReadQueriesFrom(r.Context(), id.RequestID{}, exportLimit)
		} else if d.config.QueryJournal != nil {
			queries, _ = d.config.QueryJournal.ReadAllQueries(r.Context())
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
		d.renderError(w, r, http.StatusNotFound, "command not found")

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
		d.renderError(w, r, http.StatusNotFound, "query not found")

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
