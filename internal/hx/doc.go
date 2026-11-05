// Package hx holds the HTTP plumbing shared by the root cqrshtmx package and
// its subpackages: wire-format constants, response writers, immutable-asset
// serving, and the request-ID context key.
//
// It is internal to this module on purpose. The root package re-exports the
// pieces that are part of its public API (content types, JSON keys, request-ID
// accessors); everything else stays private so the split into subpackages does
// not widen the surface consumers see.
//
// Why it exists: a subpackage cannot import the root package (the root package
// imports its subpackages to re-export them, so that edge would be a cycle).
// The helpers both sides need therefore live here.
package hx
