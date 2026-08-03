package main

import (
	"encoding/json/v2"
	"sync"
	"time"
)

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
	switch event.Type {
	case "TodoCreated":
		var payload TodoCreatedPayload
		_ = json.Unmarshal(event.Payload, &payload)
		createdAt, _ := time.Parse(time.RFC3339, payload.CreatedAt)

		p.mu.Lock()
		defer p.mu.Unlock()
		p.todos[payload.ID] = &Todo{
			ID:        payload.ID,
			Title:     payload.Title,
			Completed: false,
			CreatedAt: createdAt,
		}
		p.order = append(p.order, payload.ID)

	case "TodoToggled":
		var payload TodoToggledPayload
		//cqrs-lint:ignore(C021) decode is outside the lock below; blank line separates them visually
		_ = json.Unmarshal(event.Payload, &payload)

		p.mu.Lock()
		defer p.mu.Unlock()
		if todo, ok := p.todos[payload.ID]; ok {
			todo.Completed = !todo.Completed
		}

	case "TodoDeleted":
		var payload TodoDeletedPayload
		//cqrs-lint:ignore(C021) decode is outside the lock below; blank line separates them visually
		_ = json.Unmarshal(event.Payload, &payload)

		p.mu.Lock()
		defer p.mu.Unlock()
		delete(p.todos, payload.ID)
		newOrder := make([]string, 0, len(p.order))
		for _, id := range p.order {
			if id != payload.ID {
				newOrder = append(newOrder, id)
			}
		}
		p.order = newOrder

	case "TodoUpdated":
		var payload TodoUpdatedPayload
		//cqrs-lint:ignore(C021) decode is outside the lock below; blank line separates them visually
		_ = json.Unmarshal(event.Payload, &payload)

		p.mu.Lock()
		defer p.mu.Unlock()
		if todo, ok := p.todos[payload.ID]; ok {
			todo.Title = payload.Title
		}
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

// --- CQRS Setup ---
