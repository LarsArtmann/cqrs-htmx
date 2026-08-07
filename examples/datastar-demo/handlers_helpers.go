package main

import (
	"fmt"
	"log"
	"net/http"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type Signals struct {
	Title string `json:"title"`
	ID    string `json:"id"`
}

// writeErrorResponse sends an error as a DataStar signals patch.
func writeErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	resp := ds.NewResponse(w, r)
	if err := resp.PatchSignals([]byte(fmt.Sprintf(`{"error":{"message":%q}}`, err.Error()))); err != nil {
		log.Printf("datastar: writeErrorResponse patch failed: %v", err)
	}
}

// readSignals parses the request signals into s, writing an error response
// and returning false on parse failure.
func readSignals(w http.ResponseWriter, r *http.Request, s *Signals) bool {
	if err := ds.ReadSignals(r, s); err != nil {
		writeErrorResponse(w, r, err)
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
			resp := ds.NewResponse(w, r)
			if err := resp.MarshalAndPatchSignals(map[string]any{
				"notification": map[string]string{
					"level":   "error",
					"message": "Title is required",
				},
			}); err != nil {
				log.Printf("datastar: patch signals failed: %v", err)
			}
			return
		}

		ctx := ContextWithUser(r.Context(), &UserContext{Name: "you"})
		cmd, err := NewCreateTodo(s.Title)
		if err != nil {
			writeErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			writeErrorResponse(w, r, err)
			return
		}

		resp := ds.NewResponse(w, r)
		if err := resp.MarshalAndPatchSignals(map[string]any{
			"title": "",
			"notification": map[string]string{
				"level":   "success",
				"message": fmt.Sprintf("Created: %s", s.Title),
			},
		}); err != nil {
			log.Printf("datastar: patch signals failed: %v", err)
		}
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
			writeErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			writeErrorResponse(w, r, err)
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
			writeErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			writeErrorResponse(w, r, err)
			return
		}

		resp := ds.NewResponse(w, r)
		if err := resp.MarshalAndPatchSignals(map[string]any{
			"notification": map[string]string{
				"level":   "info",
				"message": "Todo deleted",
			},
		}); err != nil {
			log.Printf("datastar: patch signals failed: %v", err)
		}
	}
}

func handleUpdateTodo(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Signals
		if !readSignals(w, r, &s) {
			return
		}
		if s.ID == "" {
			writeErrorResponse(w, r, errorfamily.NewRejection("todo.id_required", "id is required"))
			return
		}
		if s.Title == "" {
			writeErrorResponse(w, r, errorfamily.NewRejection("todo.title_required", "title is required"))
			return
		}

		ctx := ContextWithUser(r.Context(), &UserContext{Name: "you"})
		cmd, err := NewUpdateTodo(s.ID, s.Title)
		if err != nil {
			writeErrorResponse(w, r, err)
			return
		}

		if err := cqrs.Commands.Dispatch(ctx, cmd); err != nil {
			writeErrorResponse(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleListTodos(cqrs *CQRS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qry, err := NewListTodosQry()
		if err != nil {
			writeErrorResponse(w, r, err)
			return
		}
		todos, err := query.DispatchTyped[[]Todo](r.Context(), cqrs.Queries, qry)
		if err != nil {
			writeErrorResponse(w, r, err)
			return
		}

		resp := ds.NewResponse(w, r)
		if err := resp.PatchElements(renderTodoList(todos), ds.WithSelectorID("todo-list"), ds.WithModeInner()); err != nil {
			log.Printf("datastar: patch elements failed: %v", err)
		}
		if err := resp.PatchElements(renderStatsFromQuery(cqrs), ds.WithSelectorID("stats")); err != nil {
			log.Printf("datastar: patch elements failed: %v", err)
		}
	}
}
