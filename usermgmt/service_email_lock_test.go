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

	const (
		addresses = 3
		movers    = 2
		rounds    = 6
	)
	email := func(i int) string { return fmt.Sprintf("lock%d@oauth.test", i) } // perProviderOAuth2Stub's address
	provider := func(i int) string { return fmt.Sprintf("lock%d", i) }

	for _, maxUsers := range []int{0, 10} {
		t.Run(fmt.Sprintf("MaxUsers=%d", maxUsers), func(t *testing.T) {
			t.Parallel()

			svc, err := NewService(ServiceConfig{OAuth2: perProviderOAuth2Stub{}, MaxUsers: maxUsers})
			if err != nil {
				t.Fatalf("NewService: %v", err)
			}
			t.Cleanup(svc.Stop)
			ctx := context.Background()

			// The movers are pre-registered so ChangeEmail has a user to act
			// on whatever the racing registrations do.
			moverIDs := make([]UserID, movers)
			home := func(m int) string { return fmt.Sprintf("mover-%d@test.com", m) }
			for m := range movers {
				moverIDs[m] = registerTestUser(t, svc, fmt.Sprintf("mover-%d", m), home(m)).User.ID
			}

			var importBody strings.Builder
			_, _ = importBody.WriteString("[")
			for i := range addresses {
				if i > 0 {
					_, _ = importBody.WriteString(",")
				}
				_, _ = fmt.Fprintf(&importBody, `{"email":%q}`, email(i))
			}
			_, _ = importBody.WriteString("]")

			workers := make([]func(), 0, 2*addresses+movers+1)
			for i := range addresses {
				workers = append(workers,
					func() {
						for range rounds {
							_, _ = svc.Register(ctx, RegisterRequest{Email: email(i)})
						}
					},
					func() {
						for range rounds {
							_, _ = runTestOAuth2Login(ctx, svc, provider(i))
						}
					},
				)
			}
			for m := range movers {
				workers = append(workers, func() {
					for range rounds {
						_ = svc.ChangeEmail(ctx, moverIDs[m], email(m%addresses))
						_ = svc.ChangeEmail(ctx, moverIDs[m], home(m))
					}
				})
			}
			workers = append(workers, func() {
				for range rounds {
					_, _ = svc.ImportUsersFromJSON(ctx, strings.NewReader(importBody.String()))
				}
			})

			start := make(chan struct{})
			var wg sync.WaitGroup
			for _, run := range workers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start // barrier: every path races from the same gun
					run()
				}()
			}
			close(start)
			wg.Wait()

			for i := range addresses {
				if holders := countUsersWithEmail(svc, email(i)); holders > 1 {
					t.Errorf("rows holding %s = %d, want at most 1", email(i), holders)
				}
			}
		})
	}
}
