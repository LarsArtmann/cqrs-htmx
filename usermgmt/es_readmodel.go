package usermgmt

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// UserReadModel is the projection-side store for users.
type UserReadModel struct {
	mu     sync.RWMutex
	users  map[id.AggregateID]*User
	emails map[string]id.AggregateID
}

func NewUserReadModel() *UserReadModel {
	return &UserReadModel{
		users:  make(map[id.AggregateID]*User),
		emails: make(map[string]id.AggregateID),
	}
}

func (m *UserReadModel) Name() string { return "user-read-model" }

func (m *UserReadModel) EventTypes() []event.Type { return allUserEventTypes }

//nolint:gocognit,gocyclo // inherent to 12-event switch; each case is simple decode+mutate
func (m *UserReadModel) Handle(_ context.Context, evt event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	aggID := evt.AggregateID()

	switch evt.Type() {
	case eventUserRegistered:
		p, err := unmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return fmt.Errorf("decode UserRegistered in read model: %w", err)
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		m.users[aggID] = &User{
			ID:          NewUserID(aggID.String()),
			Email:       p.Email,
			DisplayName: p.DisplayName,
			Roles:       roles,
			CreatedAt:   evt.OccurredAt(),
			UpdatedAt:   evt.OccurredAt(),
		}
		m.emails[p.Email] = aggID

	case eventRolesUpdated:
		p, err := unmarshalPayload[RolesUpdatedPayload](evt)
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
		p, err := unmarshalPayload[EmailChangedPayload](evt)
		if err != nil {
			return fmt.Errorf("decode EmailChanged in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			delete(m.emails, u.Email)
			u.Email = p.Email
			u.EmailVerified = false
			u.UpdatedAt = evt.OccurredAt()
			m.emails[p.Email] = aggID
		}

	case eventDisplayNameChanged:
		p, err := unmarshalPayload[DisplayNameChangedPayload](evt)
		if err != nil {
			return fmt.Errorf("decode DisplayNameChanged in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			u.DisplayName = p.DisplayName
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventCredentialAdded:
		p, err := unmarshalPayload[CredentialAddedPayload](evt)
		if err != nil {
			return fmt.Errorf("decode CredentialAdded in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			u.Credentials = append(u.Credentials, newCredentialFromPayload(p, evt.OccurredAt()))
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventCredentialRemoved:
		p, err := unmarshalPayload[CredentialRemovedPayload](evt)
		if err != nil {
			return fmt.Errorf("decode CredentialRemoved in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			filtered := u.Credentials[:0]
			for _, cred := range u.Credentials {
				if !bytes.Equal(cred.ID, p.ID) {
					filtered = append(filtered, cred)
				}
			}
			u.Credentials = filtered
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventUserDeleted:
		if u, ok := m.users[aggID]; ok {
			delete(m.emails, u.Email)
		}
		delete(m.users, aggID)

	case eventEmailVerified:
		if u, ok := m.users[aggID]; ok {
			u.EmailVerified = true
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventTOTPEnabled:
		p, err := unmarshalPayload[TOTPEnabledPayload](evt)
		if err != nil {
			return fmt.Errorf("decode TOTPEnabled in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			u.TOTPEnabled = true
			u.TOTPSecret = p.Secret
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventTOTPDisabled:
		if u, ok := m.users[aggID]; ok {
			u.TOTPEnabled = false
			u.TOTPSecret = nil
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventExternalAccountLinked:
		p, err := unmarshalPayload[ExternalAccountLinkedPayload](evt)
		if err != nil {
			return fmt.Errorf("decode ExternalAccountLinked in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			u.ExternalAccounts = append(u.ExternalAccounts, ExternalAccount{
				Provider:    p.Provider,
				Subject:     p.Subject,
				Email:       p.Email,
				DisplayName: p.DisplayName,
				LinkedAt:    evt.OccurredAt(),
			})
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventExternalAccountUnlinked:
		p, err := unmarshalPayload[ExternalAccountUnlinkedPayload](evt)
		if err != nil {
			return fmt.Errorf("decode ExternalAccountUnlinked in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			filtered := u.ExternalAccounts[:0]
			for _, ea := range u.ExternalAccounts {
				if ea.Provider != p.Provider || ea.Subject != p.Subject {
					filtered = append(filtered, ea)
				}
			}
			u.ExternalAccounts = filtered
			u.UpdatedAt = evt.OccurredAt()
		}

	default:
	}

	return nil
}

func (m *UserReadModel) FindByID(aggID id.AggregateID) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[aggID]
	if !ok {
		return nil, false
	}
	return u.Clone(), true
}

func (m *UserReadModel) FindByEmail(email string) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	aggID, ok := m.emails[email]
	if !ok {
		return nil, false
	}
	return m.users[aggID].Clone(), true
}

func (m *UserReadModel) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}

// AllUsers returns a deep-copied slice of all users in the read model.
// The result is sorted by CreatedAt ascending, then by ID, for deterministic output.
func (m *UserReadModel) AllUsers() []*User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	users := make([]*User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u.Clone())
	}
	sort.Slice(users, func(i, j int) bool {
		if !users[i].CreatedAt.Equal(users[j].CreatedAt) {
			return users[i].CreatedAt.Before(users[j].CreatedAt)
		}
		return users[i].ID.Get() < users[j].ID.Get()
	})
	return users
}

func (m *UserReadModel) FindByUserID(userID UserID) (*User, bool) {
	aggID, err := id.ParseAggregateID(userID.Get())
	if err != nil {
		return nil, false
	}
	return m.FindByID(aggID)
}

var _ event.Projection = (*UserReadModel)(nil)

func aggIDFromUser(userID UserID) (id.AggregateID, error) {
	aggID, err := id.ParseAggregateID(userID.Get())
	if err != nil {
		return id.AggregateID{}, fmt.Errorf("invalid UserID for AggregateID conversion: %w", err)
	}
	return aggID, nil
}

func nowUTC() time.Time { return time.Now().UTC() }
