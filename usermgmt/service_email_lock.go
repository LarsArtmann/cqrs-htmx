package usermgmt

import (
	"hash/fnv"
	"sync"
)

// emailLockStripeCount is the number of stripes in the per-Service email
// lock: small enough to stay cache-friendly, large enough that unrelated
// addresses rarely share a stripe.
const emailLockStripeCount = 64

// emailStripeLocks is a striped keyed mutex: one mutex per hash bucket of
// normalised email addresses.
//
// It closes the check-then-dispatch race between two callers that target the
// SAME address. Service.Register, Service.ChangeEmail, Service.importUsers and
// the OAuth2 first-login path (OAuth2Service.matchOrCreateUser) each hold
// their stripe across BOTH the read-model lookup and the dispatch, so the
// second caller finds the first caller's user instead of a free address.
// Addresses on different stripes never wait on each other.
//
// Scope — a SINGLE-INSTANCE guard only. The lock set lives in process memory:
// several app replicas each hold their own, so across replicas the uniqueness
// check still relies on the read model being eventually consistent. A
// database-level unique index would be the real constraint, but that is a
// separate, owner-gated decision — the read model deliberately tolerates
// duplicate identities (historical duplicates exist in live databases; see
// the keep-first rule in UserReadModel.handleUserRegistered and
// TestUserReadModel_DuplicateIdentityKeepsOldest).
//
// Lock order — registrationMu (the MaxUsers gate, taken only when MaxUsers is
// greater than zero) BEFORE an email stripe, never the reverse, and at most
// one stripe at a time. Every path that takes both — Register, importUsers and
// the OAuth2 first-login path — acquires them in that order, and no path takes
// registrationMu while holding a stripe, so the wait-for graph is acyclic and
// none of them can deadlock against the others.
//
// Like registrationMu, a stripe is held across the dispatch. Nothing running
// under it may re-enter Register, ChangeEmail, importUsers or the OAuth2
// first-login path with an address on the same stripe.
//
// The zero value is ready to use.
type emailStripeLocks struct {
	stripes [emailLockStripeCount]sync.Mutex
}

// lock acquires the stripe for email and returns the matching unlock func
// (defer it). email must be the NORMALISED address (ParseEmail: trimmed and
// lower-cased) so every spelling of one address maps to one stripe.
func (l *emailStripeLocks) lock(email string) func() {
	mu := l.stripe(email)
	mu.Lock()

	return mu.Unlock
}

// stripe returns the mutex guarding email. FNV-1a is enough: it only has to
// spread addresses evenly over the stripes, not resist adversarial input.
func (l *emailStripeLocks) stripe(email string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(email))

	return &l.stripes[h.Sum32()%emailLockStripeCount]
}
