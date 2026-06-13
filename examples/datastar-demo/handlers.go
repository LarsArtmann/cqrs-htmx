package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/starfederation/datastar-go/datastar"
)

type Signals struct {
	Title string `json:"title"`
	ID    string `json:"id"`
}

func dispatchErrorNotification(w http.ResponseWriter, r *http.Request, err error) {
	sse := datastar.NewSSE(w, r)
	sse.MarshalAndPatchSignals(map[string]any{
		"notification": map[string]string{
			"level":   "error",
			"message": err.Error(),
		},
	})
}

func handleCreateTodo(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Signals
		if err := datastar.ReadSignals(r, &s); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.Title == "" {
			sse := datastar.NewSSE(w, r)
			sse.MarshalAndPatchSignals(map[string]any{
				"notification": map[string]string{
					"level":   "error",
					"message": "Title is required",
				},
			})
			return
		}

		ctx := ContextWithUser(r.Context(), &UserContext{Name: "you"})
		cmd, err := NewCreateTodo(s.Title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			dispatchErrorNotification(w, r, err)
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.MarshalAndPatchSignals(map[string]any{
			"title": "",
		})
	}
}

func handleToggleTodo(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Signals
		if err := datastar.ReadSignals(r, &s); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := ContextWithUser(r.Context(), &UserContext{Name: "you"})
		cmd, err := NewToggleTodo(s.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			dispatchErrorNotification(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleDeleteTodo(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Signals
		if err := datastar.ReadSignals(r, &s); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := ContextWithUser(r.Context(), &UserContext{Name: "you"})
		cmd, err := NewDeleteTodo(s.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			dispatchErrorNotification(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// handleUpdateTodo demonstrates the update command pattern. The client
// sends signals with id and title; the server dispatches UpdateTodoCmd
// through the typed command dispatcher. The resulting event updates the
// read model and broadcasts to all SSE clients.
func handleUpdateTodo(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Signals
		if err := datastar.ReadSignals(r, &s); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.ID == "" {
			dispatchErrorNotification(w, r, fmt.Errorf("id is required"))
			return
		}
		if s.Title == "" {
			dispatchErrorNotification(w, r, fmt.Errorf("title is required"))
			return
		}

		ctx := ContextWithUser(r.Context(), &UserContext{Name: "you"})
		cmd, err := NewUpdateTodo(s.ID, s.Title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			dispatchErrorNotification(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleListTodos(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qry, err := NewListTodosQry()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		todos, err := query.DispatchTyped[[]Todo](r.Context(), cqrs.Queries, qry)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(
			renderTodoList(todos),
			datastar.WithSelectorID("todo-list"),
			datastar.WithModeInner(),
		)
		// Render stats via the typed query dispatcher to demonstrate
		// routing reads through cqrs.Queries. Equivalent to calling
		// renderStats(cqrs) for the demo, but the query path supports
		// caching, authorization, and cross-module instrumentation.
		sse.PatchElements(renderStatsFromQuery(cqrs), datastar.WithSelectorID("stats"))
	}
}

// handleEventStream is the real-time SSE endpoint shared by ALL connected clients.
// Each client gets its own channel via Broadcaster.Subscribe().
// Domain events from any source (user actions, bots) are fanned out to every client.
func handleEventStream(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch := cqrs.Broadcast.Subscribe()
		defer cqrs.Broadcast.Unsubscribe(ch)

		sse := datastar.NewSSE(w, r)

		for {
			select {
			case <-r.Context().Done():
				return
			case evt := <-ch:
				if sse.IsClosed() {
					return
				}

				switch evt.Kind {
				case "todo_created":
					if evt.User == "you" {
						sse.MarshalAndPatchSignals(map[string]any{
							"title": "",
							"notification": map[string]string{
								"level":   "success",
								"message": fmt.Sprintf("Created: %s", extractTitle(evt.Data)),
							},
						})
					}
				case "todo_deleted":
					if evt.User == "you" {
						sse.MarshalAndPatchSignals(map[string]any{
							"notification": map[string]string{
								"level":   "info",
								"message": "Todo deleted",
							},
						})
					}
				}

				todos := cqrs.Read.List()
				sse.PatchElements(
					renderTodoList(todos),
					datastar.WithSelectorID("todo-list"),
					datastar.WithModeInner(),
				)
				sse.PatchElements(renderStatsFromQuery(cqrs), datastar.WithSelectorID("stats"))

				sse.PatchElements(
					renderEventLog(evt),
					datastar.WithSelectorID("event-log"),
					datastar.WithModePrepend(),
				)
			}
		}
	}
}

// handleEventReplay replays the event log (read-model + event list) for a
// reconnecting client. Uses the standard Last-Event-ID header mechanism:
// browsers send it automatically on EventSource reconnection.
//
// The replay path here is intentionally simple: the entire event log is
// in-memory and small, so we send everything strictly after the last
// known ID. In production, swap the EventStore for one that pages from
// durable storage.
func handleEventReplay(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lastID := r.Header.Get("Last-Event-ID")

		// Send the current read model first.
		todos := cqrs.Read.List()
		sse := datastar.NewSSE(w, r)
		sse.PatchElements(
			renderTodoList(todos),
			datastar.WithSelectorID("todo-list"),
			datastar.WithModeInner(),
		)
		sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))

		// Then replay missed events.
		all := cqrs.Events.All()
		for _, evt := range all {
			// Match by event Time.Format("15:04:05.000") as a stand-in for
			// a stable event ID. In production, use the event's ULID.
			id := fmt.Sprintf("evt-%d", evt.OccurredAt.UnixNano())
			if lastID != "" && id <= lastID {
				continue
			}
			sse.PatchElements(
				renderEventLog(BroadcastEvent{
					Kind: eventKindFromType(evt.Type),
					User: evt.User,
					Time: evt.OccurredAt,
				}),
				datastar.WithSelectorID("event-log"),
				datastar.WithModePrepend(),
			)
		}
	}
}

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

		sse := datastar.NewSSE(w, r)
		sse.MarshalAndPatchSignals(map[string]any{
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

func renderEventLog(evt BroadcastEvent) string {
	return fmt.Sprintf(`<div class="event-entry">
	<span class="event-type">%s</span>
	<span class="event-user">%s</span>
	<span class="event-time">%s</span>
</div>`, evt.Kind, evt.User, evt.Time.Format("15:04:05"))
}

// extractTitle pulls the text content from <span class="todo-title">...</span>
func extractTitle(html string) string {
	const marker = `<span class="todo-title">`
	const end = `</span>`
	_, rest, found := strings.Cut(html, marker)
	if !found {
		return ""
	}
	before, _, found := strings.Cut(rest, end)
	if !found {
		return rest
	}
	return before
}
