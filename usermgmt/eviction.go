package usermgmt

import (
	"sync"
	"time"
)

// startPeriodicEviction runs evict in a background goroutine on the given
// interval. It returns a stop function that, when called, terminates the
// goroutine and releases the ticker. Each store that needs periodic cleanup
// (verification tokens, pending TOTP setups, WebAuthn challenge sessions)
// delegates here to share a single, well-tested implementation. The return
// value of evict is ignored. The returned stop function is safe to call
// multiple times — only the first call terminates the goroutine.
func startPeriodicEviction(evict func() int, interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	var once sync.Once
	go func() {
		for {
			select {
			case <-ticker.C:
				_ = evict()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}

// evictExpired locks l, removes every entry from m for which isExpired
// returns true, and returns the number of entries removed. Centralises the
// single-map eviction loop shared by every in-memory expiry store
// (verification tokens, pending TOTP secrets, OAuth2 state, WebAuthn
// sessions, in-memory session store). l is typically *sync.Mutex or
// *sync.RWMutex — both satisfy sync.Locker.
func evictExpired[K comparable, V any](
	l sync.Locker, m map[K]V, isExpired func(K, V) bool,
) int {
	l.Lock()
	defer l.Unlock()
	return deleteExpired(m, isExpired)
}

// deleteExpired removes every entry from m for which isExpired returns true
// and returns the number of entries removed. Caller must hold the lock on m.
// Used directly when a single lock spans multiple maps (e.g. AccountLockout
// has attempts + lockedAt that must stay in sync).
func deleteExpired[K comparable, V any](
	m map[K]V, isExpired func(K, V) bool,
) int {
	n := 0
	for k, v := range m {
		if isExpired(k, v) {
			delete(m, k)
			n++
		}
	}
	return n
}

// deleteKeyed locks l and removes the entry at key from m. Centralises the
// trivial lock+delete idiom shared by every single-key Delete method on the
// in-memory stores.
func deleteKeyed[K comparable, V any](l sync.Locker, m map[K]V, key K) {
	l.Lock()
	defer l.Unlock()
	delete(m, key)
}
