package usermgmt

import (
	"bytes"
	"context"
	"sort"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// UserReadModel is the projection-side store for users.
//cqrs-lint:ignore(C035) protected by embedded readModelCore.mu (sync.RWMutex)
type UserReadModel struct {
	readModelCore[*UserReadModel]
	//cqrs-lint:ignore(P011) bounded by finite user count; in-memory dev/test default
	users            map[id.StreamID]*User
	//cqrs-lint:ignore(P011) bounded by finite user count; in-memory dev/test default
	emails           map[string]id.StreamID
	//cqrs-lint:ignore(P011) bounded by finite user count; in-memory dev/test default
	externalAccounts map[externalAccountKey]id.StreamID
}

// externalAccountKey is the composite key for the global provider+subject → user index.
// Ensures a given provider subject can only be linked to one user at a time.
type externalAccountKey struct {
	provider string
	subject  string
}

func NewUserReadModel() *UserReadModel {
	return &UserReadModel{
		readModelCore: readModelCore[*UserReadModel]{
			handlers: map[event.Type]eventHandler[*UserReadModel]{
				eventUserRegistered:          (*UserReadModel).handleUserRegistered,
				eventRolesUpdated:            (*UserReadModel).handleRolesUpdated,
				eventEmailChanged:            (*UserReadModel).handleEmailChanged,
				eventDisplayNameChanged:      (*UserReadModel).handleDisplayNameChanged,
				eventCredentialAdded:         (*UserReadModel).handleCredentialAdded,
				eventCredentialRemoved:       (*UserReadModel).handleCredentialRemoved,
				eventUserDeleted:             (*UserReadModel).handleUserDeleted,
				eventEmailVerified:           (*UserReadModel).handleEmailVerified,
				eventTOTPEnabled:             (*UserReadModel).handleTOTPEnabled,
				eventTOTPDisabled:            (*UserReadModel).handleTOTPDisabled,
				eventExternalAccountLinked:   (*UserReadModel).handleExternalAccountLinked,
				eventExternalAccountUnlinked: (*UserReadModel).handleExternalAccountUnlinked,
			},
		},
		users:            make(map[id.StreamID]*User),
		emails:           make(map[string]id.StreamID),
		externalAccounts: make(map[externalAccountKey]id.StreamID),
	}
}

func (m *UserReadModel) Name() string { return "user-read-model" }

func (m *UserReadModel) EventTypes() []event.Type { return allUserEventTypes }

func (m *UserReadModel) Handle(_ context.Context, evt event.Event) error {
	return m.handleEvent(m, evt)
}

// decodePayload unmarshals an event payload of type T, wrapping decode failures
// as Corruption errors with the event name for diagnostics.
func decodePayload[T any](evt event.Event, name string) (T, error) {
	p, err := unmarshalPayload[T](evt)
	if err != nil {
		var zero T
		return zero, errorfamily.WrapCorruption(
			err,
			"usermgmt.readmodel.decode_failed",
			"decode "+name+" in read model",
		)
	}
	return p, nil
}

func (m *UserReadModel) handleUserRegistered(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[UserRegisteredPayload](evt, "UserRegistered")
	if err != nil {
		return err
	}
	m.users[aggID] = &User{
		ID:          NewUserID(aggID.String()),
		Email:       p.Email,
		DisplayName: p.DisplayName,
		CreatedAt:   evt.OccurredAt(),
		UpdatedAt:   evt.OccurredAt(),
	}
	m.emails[p.Email] = aggID
	return nil
}

func (m *UserReadModel) handleRolesUpdated(_ id.StreamID, evt event.Event) error {
	_, err := decodePayload[RolesUpdatedPayload](evt, "RolesUpdated")
	return err
}

func (m *UserReadModel) handleEmailChanged(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[EmailChangedPayload](evt, "EmailChanged")
	if err != nil {
		return err
	}
	if u, ok := m.users[aggID]; ok {
		delete(m.emails, u.Email)
		u.Email = p.Email
		u.EmailVerified = false
		u.UpdatedAt = evt.OccurredAt()
		m.emails[p.Email] = aggID
	}
	return nil
}

func (m *UserReadModel) handleDisplayNameChanged(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[DisplayNameChangedPayload](evt, "DisplayNameChanged")
	if err != nil {
		return err
	}
	if u, ok := m.users[aggID]; ok {
		u.DisplayName = p.DisplayName
		u.UpdatedAt = evt.OccurredAt()
	}
	return nil
}

func (m *UserReadModel) handleCredentialAdded(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[CredentialAddedPayload](evt, "CredentialAdded")
	if err != nil {
		return err
	}
	if u, ok := m.users[aggID]; ok {
		u.Credentials = append(u.Credentials, NewCredentialFromPayload(p, evt.OccurredAt()))
		u.UpdatedAt = evt.OccurredAt()
	}
	return nil
}

func (m *UserReadModel) handleCredentialRemoved(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[CredentialRemovedPayload](evt, "CredentialRemoved")
	if err != nil {
		return err
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
	return nil
}

func (m *UserReadModel) handleUserDeleted(aggID id.StreamID, _ event.Event) error {
	if u, ok := m.users[aggID]; ok {
		delete(m.emails, u.Email)
		for _, ea := range u.ExternalAccounts {
			delete(m.externalAccounts, externalAccountKey{provider: ea.Provider, subject: ea.Subject})
		}
	}
	delete(m.users, aggID)
	return nil
}

func (m *UserReadModel) handleEmailVerified(aggID id.StreamID, evt event.Event) error {
	if u, ok := m.users[aggID]; ok {
		u.EmailVerified = true
		u.UpdatedAt = evt.OccurredAt()
	}
	return nil
}

func (m *UserReadModel) handleTOTPEnabled(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[TOTPEnabledPayload](evt, "TOTPEnabled")
	if err != nil {
		return err
	}
	if u, ok := m.users[aggID]; ok {
		u.TOTPEnabled = true
		u.TOTPSecret = p.Secret
		u.UpdatedAt = evt.OccurredAt()
	}
	return nil
}

func (m *UserReadModel) handleTOTPDisabled(aggID id.StreamID, evt event.Event) error {
	if u, ok := m.users[aggID]; ok {
		u.TOTPEnabled = false
		u.TOTPSecret = nil
		u.UpdatedAt = evt.OccurredAt()
	}
	return nil
}

func (m *UserReadModel) handleExternalAccountLinked(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[ExternalAccountLinkedPayload](evt, "ExternalAccountLinked")
	if err != nil {
		return err
	}
	if u, ok := m.users[aggID]; ok {
		u.ExternalAccounts = append(u.ExternalAccounts, ExternalAccount{
			ExternalAccountCore: ExternalAccountCore{
				Provider:    p.Provider,
				Subject:     p.Subject,
				Email:       p.Email,
				DisplayName: p.DisplayName,
			},
			LinkedAt: evt.OccurredAt(),
		})
		u.UpdatedAt = evt.OccurredAt()
		m.externalAccounts[externalAccountKey{provider: p.Provider, subject: p.Subject}] = aggID
	}
	return nil
}

func (m *UserReadModel) handleExternalAccountUnlinked(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[ExternalAccountUnlinkedPayload](evt, "ExternalAccountUnlinked")
	if err != nil {
		return err
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
	return nil
}

func (m *UserReadModel) FindByID(aggID id.StreamID) (*User, bool) {
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
		return users[i].ID.Get().String() < users[j].ID.Get().String()
	})
	return users
}

func (m *UserReadModel) FindByUserID(userID UserID) (*User, bool) {
	aggID, err := id.ParseStreamID(userID.Get().String())
	if err != nil {
		return nil, false
	}
	return m.FindByID(aggID)
}

var _ projection.Projection = (*UserReadModel)(nil)

// aggIDFromBranded converts any branded string ID to an AggregateID.
// Shared by aggIDFromUser, aggIDFromTenant, aggIDFromBot — eliminates
// triplicated ParseAggregateID + Wrapf boilerplate.
func aggIDFromBranded(raw, sentinel string) (id.StreamID, error) {
	aggID, err := id.ParseStreamID(raw)
	if err != nil {
		return id.StreamID{}, errorfamily.Wrapf(
			err,
			event.Infrastructure,
			sentinel,
			"invalid branded ID for AggregateID conversion",
		)
	}
	return aggID, nil
}

func aggIDFromUser(userID UserID) (id.StreamID, error) {
	return aggIDFromBranded(userID.Get().String(), "usermgmt.readmodel.invalid_userid")
}

func nowUTC() time.Time { return time.Now().UTC() }
