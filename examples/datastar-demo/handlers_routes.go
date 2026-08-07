package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func eventKindFromType(t string) string {
	switch t {
	case "TodoCreated":
		return "todo_created"
	case "TodoToggled", "TodoUpdated":
		return "todo_updated"
	case "TodoDeleted":
		return "todo_deleted"
	}
	return "unknown"
}

// handleSimulate starts background goroutines that act as simulated users.
// Returns immediately — the bots keep running and all updates flow through the broadcast.
func handleSimulate(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		for _, name := range botNames {
			go SimulateUser(ctx, cqrs, name)
		}

		resp := ds.NewResponse(w, r)
		resp.MarshalAndPatchSignals(map[string]any{
			"simulating": true,
			"notification": map[string]string{
				"level":   "warning",
				"message": fmt.Sprintf("Simulating %d users: %v", len(botNames), botNames),
			},
		})
	}
}

// --- HTML Renderers ---

func renderTodo(t Todo) string {
	checked := ""
	completedClass := ""
	if t.Completed {
		checked = "checked"
		completedClass = " completed"
	}
	return fmt.Sprintf(`<li id="todo-%s" class="todo-item%s">
	<input type="checkbox" %s data-on:click="@post('/api/todos/toggle')" data-signals:id="'%s'">
	<input type="text" class="todo-title-input" value="%s" data-bind:title data-signals:edit_id="'%s'">
	<button class="edit-btn" data-on:click="@post('/api/todos/update')" data-signals:id="$edit_id || '%s'">edit</button>
	<button class="delete-btn" data-on:click="@post('/api/todos/delete')" data-signals:id="'%s'">x</button>
</li>`, t.ID, completedClass, checked, t.ID, t.Title, t.ID, t.ID, t.ID)
}

func renderTodoList(todos []Todo) string {
	if len(todos) == 0 {
		return `<li class="empty-state">No todos yet. Add one above!</li>`
	}
	var b strings.Builder
	for _, t := range todos {
		b.WriteString(renderTodo(t))
	}
	return b.String()
}

func renderStats(cqrs *CQRS) string {
	total, active, completed := cqrs.Read.Stats()
	return fmt.Sprintf(`<div id="stats" class="stats">
	<span>Total: <strong>%d</strong></span>
	<span>Active: <strong>%d</strong></span>
	<span>Completed: <strong>%d</strong></span>
</div>`, total, active, completed)
}

// renderStatsFromQuery is the recommended path for rendering stats.
// It dispatches a GetStatsQry through the typed query dispatcher
// instead of reading the projector directly, so any cross-cutting
// concerns (caching, authorization, instrumentation) flow through
// the same pipeline as production reads.
func renderStatsFromQuery(cqrs *CQRS) string {
	qry, err := NewGetStatsQry()
	if err != nil {
		return renderStats(cqrs) // fall back to direct read on construction error
	}
	stats, err := query.DispatchTyped[Stats](context.Background(), cqrs.Queries, qry)
	if err != nil {
		return renderStats(cqrs) // fall back on dispatch error
	}
	return fmt.Sprintf(`<div id="stats" class="stats">
	<span>Total: <strong>%d</strong></span>
	<span>Active: <strong>%d</strong></span>
	<span>Completed: <strong>%d</strong></span>
</div>`, stats.Total, stats.Active, stats.Completed)
}

func renderEventLogEntry(e DomainEvent) string {
	return fmt.Sprintf(`<div class="event-entry">
	<span class="event-type">%s</span>
	<span class="event-user">%s</span>
	<span class="event-time">%s</span>
</div>`, eventKindFromType(e.Type), e.User, e.OccurredAt.Format("15:04:05"))
}
