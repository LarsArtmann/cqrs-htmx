package datastar

import _ "embed"

// datastarVersion is the version of the embedded Datastar JavaScript library.
// This is the JS client version, separate from the Go SDK version (go.mod).
const datastarVersion = "1.0.2"

//go:embed datastar/datastar.js
var datastarJS []byte
