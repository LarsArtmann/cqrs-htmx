package usermgmt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// UserReadModel is the projection-side store for users. It maintains an
// in-memory map of users keyed by aggregate ID, with an email index for
// O(1) lookup by email.
type UserReadModel struct {
	mu     sync.RWMutex
	users  map[id.AggregateID]*User
	emails map[string]id.AggregateID
}

// NewUserReadModel creates an empty UserReadModel.
func NewUserReadModel() *UserReadModel {
	return &UserReadModel{
		users:  make(map[id.AggregateID]*User),
		emails: make(map[string]id.AggregateID),
	}
}

func (m *UserReadModel) Name() string { return "user-read-model" }

func (m *UserReadModel) EventTypes() []event.Type {
	return allUserEventTypes
}

func (m *UserReadModel) Handle(_ context.Context, evt event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	aggID := evt.AggregateID()
	c := codec.JSONCodec{}

	switch evt.Type() {
	case eventUserRegistered:
		p, err := event.DecodePayload[UserRegisteredPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode UserRegistered in read model: %w", err)
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		m.users[aggID] = &User{
			ID:           NewUserID(aggID.String()),
			Email:        p.Email,
			DisplayName:  p.DisplayName,
			PasswordHash: p.PasswordHash,
			Roles:        roles,
			CreatedAt:    evt.OccurredAt(),
			UpdatedAt:    evt.OccurredAt(),
		}
		m.emails[p.Email] = aggID

	case eventPasswordChanged:
		p, err := event.DecodePayload[PasswordChangedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode PasswordChanged in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			u.PasswordHash = p.PasswordHash
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventRolesUpdated:
		p, err := event.DecodePayload[RolesUpdatedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode RolesUpdated in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			roles := make([]Role, len(p.Roles))
			copy(roles, p.Roles)
			u.Roles = roles
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventEmailChanged:
		p, err := event.DecodePayload[EmailChangedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode EmailChanged in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			oldEmail := u.Email
			u.Email = p.Email
			u.UpdatedAt = evt.OccurredAt()
			delete(m.emails, oldEmail)
			m.emails[p.Email] = aggID
		}

	case eventDisplayNameChanged:
		p, err := event.DecodePayload[DisplayNameChangedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode DisplayNameChanged in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			u.DisplayName = p.DisplayName
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventUserDeleted:
		if u, ok := m.users[aggID]; ok {
			delete(m.emails, u.Email)
		}
		delete(m.users, aggID)

	default:
		// Ignore events we don't handle
	}

	return nil
}

// FindByID returns a clone of the user with the given aggregate ID, or nil.
func (m *UserReadModel) FindByID(aggID id.AggregateID) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[aggID]
	if !ok {
		return nil, false
	}
	return u.Clone(), true
}

// FindByEmail returns a clone of the user with the given email, or nil.
func (m *UserReadModel) FindByEmail(email string) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	aggID, ok := m.emails[email]
	if !ok {
		return nil, false
	}
	u := m.users[aggID]
	return u.Clone(), true
}

// Count returns the number of users in the read model.
func (m *UserReadModel) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}

// FindByUserID converts a usermgmt.UserID to an AggregateID and looks up the user.
func (m *UserReadModel) FindByUserID(userID UserID) (*User, bool) {
	aggID, err := id.ParseAggregateID(userID.Get())
	if err != nil {
		return nil, false
	}
	return m.FindByID(aggID)
}

var _ event.Projection = (*UserReadModel)(nil)

// aggIDFromUser converts a UserID to an AggregateID. Panics if the UserID is empty.
func aggIDFromUser(userID UserID) id.AggregateID {
	aggID, err := id.ParseAggregateID(userID.Get())
	if err != nil {
		panic("usermgmt: invalid UserID for AggregateID conversion: " + err.Error())
	}
	return aggID
}

// nowUTC returns the current time in UTC.
func nowUTC() time.Time { return time.Now().UTC() }
