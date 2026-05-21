package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/core/query"
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

		todoID := cmd.AggregateID().String()
		todo := findTodo(cqrs, todoID)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(
			renderTodo(todo),
			datastar.WithSelectorID("todo-list"),
			datastar.WithModeAppend(),
		)
		sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))
		sse.MarshalAndPatchSignals(map[string]any{
			"title": "",
			"notification": map[string]string{
				"level":   "success",
				"message": fmt.Sprintf("Created: %s", todo.Title),
			},
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

		todo := findTodo(cqrs, s.ID)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(renderTodo(todo))
		sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))
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

		sse := datastar.NewSSE(w, r)
		sse.RemoveElement("#todo-" + s.ID)
		sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))
		sse.MarshalAndPatchSignals(map[string]any{
			"notification": map[string]string{
				"level":   "info",
				"message": "Todo deleted",
			},
		})
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

// handleEventStream is the real-time SSE endpoint that ALL connected clients share.
// It listens to the broadcast channel (fed by domain events from any source) and
// pushes DOM patches + event log entries to every connected browser.
func handleEventStream(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		for {
			select {
			case <-r.Context().Done():
				return
			case evt := <-cqrs.Broadcast:
				if sse.IsClosed() {
					return
				}
				switch evt.Kind {
				case "todo_created":
					sse.PatchElements(
						evt.Data,
						datastar.WithSelectorID("todo-list"),
						datastar.WithModeAppend(),
					)
					sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))
				case "todo_updated":
					sse.PatchElements(evt.Data)
					sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))
				case "todo_deleted":
					sse.RemoveElement(evt.Data)
					sse.PatchElements(renderStats(cqrs), datastar.WithSelectorID("stats"))
				}
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
// Each bot creates, toggles, and deletes todos through the same CQRS pipeline.
func handleSimulate(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(context.Background())

		for _, name := range botNames {
			go SimulateUser(ctx, cqrs, name)
		}

		sse := datastar.NewSSE(w, r)
		sse.MarshalAndPatchSignals(map[string]any{
			"notification": map[string]string{
				"level":   "warning",
				"message": fmt.Sprintf("Simulating %d users: %v", len(botNames), botNames),
			},
			"simulating": true,
		})

		<-r.Context().Done()
		cancel()
	}
}

func findTodo(cqrs *CQRS, id string) Todo {
	result, _ := cqrs.Queries.Dispatch(context.Background(), query.MustNew("ListTodos"))
	todos, _ := result.([]Todo)
	for _, t := range todos {
		if t.ID == id {
			return t
		}
	}
	return Todo{ID: id}
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
	html := ""
	for _, t := range todos {
		html += renderTodo(t)
	}
	return html
}

func renderStats(cqrs *CQRS) string {
	total, active, completed := cqrs.Read.Stats()
	return fmt.Sprintf(`<div id="stats" class="stats">
	<span>Total: <strong>%d</strong></span>
	<span>Active: <strong>%d</strong></span>
	<span>Completed: <strong>%d</strong></span>
</div>`, total, active, completed)
}

// renderEventLog renders a broadcast event as an event-log entry with user attribution.
func renderEventLog(evt BroadcastEvent) string {
	return fmt.Sprintf(`<div class="event-entry">
	<span class="event-type">%s</span>
	<span class="event-user">%s</span>
	<span class="event-time">%s</span>
</div>`, evt.Kind, evt.User, evt.Time.Format("15:04:05"))
}
