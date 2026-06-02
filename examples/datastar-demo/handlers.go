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
			sse := datastar.NewSSE(w, r)
			sse.MarshalAndPatchSignals(map[string]any{
				"notification": map[string]string{
					"level":   "error",
					"message": err.Error(),
				},
			})
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
			sse := datastar.NewSSE(w, r)
			sse.MarshalAndPatchSignals(map[string]any{
				"notification": map[string]string{
					"level":   "error",
					"message": err.Error(),
				},
			})
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
			sse := datastar.NewSSE(w, r)
			sse.MarshalAndPatchSignals(map[string]any{
				"notification": map[string]string{
					"level":   "error",
					"message": err.Error(),
				},
			})
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleListTodos(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := cqrs.Queries.Dispatch(r.Context(), query.MustNew("ListTodos"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		todos, ok := result.([]Todo)
		if !ok {
			http.Error(w, "unexpected result type", http.StatusInternalServerError)
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(
			renderTodoList(todos),
			datastar.WithSelectorID("todo-list"),
			datastar.WithModeInner(),
		)
		sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))
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
				sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))

				sse.PatchElements(
					renderEventLog(evt),
					datastar.WithSelectorID("event-log"),
					datastar.WithModePrepend(),
				)
			}
		}
	}
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
	<span class="todo-title">%s</span>
	<button class="delete-btn" data-on:click="@post('/api/todos/delete')" data-signals:id="'%s'">x</button>
</li>`, t.ID, completedClass, checked, t.ID, t.Title, t.ID)
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
	rest, ok := strings.CutPrefix(html, marker)
	if !ok {
		if i := strings.Index(html, marker); i >= 0 {
			rest = html[i+len(marker):]
		} else {
			return ""
		}
	}
	before, _, found := strings.Cut(rest, end)
	if !found {
		return rest
	}
	return before
}
