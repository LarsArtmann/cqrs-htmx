package cqrshtmx

import _ "embed"

// syncVersion is the version of the embedded offline sync assets.
const syncVersion = "1.1.0"

//go:embed sync/sync-worker.js
var syncWorkerJS []byte

//go:embed sync/sync-client.js
var syncClientJS []byte
