package main

import (
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	"github.com/starfederation/datastar-go/datastar"
)

type Signals struct {
	Title string `json:"title"`
	ID    string `json:"id"`
}

// readSignals parses the request signals into s, writing a 400 response and
// returning false on parse failure. Use this at the top of every datastar
// command handler to keep the error-handling shape consistent.
func readSignals(w http.ResponseWriter, r *http.Request, s *Signals) bool {
	if err := datastar.ReadSignals(r, s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
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
		if !readSignals(w, r, &s) {
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
		if !readSignals(w, r, &s) {
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
		if !readSignals(w, r, &s) {
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
		if !readSignals(w, r, &s) {
			return
		}
		if s.ID == "" {
			dispatchErrorNotification(w, r, event.NewRejection("todo.id_required", "id is required"))
			return
		}
		if s.Title == "" {
			dispatchErrorNotification(w, r, event.NewRejection("todo.title_required", "title is required"))
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
