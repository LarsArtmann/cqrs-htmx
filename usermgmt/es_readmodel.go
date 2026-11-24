package usermgmt

import (
	"bytes"
	"context"
	"slices"
	"sort"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// UserReadModel is the projection-side store for users.
// cqrs-lint:ignore(C035) protected by embedded readModelCore.mu (sync.RWMutex)
type UserReadModel struct {
	readModelCore[*UserReadModel]
	//cqrs-lint:ignore(P011) bounded by finite user count; in-memory dev/test default
	users map[id.StreamID]*User
	//cqrs-lint:ignore(P011) bounded by finite user count; in-memory dev/test default
	emails map[string]id.StreamID
	//cqrs-lint:ignore(P011) bounded by finite user count; in-memory dev/test default
	externalAccounts map[externalAccountKey]id.StreamID
	// mergedInto maps each merged-away user's stream to the survivor it points
	// at. Merged-away rows leave `users`, so this is the only trace they keep.
	//cqrs-lint:ignore(P011) bounded by finite user count; in-memory dev/test default
	mergedInto map[id.StreamID]id.StreamID
}

// externalAccountKey is the composite key for the global provider+subject → user index.
// Ensures a given provider subject can only be linked to one user at a time.
type externalAccountKey struct {
	provider string
	subject  string
}

func NewUserReadModel() *UserReadModel {
	return &UserReadModel{
		drop: (*UserReadModel).dropOnRetiredStream,
		handlers: map[event.Type]eventHandler[*UserReadModel]{
			eventUserRegistered:          (*UserReadModel).handleUserRegistered,
			eventRolesUpdated:            (*UserReadModel).handleRolesUpdated,
			eventEmailChanged:            (*UserReadModel).handleEmailChanged,
			eventDisplayNameChanged:      (*UserReadModel).handleDisplayNameChanged,
			eventCredentialAdded:         (*UserReadModel).handleCredentialAdded,
			eventCredentialRemoved:       (*UserReadModel).handleCredentialRemoved,
			eventCredentialUsed:          (*UserReadModel).handleCredentialUsed,
			eventUserDeleted:             (*UserReadModel).handleUserDeleted,
			eventEmailVerified:           (*UserReadModel).handleEmailVerified,
			eventTOTPEnabled:             (*UserReadModel).handleTOTPEnabled,
			eventTOTPDisabled:            (*UserReadModel).handleTOTPDisabled,
			eventExternalAccountLinked:   (*UserReadModel).handleExternalAccountLinked,
			eventExternalAccountUnlinked: (*UserReadModel).handleExternalAccountUnlinked,
			eventUserMerged:              (*UserReadModel).handleUserMerged,
			eventUserMergedInto:          (*UserReadModel).handleUserMergedInto,
		},
		users:            make(map[id.StreamID]*User),
		emails:           make(map[string]id.StreamID),
		externalAccounts: make(map[externalAccountKey]id.StreamID),
		mergedInto:       make(map[id.StreamID]id.StreamID),
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
		).WithContextAny("agg_id", evt.StreamID())
	}
	return p, nil
}

func (m *UserReadModel) handleUserRegistered(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[UserRegisteredPayload](evt, "UserRegistered")
	if err != nil {
		return err //nolint:erraudit // HELPER: decodePayload attaches agg_id
	}
	m.users[aggID] = &User{
		ID:          NewUserID(aggID.Get()),
		Email:       p.Email,
		DisplayName: p.DisplayName,
		CreatedAt:   evt.OccurredAt(),
		UpdatedAt:   evt.OccurredAt(),
	}
	// Keep-first: duplicate registrations sharing an email (historical dupes
	// exist in live DBs) must never steal the email mapping from the original
	// user — newest-wins silently re-points every login at an empty user
	// while the data stays attributed to the original.
	if _, exists := m.emails[p.Email]; !exists {
		m.emails[p.Email] = aggID
	}
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
		// Only evict when THIS user owns the mapping (keep-first: a newer
		// duplicate sharing the email must not remove the original's entry).
		if m.emails[u.Email] == aggID {
			delete(m.emails, u.Email)
		}
		u.Email = p.Email
		u.EmailVerified = false
		// Mirror FoldUser (identity-model/fold.go EventEmailChanged): an
		// email change resets TOTP. Without this the read model keeps
		// TOTPEnabled=true while the aggregate fold already disabled it,
		// leaving the user locked into an undisableable second factor
		// (DisableTOTP sees the fold's TOTPEnabled=false and emits nothing).
		u.TOTPEnabled = false
		u.TOTPSecret = nil
		u.UpdatedAt = evt.OccurredAt()
		if _, exists := m.emails[p.Email]; !exists {
			m.emails[p.Email] = aggID
		}
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
		// Skip when the credential ID is already present: after a hydrate the
		// drain may re-apply events whose SQL view write survived a crash but
		// whose checkpoint save did not, and appending blindly would duplicate
		// the credential.
		if u.HasCredential(p.ID) {
			return nil
		}
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

func (m *UserReadModel) handleCredentialUsed(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[CredentialUsedPayload](evt, "CredentialUsed")
	if err != nil {
		return err
	}
	if u, ok := m.users[aggID]; ok {
		if cred := findCredential(u.Credentials, p.CredentialID); cred != nil {
			*cred = cred.ApplyUse(CredentialUse{
				CredentialID: p.CredentialID,
				SignCount:    p.SignCount,
				UserPresent:  p.UserPresent,
				UserVerified: p.UserVerified,
				BackupState:  p.BackupState,
			})
			u.UpdatedAt = evt.OccurredAt()
		}
	}
	return nil
}

func (m *UserReadModel) handleUserDeleted(aggID id.StreamID, _ event.Event) error {
	// Guarded eviction: under keep-first, a deleted duplicate must not remove
	// the original user's email / external-account mapping. Only entries that
	// resolve to this stream go, including aliases it inherited from merges.
	evict(m.emails, aggID)
	evict(m.externalAccounts, aggID)
	delete(m.users, aggID)
	return nil
}

// dropOnRetiredStream keeps a merged-away stream retired. Its row is gone, so
// every handler but UserRegistered is already a no-op on it; the registration
// would build a new row, which is how an event re-delivered out of order
// (after a hydrate the drain replays events that precede the merge) could
// bring a merged identity back. UserMergedInto itself stays applicable so a
// re-delivered retirement lands as the no-op it is.
func (m *UserReadModel) dropOnRetiredStream(aggID id.StreamID, evt event.Event) bool {
	return evt.Type() != eventUserMergedInto && m.isRetired(aggID)
}

// isRetired reports whether the stream was merged away. Callers hold at least
// a read lock.
func (m *UserReadModel) isRetired(aggID id.StreamID) bool {
	_, retired := m.mergedInto[aggID]
	return retired
}

// Retired reports whether the stream was merged away into a survivor. Unlike
// the finders it does not follow the pointer. The SQL read model uses it to
// keep a retired stream out of its table.
func (m *UserReadModel) Retired(aggID id.StreamID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRetired(aggID)
}

// MergeTarget names the survivor a merged-away stream points at, without
// following the chain, and reports whether the stream is merged at all. A
// service that must tell a merged identity from a live one asks here; a
// lookup that wants the live user uses [UserReadModel.FindByID].
func (m *UserReadModel) MergeTarget(aggID id.StreamID) (id.StreamID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	survivor, merged := m.mergedInto[aggID]
	return survivor, merged
}

// handleUserMerged records on the survivor which identities it absorbed.
// The adopted assets arrive as ordinary lifecycle events before this one.
func (m *UserReadModel) handleUserMerged(_ id.StreamID, evt event.Event) error {
	aggID := evt.StreamID()
	p, err := decodePayload[identitymodel.UserMergedPayload](evt, "UserMerged")
	if err != nil {
		return err
	}
	u, ok := m.users[aggID]
	if !ok {
		return nil
	}
	for _, loser := range p.LoserIDs {
		if !slices.Contains(u.MergedUserIDs, loser) {
			u.MergedUserIDs = append(u.MergedUserIDs, loser)
		}
	}
	u.UpdatedAt = evt.OccurredAt()
	return nil
}

// handleUserMergedInto retires the stream the event lands on in favour of
// its survivor. Every index entry that resolved to the retired stream now
// resolves to the survivor, so a merged ID, email or external subject keeps
// answering with the surviving identity.
func (m *UserReadModel) handleUserMergedInto(aggID id.StreamID, evt event.Event) error {
	p, err := decodePayload[identitymodel.UserMergedIntoPayload](evt, "UserMergedInto")
	if err != nil {
		return err
	}
	survivor, err := aggIDFromUser(p.SurvivorID)
	if err != nil {
		return err
	}
	repoint(m.emails, aggID, survivor)
	repoint(m.externalAccounts, aggID, survivor)
	delete(m.users, aggID)
	m.mergedInto[aggID] = survivor
	return nil
}

// repoint moves every index entry owned by one stream to another.
func repoint[K comparable](index map[K]id.StreamID, from, to id.StreamID) {
	for key, owner := range index {
		if owner == from {
			index[key] = to
		}
	}
}

// evict removes every index entry owned by one stream.
func evict[K comparable](index map[K]id.StreamID, owner id.StreamID) {
	for key, candidate := range index {
		if candidate == owner {
			delete(index, key)
		}
	}
}

// mergeChainLimit bounds how many pointers one lookup follows: one per
// pointer in the map. A chain cannot be longer, so a walk that reaches the
// limit sits on a cycle.
func (m *UserReadModel) mergeChainLimit() int {
	return len(m.mergedInto) + 1
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
		// Skip an already-linked provider+subject pair: after a hydrate the
		// drain may re-apply events whose SQL view write survived a crash but
		// whose checkpoint save did not (at-least-once re-delivery).
		if !slices.ContainsFunc(u.ExternalAccounts, func(ea ExternalAccount) bool {
			return ea.Provider == p.Provider && ea.Subject == p.Subject
		}) {
			u.ExternalAccounts = append(u.ExternalAccounts, ExternalAccount{
				Provider:    p.Provider,
				Subject:     p.Subject,
				Email:       p.Email,
				DisplayName: p.DisplayName,
				LinkedAt:    evt.OccurredAt(),
			})
		}
		u.UpdatedAt = evt.OccurredAt()
		// Keep-first: a duplicate user linking the same provider+subject must
		// not steal the lookup from the original (one subject = one identity).
		key := externalAccountKey{provider: p.Provider, subject: p.Subject}
		if _, exists := m.externalAccounts[key]; !exists {
			m.externalAccounts[key] = aggID
		}
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
		key := externalAccountKey{provider: p.Provider, subject: p.Subject}
		if m.externalAccounts[key] == aggID {
			delete(m.externalAccounts, key)
		}
	}
	return nil
}

func (m *UserReadModel) FindByID(aggID id.StreamID) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.live(m.resolve(aggID))
}

func (m *UserReadModel) FindByEmail(email string) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	aggID, ok := m.emails[email]
	if !ok {
		return nil, false
	}
	return m.live(m.resolve(aggID))
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
	return m.live(m.resolve(aggID))
}

// resolve follows the merge chain from a stream to the live survivor at its
// end; a stream that was not merged away answers with itself. Every pointer
// names the direct survivor of one merge, so a chain is at most as long as the
// pointer map: a walk that used every hop and is still on a merged stream
// found a cycle, where no stream is live, and answers with the zero ID.
// Callers hold at least a read lock.
func (m *UserReadModel) resolve(aggID id.StreamID) id.StreamID {
	for range m.mergeChainLimit() {
		survivor, merged := m.mergedInto[aggID]
		if !merged {
			return aggID
		}
		aggID = survivor
	}
	return id.StreamID{}
}

// live returns a copy of the live user on aggID. Callers hold at least a read
// lock. A missing row answers not-found rather than a nil user, so an index
// entry that outlived its row can never leak a nil clone.
func (m *UserReadModel) live(aggID id.StreamID) (*User, bool) {
	u, ok := m.users[aggID]
	if !ok {
		return nil, false
	}
	return u.Clone(), true
}

// row returns a copy of what is stored on aggID itself. Unlike the finders it
// does not follow a merge pointer: a merged stream answers not-found here
// while FindByID answers with its survivor. Callers need the user as it was
// before a merge retires it.
func (m *UserReadModel) row(aggID id.StreamID) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[aggID]
	if !ok {
		return nil, false
	}
	return u.Clone(), true
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
