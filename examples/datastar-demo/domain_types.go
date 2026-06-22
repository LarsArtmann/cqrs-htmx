package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

type UserContext struct {
	Name string
}

func UserFromContext(ctx context.Context) *UserContext {
	if u, ok := ctx.Value(userCtxKey{}).(*UserContext); ok {
		return u
	}
	return nil
}

func ContextWithUser(ctx context.Context, u *UserContext) context.Context {
	return context.WithValue(ctx, userCtxKey{}, u)
}

type userCtxKey struct{}

// --- Domain Types ---

type Todo struct {
	ID        string
	Title     string
	Completed bool
	CreatedAt time.Time
}

// --- Domain Events ---

type DomainEvent struct {
	AggregateID string
	Type        string
	User        string
	Payload     json.RawMessage
	OccurredAt  time.Time
}

type TodoCreatedPayload struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

type TodoToggledPayload struct {
	ID string `json:"id"`
}

type TodoDeletedPayload struct {
	ID string `json:"id"`
}

// --- Commands ---

type CreateTodoCmd struct {
	*command.BasicCommand
	Title string
}

type ToggleTodoCmd struct {
	*command.BasicCommand
}

type DeleteTodoCmd struct {
	*command.BasicCommand
}

// UpdateTodoCmd changes the title of an existing todo.
// It demonstrates the update-pattern: an event-driven replacement of
// a single field on an aggregate.
type UpdateTodoCmd struct {
	*command.BasicCommand
	Title string
}

// Payload emitted by UpdateTodoCmd when a todo's title changes.
type TodoUpdatedPayload struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// --- Queries ---

type ListTodosQry struct {
	*query.BasicQuery
}

func NewListTodosQry() (*ListTodosQry, error) {
	core, err := query.New("ListTodos")
	if err != nil {
		return nil, err
	}
	return &ListTodosQry{BasicQuery: core}, nil
}

// Stats is the read-only result of a GetStatsQry.
type Stats struct {
	Total     int
	Active    int
	Completed int
}

// GetStatsQry is a typed query that returns aggregate todo counts.
type GetStatsQry struct {
	*query.BasicQuery
}

func NewGetStatsQry() (*GetStatsQry, error) {
	core, err := query.New("GetStats")
	if err != nil {
		return nil, err
	}
	return &GetStatsQry{BasicQuery: core}, nil
}
