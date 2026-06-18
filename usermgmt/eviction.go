package usermgmt

import "time"

// startPeriodicEviction runs evict in a background goroutine on the given
// interval. It returns a stop function that, when called, terminates the
// goroutine and releases the ticker. Each store that needs periodic cleanup
// (verification tokens, pending TOTP setups, WebAuthn challenge sessions)
// delegates here to share a single, well-tested implementation. The return
// value of evict is ignored.
func startPeriodicEviction(evict func() int, interval time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
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
	return func() { close(done) }
}
