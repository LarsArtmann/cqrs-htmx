package main

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// --- Command Constructors ---

func NewCreateTodo(title string) (*CreateTodoCmd, error) {
	aggID := id.NewStreamID()
	core, err := command.New("CreateTodo", aggID)
	if err != nil {
		return nil, err
	}
	return &CreateTodoCmd{BasicCommand: core, Title: title}, nil
}

// newTodoCore is the shared backend for the typed todo command constructors:
// it validates the incoming todo ID, builds the named command, and wraps any
// failure with the todo aggregate's error tags. Callers wrap the returned
// core in their specific *<Name>TodoCmd type.
func newTodoCore(todoID, name string) (*command.BasicCommand, error) {
	aggID, err := id.ParseStreamID(todoID)
	if err != nil {
		return nil, errorfamily.Wrapf(err, event.Rejection, "todo.invalid_id", "invalid todo ID %q", todoID)
	}
	core, err := command.New(command.Type(name), aggID)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"todo.command_failed",
			"create %s command for todo %q",
			name,
			todoID,
		)
	}
	return core, nil
}

func NewToggleTodo(todoID string) (*ToggleTodoCmd, error) {
	core, err := newTodoCore(todoID, "ToggleTodo")
	if err != nil {
		return nil, err
	}
	return &ToggleTodoCmd{BasicCommand: core}, nil
}

func NewDeleteTodo(todoID string) (*DeleteTodoCmd, error) {
	core, err := newTodoCore(todoID, "DeleteTodo")
	if err != nil {
		return nil, err
	}
	return &DeleteTodoCmd{BasicCommand: core}, nil
}

func NewUpdateTodo(todoID, title string) (*UpdateTodoCmd, error) {
	core, err := newTodoCore(todoID, "UpdateTodo")
	if err != nil {
		return nil, err
	}
	return &UpdateTodoCmd{BasicCommand: core, Title: title}, nil
}

// --- Simulator ---

var botNames = []string{
	"alice", "bob", "carol", "dave", "eve",
	"frank", "grace", "heidi", "ivan", "judy",
}

var botTodos = []string{
	"Review pull request", "Fix CI pipeline", "Update dependencies",
	"Write integration tests", "Refactor auth middleware", "Add rate limiting",
	"Document the API", "Set up monitoring", "Clean up tech debt", "Ship v2",
	"Fix memory leak", "Add CORS headers", "Implement caching", "Update README",
}

// SimulateUser runs a bot that performs random todo actions as a background goroutine.
// All events go through the normal CQRS pipeline and broadcast to every connected SSE client.
func SimulateUser(ctx context.Context, cqrs *CQRS, name string) {
	r := func() { time.Sleep(time.Duration(800+time.Now().UnixNano()%2200) * time.Millisecond) }

	userCtx := ContextWithUser(ctx, &UserContext{Name: name})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		roll := time.Now().UnixNano() % 10

		switch {
		case roll < 5:
			title := botTodos[time.Now().UnixNano()%int64(len(botTodos))]
			cmd, _ := NewCreateTodo(title)
			_ = cqrs.Commands.Dispatch(userCtx, cmd)
		case roll < 8:
			todos := cqrs.Read.List()
			if len(todos) > 0 {
				t := todos[time.Now().UnixNano()%int64(len(todos))]
				cmd, _ := NewToggleTodo(t.ID)
				_ = cqrs.Commands.Dispatch(userCtx, cmd)
			}
		default:
			todos := cqrs.Read.List()
			completed := make([]Todo, 0)
			for _, t := range todos {
				if t.Completed {
					completed = append(completed, t)
				}
			}
			if len(completed) > 0 {
				t := completed[time.Now().UnixNano()%int64(len(completed))]
				cmd, _ := NewDeleteTodo(t.ID)
				_ = cqrs.Commands.Dispatch(userCtx, cmd)
			}
		}

		r()
	}
}
