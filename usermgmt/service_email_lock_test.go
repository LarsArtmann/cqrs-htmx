package usermgmt

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestEmailStripeLocks_SameAddressSerializes pins the keyed-lock contract two
// callers targeting one address rely on: the second taker waits until the
// first releases the stripe.
func TestEmailStripeLocks_SameAddressSerializes(t *testing.T) {
	t.Parallel()

	var locks emailStripeLocks
	unlock := locks.lock("someone@example.com")

	acquired := make(chan struct{})
	go func() {
		second := locks.lock("someone@example.com")
		close(acquired)
		second()
	}()

	select {
	case <-acquired:
		t.Fatal("the same address was locked twice at once")
	case <-time.After(50 * time.Millisecond):
		// Still blocked behind the first holder, as it must be.
	}

	unlock()
	select {
	case <-acquired:
	case <-time.After(10 * time.Second):
		t.Fatal("the waiting taker was not released after unlock")
	}
}

// TestEmailStripeLocks_DifferentAddressesDoNotBlock pins the other half of the
// contract: an address that hashes to a free stripe is never held up by an
// unrelated one.
func TestEmailStripeLocks_DifferentAddressesDoNotBlock(t *testing.T) {
	t.Parallel()

	var locks emailStripeLocks
	const held = "someone@example.com"
	unlock := locks.lock(held)
	defer unlock()

	other := "other@example.com"
	for locks.stripe(other) == locks.stripe(held) {
		other += "x"
	}

	acquired := make(chan struct{})
	go func() {
		locks.lock(other)()
		close(acquired)
	}()
	select {
	case <-acquired:
	case <-time.After(10 * time.Second):
		t.Fatal("an address on a free stripe blocked behind an unrelated one")
	}
}

// TestEmailStripeLocks_SpreadsAddresses guards against a degenerate stripe
// choice that would serialize unrelated registrations on a few mutexes.
func TestEmailStripeLocks_SpreadsAddresses(t *testing.T) {
	t.Parallel()

	var locks emailStripeLocks
	distinct := map[*sync.Mutex]struct{}{}
	for i := range emailLockStripeCount {
		distinct[locks.stripe(fmt.Sprintf("user%d@example.com", i))] = struct{}{}
	}

	// FNV-1a is deterministic, so the count is fixed (about 40 of 64). The
	// floor only catches a hash that piles everything onto a few stripes.
	if len(distinct) < emailLockStripeCount/2 {
		t.Errorf("%d addresses spread over %d stripes, want at least %d",
			emailLockStripeCount, len(distinct), emailLockStripeCount/2)
	}
}

// TestService_EmailLock_OrderAcrossPaths drives every path that takes the
// registration lock and/or an email stripe at once — Register, ChangeEmail,
// importUsers and the OAuth2 first-login — over one small set of addresses,
// with the MaxUsers gate off (only stripes) and on (registrationMu, then a
// stripe). A wrong lock order (a stripe held while waiting for registrationMu,
// or two stripes nested) would deadlock the service, so the run completing is
// the first assertion; the second is the invariant the locks exist for: no
// address ends up on two users.
func TestService_EmailLock_OrderAcrossPaths(t *testing.T) {
	t.Parallel()

	for _, maxUsers := range []int{0, 10} {
		t.Run(fmt.Sprintf("MaxUsers=%d", maxUsers), func(t *testing.T) {
			t.Parallel()

			arena := newEmailLockArena(t, maxUsers)
			raceWorkers(arena.workers())
			arena.assertNoAddressShared(t)
		})
	}
}

// The arena's size: as many addresses as the OAuth2 stub serves, two
// pre-registered users that move between them, and the rounds each racing
// goroutine runs.
const (
	emailLockAddresses = 3
	emailLockMovers    = 2
	emailLockRounds    = 6
)

// emailLockArena is one service under the lock-order test: the addresses the
// racing paths fight over, the OAuth2 providers that mint them, and the
// pre-registered users ("movers") that change their email back and forth.
type emailLockArena struct {
	svc        *Service
	ctx        context.Context
	emails     []string
	providers  []string
	moverIDs   []UserID
	moverHomes []string
	importBody string
}

// newEmailLockArena builds the service with the OAuth2 stub, pre-registers the
// movers and prepares the JSON import that carries every racing address.
func newEmailLockArena(t *testing.T, maxUsers int) *emailLockArena {
	t.Helper()

	svc, err := NewService(ServiceConfig{OAuth2: perProviderOAuth2Stub{}, MaxUsers: maxUsers})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	t.Cleanup(svc.Stop)

	arena := &emailLockArena{
		svc:        svc,
		ctx:        context.Background(),
		moverIDs:   make([]UserID, emailLockMovers),
		moverHomes: make([]string, emailLockMovers),
	}

	for i := range emailLockAddresses {
		arena.emails = append(arena.emails, fmt.Sprintf("lock%d@oauth.test", i)) // perProviderOAuth2Stub's address
		arena.providers = append(arena.providers, fmt.Sprintf("lock%d", i))
	}

	for m := range emailLockMovers {
		arena.moverHomes[m] = fmt.Sprintf("mover-%d@test.com", m)
		arena.moverIDs[m] = registerTestUser(t, svc, fmt.Sprintf("mover-%d", m), arena.moverHomes[m]).User.ID
	}

	arena.importBody = emailLockImportBody(arena.emails)

	return arena
}

// emailLockImportBody is the JSON import document carrying every address.
func emailLockImportBody(emails []string) string {
	var body strings.Builder

	_, _ = body.WriteString("[")

	for i, email := range emails {
		if i > 0 {
			_, _ = body.WriteString(",")
		}

		_, _ = fmt.Fprintf(&body, `{"email":%q}`, email)
	}

	_, _ = body.WriteString("]")

	return body.String()
}

// workers lists the goroutine bodies that race: per address a Register and an
// OAuth2 first-login, per mover a ChangeEmail pair, and one import of every
// address.
func (a *emailLockArena) workers() []func() {
	workers := make([]func(), 0, 2*len(a.emails)+len(a.moverIDs)+1)

	for i := range a.emails {
		workers = append(workers, a.registerWorker(i), a.oauthWorker(i))
	}

	for m := range a.moverIDs {
		workers = append(workers, a.moverWorker(m))
	}

	return append(workers, a.importWorker())
}

func (a *emailLockArena) registerWorker(i int) func() {
	return func() {
		for range emailLockRounds {
			_, _ = a.svc.Register(a.ctx, RegisterRequest{Email: a.emails[i]})
		}
	}
}

func (a *emailLockArena) oauthWorker(i int) func() {
	return func() {
		for range emailLockRounds {
			_, _ = runTestOAuth2Login(a.ctx, a.svc, a.providers[i])
		}
	}
}

// moverWorker moves one user onto a racing address and back home, so
// ChangeEmail contends with the registrations for the same stripes.
func (a *emailLockArena) moverWorker(m int) func() {
	return func() {
		for range emailLockRounds {
			_ = a.svc.ChangeEmail(a.ctx, a.moverIDs[m], a.emails[m%len(a.emails)])
			_ = a.svc.ChangeEmail(a.ctx, a.moverIDs[m], a.moverHomes[m])
		}
	}
}

func (a *emailLockArena) importWorker() func() {
	return func() {
		for range emailLockRounds {
			_, _ = a.svc.ImportUsersFromJSON(a.ctx, strings.NewReader(a.importBody))
		}
	}
}

// raceWorkers runs every worker on its own goroutine, released from one
// barrier so they all contend at once.
func raceWorkers(workers []func()) {
	start := make(chan struct{})

	var wg sync.WaitGroup

	for _, run := range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start
			run()
		}()
	}

	close(start)
	wg.Wait()
}

// assertNoAddressShared fails when any racing address ended up on two users.
func (a *emailLockArena) assertNoAddressShared(t *testing.T) {
	t.Helper()

	for _, email := range a.emails {
		if holders := countUsersWithEmail(a.svc, email); holders > 1 {
			t.Errorf("rows holding %s = %d, want at most 1", email, holders)
		}
	}
}
