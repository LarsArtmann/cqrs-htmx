package core

import "flag"

// Module-wide golden-update invocations (the README's `go test ./... -update`)
// hand the flag to every package's test binary. This package has no golden
// tests, so the flag is registered as a no-op; drop this shim if golden tests
// ever land here.
var _ = flag.Bool("update", false, "no-op: golden tests live in the module root package")
