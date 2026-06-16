package usermgmt

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
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

//nolint:gocognit // inherent to 7-event switch; each case is simple decode+mutate
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
			ID:          NewUserID(aggID.String()),
			Email:       p.Email,
			DisplayName: p.DisplayName,
			Roles:       roles,
			CreatedAt:   evt.OccurredAt(),
			UpdatedAt:   evt.OccurredAt(),
		}
		m.emails[p.Email] = aggID

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
			delete(m.emails, u.Email)
			u.Email = p.Email
			u.EmailVerified = false
			u.UpdatedAt = evt.OccurredAt()
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

	case eventCredentialAdded:
		p, err := event.DecodePayload[CredentialAddedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode CredentialAdded in read model: %w", err)
		}
		if u, ok := m.users[aggID]; ok {
			u.Credentials = append(u.Credentials, WebAuthnCredential{
				ID:              p.ID,
				PublicKey:       p.PublicKey,
				AttestationType: p.AttestationType,
				Transports:      append([]string(nil), p.Transports...),
				AAGUID:          append([]byte(nil), p.AAGUID...),
				SignCount:       p.SignCount,
				BackupEligible:  p.BackupEligible,
				BackupState:     p.BackupState,
				Name:            p.Name,
				CreatedAt:       evt.OccurredAt(),
			})
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventCredentialRemoved:
		p, err := event.DecodePayload[CredentialRemovedPayload](evt, c)
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
		p, err := event.DecodePayload[TOTPEnabledPayload](evt, c)
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
