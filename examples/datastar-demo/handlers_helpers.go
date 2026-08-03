package main

import (
	"fmt"
	"net/http"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type Signals struct {
	Title string `json:"title"`
	ID    string `json:"id"`
}

// readSignals parses the request signals into s, writing an error response
// and returning false on parse failure.
func readSignals(w http.ResponseWriter, r *http.Request, s *Signals) bool {
	if err := ds.ReadSignals(r, s); err != nil {
		ds.ErrorResponse(w, r, err)
		return false
	}
	return true
}

func handleCreateTodo(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Signals
		if !readSignals(w, r, &s) {
			return
		}
		if s.Title == "" {
			ds.NewResponse(w, r).PatchSignals(map[string]any{
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
			ds.ErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			ds.ErrorResponse(w, r, err)
			return
		}

		ds.NewResponse(w, r).PatchSignals(map[string]any{
			"title": "",
			"notification": map[string]string{
				"level":   "success",
				"message": fmt.Sprintf("Created: %s", s.Title),
			},
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
			ds.ErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			ds.ErrorResponse(w, r, err)
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
			ds.ErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			ds.ErrorResponse(w, r, err)
			return
		}

		ds.NewResponse(w, r).PatchSignals(map[string]any{
			"notification": map[string]string{
				"level":   "info",
				"message": "Todo deleted",
			},
		})
	}
}

func handleUpdateTodo(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Signals
		if !readSignals(w, r, &s) {
			return
		}
		if s.ID == "" {
			ds.ErrorResponse(w, r, errorfamily.NewRejection("todo.id_required", "id is required"))
			return
		}
		if s.Title == "" {
			ds.ErrorResponse(w, r, errorfamily.NewRejection("todo.title_required", "title is required"))
			return
		}

		ctx := ContextWithUser(r.Context(), &UserContext{Name: "you"})
		cmd, err := NewUpdateTodo(s.ID, s.Title)
		if err != nil {
			ds.ErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			ds.ErrorResponse(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleListTodos(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qry, err := NewListTodosQry()
		if err != nil {
			ds.ErrorResponse(w, r, err)
			return
		}
		todos, err := query.DispatchTyped[[]Todo](r.Context(), cqrs.Queries, qry)
		if err != nil {
			ds.ErrorResponse(w, r, err)
			return
		}

		ds.NewResponse(w, r).
			PatchElements(renderTodoList(todos), ds.WithSelectorID("todo-list"), ds.WithModeInner()).
			PatchElements(renderStatsFromQuery(cqrs), ds.WithSelectorID("stats"))
	}
}
