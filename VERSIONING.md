# Versioning Policy

This library follows [Semantic Versioning 2.0.0](https://semver.org/) with the
module suffix convention described in the [Go module wiki](https://go.dev/wiki/Modules#semantic-import-versioning).

## Summary

| Change                     | Version bump                  | Examples                                                                 |
| -------------------------- | ----------------------------- | ------------------------------------------------------------------------ |
| Breaking API change        | **MAJOR** (v3 → v4)           | Renamed/removed exported types, changed signatures, changed import paths |
| New feature (non-breaking) | **MINOR** (v3.x → v3.x+1)     | New exported types, new optional config fields, new HandlerOptions       |
| Bug fix, performance, docs | **PATCH** (v3.x.y → v3.x.y+1) | Internal refactors, error message improvements, test additions           |

## Module Paths

Each major version lives at a distinct import path:

- **v3**: `github.com/larsartmann/cqrs-htmx/v4` (current)
- **v4**: `github.com/larsartmann/cqrs-htmx/v4` (future)

Sub-modules follow the same pattern:

- `github.com/larsartmann/cqrs-htmx/usermgmt/v4`
- `github.com/larsartmann/cqrs-htmx/adminui/v4`

API documentation generation now lives in `go-cqrs-lite/catalog/v3` (the `simple` and `docserver` sub-packages), not in cqrs-htmx.

## What Counts as Breaking

- Removing or renaming an exported type, function, method, constant, or variable
- Adding a required parameter to a function or method
- Changing the type of an exported field or return value
- Changing the JSON wire format of persisted event payloads (event sourcing compatibility)
- Changing the default behavior when a consumer's zero-value config previously worked

## What Is NOT Breaking

- Adding new exported types, functions, or methods
- Adding new optional fields to config structs (zero-value must preserve old behavior)
- Adding new `HandlerOption` factories
- Internal refactors that don't change the public API
- Changing error messages (the error code and family are the stable contract)

## Release Cadence

- **PATCH**: Released as needed for bug fixes.
- **MINOR**: Released when new features are ready (typically every few weeks).
- **MAJOR**: Only when accumulated breaking changes justify a new major version.

## Deprecation Policy

Before removing a public API in a major version:

1. Mark it deprecated with a `// Deprecated:` comment in the prior minor release.
2. Document the replacement in the deprecation comment.
3. Keep it functional until the next major version.

## Compatibility Promise for v3

The v3 line maintains backward compatibility within the minor version series.
Consumers can safely upgrade patch and minor versions without code changes.

Event payloads persisted by v3.x are forward-compatible within v3 — new fields
are additive and old code ignores unknown fields during JSON unmarshaling.
