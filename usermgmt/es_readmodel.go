package usermgmt

import (
	"bytes"
	"context"
	"sort"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// UserReadModel is the projection-side store for users.
type UserReadModel struct {
	mu               sync.RWMutex
	users            map[id.AggregateID]*User
	emails           map[string]id.AggregateID
	externalAccounts map[externalAccountKey]id.AggregateID
}

// externalAccountKey is the composite key for the global provider+subject → user index.
// Ensures a given provider subject can only be linked to one user at a time.
type externalAccountKey struct {
	provider string
	subject  string
}

func NewUserReadModel() *UserReadModel {
	return &UserReadModel{
		users:            make(map[id.AggregateID]*User),
		emails:           make(map[string]id.AggregateID),
		externalAccounts: make(map[externalAccountKey]id.AggregateID),
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
			return event.WrapCorruption(err, "usermgmt.readmodel.decode_failed", "decode UserRegistered in read model")
		}
		m.users[aggID] = &User{
			ID:          NewUserID(aggID.String()),
			Email:       p.Email,
			DisplayName: p.DisplayName,
			CreatedAt:   evt.OccurredAt(),
			UpdatedAt:   evt.OccurredAt(),
		}
		m.emails[p.Email] = aggID

	case eventRolesUpdated:
		// Roles are managed by the Membership aggregate. We still decode the
		// payload to detect corruption, but no longer store roles on User.
		_, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return event.WrapCorruption(err, "usermgmt.readmodel.decode_failed", "decode RolesUpdated in read model")
		}

	case eventEmailChanged:
		p, err := unmarshalPayload[EmailChangedPayload](evt)
		if err != nil {
			return event.WrapCorruption(err, "usermgmt.readmodel.decode_failed", "decode EmailChanged in read model")
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
			return event.WrapCorruption(
				err,
				"usermgmt.readmodel.decode_failed",
				"decode DisplayNameChanged in read model",
			)
		}
		if u, ok := m.users[aggID]; ok {
			u.DisplayName = p.DisplayName
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventCredentialAdded:
		p, err := unmarshalPayload[CredentialAddedPayload](evt)
		if err != nil {
			return event.WrapCorruption(err, "usermgmt.readmodel.decode_failed", "decode CredentialAdded in read model")
		}
		if u, ok := m.users[aggID]; ok {
			u.Credentials = append(u.Credentials, newCredentialFromPayload(p, evt.OccurredAt()))
			u.UpdatedAt = evt.OccurredAt()
		}

	case eventCredentialRemoved:
		p, err := unmarshalPayload[CredentialRemovedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.readmodel.decode_failed",
				"decode CredentialRemoved in read model",
			)
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
			for _, ea := range u.ExternalAccounts {
				delete(m.externalAccounts, externalAccountKey{provider: ea.Provider, subject: ea.Subject})
			}
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
			return event.WrapCorruption(err, "usermgmt.readmodel.decode_failed", "decode TOTPEnabled in read model")
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
			return event.WrapCorruption(
				err,
				"usermgmt.readmodel.decode_failed",
				"decode ExternalAccountLinked in read model",
			)
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
			m.externalAccounts[externalAccountKey{provider: p.Provider, subject: p.Subject}] = aggID
		}

	case eventExternalAccountUnlinked:
		p, err := unmarshalPayload[ExternalAccountUnlinkedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.readmodel.decode_failed",
				"decode ExternalAccountUnlinked in read model",
			)
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
			delete(m.externalAccounts, externalAccountKey{provider: p.Provider, subject: p.Subject})
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

// FindByExternalAccount looks up a user by their linked provider+subject pair.
// Returns the user and true if found. This enforces global uniqueness: a given
// provider subject can only be linked to one user at a time.
func (m *UserReadModel) FindByExternalAccount(provider, subject string) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	aggID, ok := m.externalAccounts[externalAccountKey{provider: provider, subject: subject}]
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

// aggIDFromBranded converts any branded string ID to an AggregateID.
// Shared by aggIDFromUser, aggIDFromTenant, aggIDFromBot — eliminates
// triplicated ParseAggregateID + Wrapf boilerplate.
func aggIDFromBranded(raw, sentinel string) (id.AggregateID, error) {
	aggID, err := id.ParseAggregateID(raw)
	if err != nil {
		return id.AggregateID{}, event.Wrapf(
			err,
			event.Infrastructure,
			sentinel,
			"invalid branded ID for AggregateID conversion",
		)
	}
	return aggID, nil
}

func aggIDFromUser(userID UserID) (id.AggregateID, error) {
	return aggIDFromBranded(userID.Get(), "usermgmt.readmodel.invalid_userid")
}

func nowUTC() time.Time { return time.Now().UTC() }
