package main

import (
	"context"
	"encoding/json/v2"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// --- CQRS Setup ---

type BroadcastEvent struct {
	Kind string // "todo_created", "todo_updated", "todo_deleted"
	User string
	Data string // HTML fragment or CSS selector
	Time time.Time
}

type Broadcaster struct {
	mu   sync.Mutex
	subs map[chan BroadcastEvent]struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{subs: make(map[chan BroadcastEvent]struct{})}
}

func (b *Broadcaster) Subscribe() chan BroadcastEvent {
	ch := make(chan BroadcastEvent, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broadcaster) Unsubscribe(ch chan BroadcastEvent) {
	b.mu.Lock()
	delete(b.subs, ch)
	b.mu.Unlock()
}

// Send broadcasts an event to all subscribers. The snapshot-then-send
// pattern is safe ONLY because Unsubscribe does not close the channel.
// If you change Unsubscribe to close(ch), you MUST hold the lock during
// the send loop (see cqrs-htmx fanout.go for the correct pattern).
func (b *Broadcaster) Send(evt BroadcastEvent) {
	b.mu.Lock()
	subs := make([]chan BroadcastEvent, 0, len(b.subs))
	for ch := range b.subs {
		subs = append(subs, ch)
	}
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
		}
	}
}

type CQRS struct {
	Commands  *command.Dispatcher
	Queries   *query.Dispatcher
	Events    *EventStore
	Read      *Projector
	Broadcast *Broadcaster
}

func NewCQRS() *CQRS {
	events := NewEventStore()
	read := NewProjector()
	broadcast := NewBroadcaster()

	events.Subscribe(read.Apply)
	events.Subscribe(func(e DomainEvent) {
		evt := BroadcastEvent{User: e.User, Time: e.OccurredAt}
		switch e.Type {
		case "TodoCreated":
			var p TodoCreatedPayload
			_ = json.Unmarshal(e.Payload, &p)
			todo := Todo{ID: p.ID, Title: p.Title, CreatedAt: time.Now()}
			evt.Kind = "todo_created"
			evt.Data = renderTodo(todo)
		case "TodoToggled":
			evt.Kind = "todo_updated"
			t := findTodoByID(read, e.AggregateID)
			evt.Data = renderTodo(t)
		case "TodoUpdated":
			evt.Kind = "todo_updated"
			t := findTodoByID(read, e.AggregateID)
			evt.Data = renderTodo(t)
		case "TodoDeleted":
			evt.Kind = "todo_deleted"
			evt.Data = "#todo-" + e.AggregateID
		}
		broadcast.Send(evt)
	})

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
	_ = command.RegisterTyped(c.Commands, "CreateTodo", func(ctx context.Context, cmd *CreateTodoCmd) error {
		todoID := cmd.AggregateID().String()
		payload, _ := json.Marshal(TodoCreatedPayload{
			ID:        todoID,
			Title:     cmd.Title,
			CreatedAt: time.Now().Format(time.RFC3339),
		})

		c.appendDomainEvent(ctx, todoID, "TodoCreated", payload)
		return nil
	})

	_ = command.RegisterTyped(c.Commands, "ToggleTodo", func(ctx context.Context, cmd *ToggleTodoCmd) error {
		todoID := cmd.AggregateID().String()
		payload, _ := json.Marshal(TodoToggledPayload{ID: todoID})

		c.appendDomainEvent(ctx, todoID, "TodoToggled", payload)
		return nil
	})

	_ = command.RegisterTyped(c.Commands, "DeleteTodo", func(ctx context.Context, cmd *DeleteTodoCmd) error {
		todoID := cmd.AggregateID().String()
		payload, _ := json.Marshal(TodoDeletedPayload{ID: todoID})

		c.appendDomainEvent(ctx, todoID, "TodoDeleted", payload)
		return nil
	})

	_ = command.RegisterTyped(c.Commands, "UpdateTodo", func(ctx context.Context, cmd *UpdateTodoCmd) error {
		todoID := cmd.AggregateID().String()
		payload, _ := json.Marshal(TodoUpdatedPayload{ID: todoID, Title: cmd.Title})

		c.appendDomainEvent(ctx, todoID, "TodoUpdated", payload)
		return nil
	})
}

func (c *CQRS) registerQueryHandlers() {
	_ = query.RegisterTyped(c.Queries, "ListTodos", func(ctx context.Context, q *ListTodosQry) ([]Todo, error) {
		return c.Read.List(), nil
	})

	_ = query.RegisterTyped(c.Queries, "GetStats", func(_ context.Context, _ *GetStatsQry) (Stats, error) {
		total, active, completed := c.Read.Stats()
		return Stats{Total: total, Active: active, Completed: completed}, nil
	})
}
