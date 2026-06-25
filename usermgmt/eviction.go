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
