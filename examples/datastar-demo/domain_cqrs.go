package main

import (
	"context"
	"encoding/json/v2"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
)

// --- CQRS Setup ---

type CQRS struct {
	Commands  *command.Dispatcher
	Queries   *query.Dispatcher
	Events    *EventStore
	Read      *Projector
	Broadcast *ds.Broadcaster
}

func NewCQRS() *CQRS {
	events := NewEventStore()
	read := NewProjector()
	broadcast := ds.NewBroadcaster()

	//cqrs-lint:ignore(C027) example event bus: Subscribe is the only projection mechanism here, no projectionhost in this demo
	events.Subscribe(read.Apply)

	cmdDisp := command.NewDispatcher()
	qryDisp := query.NewDispatcher()

	cqrs := &CQRS{
		Commands:  cmdDisp,
		Queries:   qryDisp,
		Events:    events,
		Read:      read,
		Broadcast: broadcast,
	}

	cqrs.registerCommandHandlers()
	cqrs.registerQueryHandlers()

	// Broadcast bridge: domain events → Datastar patches → all SSE clients.
	// The Broadcaster handles fan-out and reconnection replay automatically.
	//cqrs-lint:ignore(C027) example broadcast bridge, not a read-model projection
	events.Subscribe(func(e DomainEvent) {
		cqrs.Broadcast.BroadcastMany(
			ds.ElementsPatch(renderTodoList(read.List()), ds.WithSelectorID("todo-list"), ds.WithModeInner()),
			ds.ElementsPatch(renderStatsFromQuery(cqrs), ds.WithSelectorID("stats")),
			ds.ElementsPatch(renderEventLogEntry(e), ds.WithSelectorID("event-log"), ds.WithModePrepend()),
		)
	})

	return cqrs
}

func userName(ctx context.Context) string {
	if u := UserFromContext(ctx); u != nil {
		return u.Name
	}
	return "you"
}

// appendDomainEvent records a domain event with the common envelope fields
// (aggregate ID, user from context, occurred-at). Only the event type and
// payload vary per command — keeping the envelope in one place prevents drift.
func (c *CQRS) appendDomainEvent(ctx context.Context, aggID, eventType string, payload []byte) {
	c.Events.Append(DomainEvent{
		AggregateID: aggID,
		Type:        eventType,
		User:        userName(ctx),
		Payload:     payload,
		OccurredAt:  time.Now(),
	})
}

func (c *CQRS) registerCommandHandlers() {
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(c.Commands, "CreateTodo", func(ctx context.Context, cmd *CreateTodoCmd) error {
		todoID := cmd.StreamID().String()
		payload, _ := json.Marshal(TodoCreatedPayload{
			ID:        todoID,
			Title:     cmd.Title,
			CreatedAt: time.Now().Format(time.RFC3339),
		})

		c.appendDomainEvent(ctx, todoID, "TodoCreated", payload)
		return nil
	})

	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(c.Commands, "ToggleTodo", func(ctx context.Context, cmd *ToggleTodoCmd) error {
		todoID := cmd.StreamID().String()
		payload, _ := json.Marshal(TodoToggledPayload{ID: todoID})

		c.appendDomainEvent(ctx, todoID, "TodoToggled", payload)
		return nil
	})

	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(c.Commands, "DeleteTodo", func(ctx context.Context, cmd *DeleteTodoCmd) error {
		todoID := cmd.StreamID().String()
		payload, _ := json.Marshal(TodoDeletedPayload{ID: todoID})

		c.appendDomainEvent(ctx, todoID, "TodoDeleted", payload)
		return nil
	})

	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(c.Commands, "UpdateTodo", func(ctx context.Context, cmd *UpdateTodoCmd) error {
		todoID := cmd.StreamID().String()
		payload, _ := json.Marshal(TodoUpdatedPayload{ID: todoID, Title: cmd.Title})

		c.appendDomainEvent(ctx, todoID, "TodoUpdated", payload)
		return nil
	})
}

func (c *CQRS) registerQueryHandlers() {
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = query.RegisterTyped(c.Queries, "ListTodos", func(ctx context.Context, q *ListTodosQry) ([]Todo, error) {
		return c.Read.List(), nil
	})

	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = query.RegisterTyped(c.Queries, "GetStats", func(_ context.Context, _ *GetStatsQry) (Stats, error) {
		total, active, completed := c.Read.Stats()
		return Stats{Total: total, Active: active, Completed: completed}, nil
	})
}
