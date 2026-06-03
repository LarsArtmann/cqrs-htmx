package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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

// --- Queries ---

type ListTodosQry struct {
	*query.BasicQuery
}

// --- Event Store (in-memory) ---

type EventStore struct {
	mu     sync.Mutex
	events []DomainEvent
	subs   []func(DomainEvent)
}

func NewEventStore() *EventStore {
	return &EventStore{}
}

func (s *EventStore) Append(event DomainEvent) {
	s.mu.Lock()
	s.events = append(s.events, event)
	subs := make([]func(DomainEvent), len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	for _, fn := range subs {
		fn(event)
	}
}

func (s *EventStore) All() []DomainEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DomainEvent, len(s.events))
	copy(out, s.events)
	return out
}

func (s *EventStore) Subscribe(fn func(DomainEvent)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs = append(s.subs, fn)
}

// --- Projector (read model from events) ---

type Projector struct {
	mu    sync.Mutex
	todos map[string]*Todo
	order []string
}

func NewProjector() *Projector {
	return &Projector{
		todos: make(map[string]*Todo),
	}
}

func (p *Projector) Apply(event DomainEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch event.Type {
	case "TodoCreated":
		var payload TodoCreatedPayload
		_ = json.Unmarshal(event.Payload, &payload)
		createdAt, _ := time.Parse(time.RFC3339, payload.CreatedAt)
		p.todos[payload.ID] = &Todo{
			ID:        payload.ID,
			Title:     payload.Title,
			Completed: false,
			CreatedAt: createdAt,
		}
		p.order = append(p.order, payload.ID)

	case "TodoToggled":
		var payload TodoToggledPayload
		_ = json.Unmarshal(event.Payload, &payload)
		if todo, ok := p.todos[payload.ID]; ok {
			todo.Completed = !todo.Completed
		}

	case "TodoDeleted":
		var payload TodoDeletedPayload
		_ = json.Unmarshal(event.Payload, &payload)
		delete(p.todos, payload.ID)
		newOrder := make([]string, 0, len(p.order))
		for _, id := range p.order {
			if id != payload.ID {
				newOrder = append(newOrder, id)
			}
		}
		p.order = newOrder
	}
}

func (p *Projector) List() []Todo {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]Todo, 0, len(p.order))
	for _, id := range p.order {
		if todo, ok := p.todos[id]; ok {
			result = append(result, *todo)
		}
	}
	return result
}

func (p *Projector) Stats() (total, active, completed int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	total = len(p.todos)
	for _, todo := range p.todos {
		if todo.Completed {
			completed++
		} else {
			active++
		}
	}
	return total, active, completed
}

func (p *Projector) GetByID(id string) (Todo, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	t, ok := p.todos[id]
	if !ok {
		return Todo{}, false
	}
	return *t, true
}

func findTodoByID(p *Projector, id string) Todo {
	t, ok := p.GetByID(id)
	if !ok {
		return Todo{ID: id}
	}
	return t
}

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

func (c *CQRS) registerCommandHandlers() {
	_ = c.Commands.Register("CreateTodo", func(ctx context.Context, cmd command.Command) error {
		tc, ok := cmd.(*CreateTodoCmd)
		if !ok {
			return fmt.Errorf("unexpected command type: %T", cmd)
		}

		todoID := cmd.AggregateID().String()
		payload, _ := json.Marshal(TodoCreatedPayload{
			ID:        todoID,
			Title:     tc.Title,
			CreatedAt: time.Now().Format(time.RFC3339),
		})

		c.Events.Append(DomainEvent{
			AggregateID: todoID,
			Type:        "TodoCreated",
			User:        userName(ctx),
			Payload:     payload,
			OccurredAt:  time.Now(),
		})
		return nil
	})

	_ = c.Commands.Register("ToggleTodo", func(ctx context.Context, cmd command.Command) error {
		todoID := cmd.AggregateID().String()
		payload, _ := json.Marshal(TodoToggledPayload{ID: todoID})

		c.Events.Append(DomainEvent{
			AggregateID: todoID,
			Type:        "TodoToggled",
			User:        userName(ctx),
			Payload:     payload,
			OccurredAt:  time.Now(),
		})
		return nil
	})

	_ = c.Commands.Register("DeleteTodo", func(ctx context.Context, cmd command.Command) error {
		todoID := cmd.AggregateID().String()
		payload, _ := json.Marshal(TodoDeletedPayload{ID: todoID})

		c.Events.Append(DomainEvent{
			AggregateID: todoID,
			Type:        "TodoDeleted",
			User:        userName(ctx),
			Payload:     payload,
			OccurredAt:  time.Now(),
		})
		return nil
	})
}

func (c *CQRS) registerQueryHandlers() {
	_ = c.Queries.Register("ListTodos", func(ctx context.Context, q query.Query) (any, error) {
		return c.Read.List(), nil
	})

	_ = c.Queries.Register("GetStats", func(ctx context.Context, q query.Query) (any, error) {
		total, active, completed := c.Read.Stats()
		return map[string]int{
			"total":     total,
			"active":    active,
			"completed": completed,
		}, nil
	})
}

// --- Command Constructors ---

func NewCreateTodo(title string) (*CreateTodoCmd, error) {
	aggID := id.NewAggregateID()
	core, err := command.New("CreateTodo", aggID)
	if err != nil {
		return nil, err
	}
	return &CreateTodoCmd{BasicCommand: core, Title: title}, nil
}

func NewToggleTodo(todoID string) (*ToggleTodoCmd, error) {
	aggID, err := id.ParseAggregateID(todoID)
	if err != nil {
		return nil, fmt.Errorf("invalid todo ID %q: %w", todoID, err)
	}
	core, err := command.New("ToggleTodo", aggID)
	if err != nil {
		return nil, fmt.Errorf("create ToggleTodo command for todo %q: %w", todoID, err)
	}
	return &ToggleTodoCmd{BasicCommand: core}, nil
}

func NewDeleteTodo(todoID string) (*DeleteTodoCmd, error) {
	aggID, err := id.ParseAggregateID(todoID)
	if err != nil {
		return nil, fmt.Errorf("invalid todo ID %q: %w", todoID, err)
	}
	core, err := command.New("DeleteTodo", aggID)
	if err != nil {
		return nil, fmt.Errorf("create DeleteTodo command for todo %q: %w", todoID, err)
	}
	return &DeleteTodoCmd{BasicCommand: core}, nil
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
