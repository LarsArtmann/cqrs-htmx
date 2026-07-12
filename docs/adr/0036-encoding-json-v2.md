# ADR 0036: encoding/json/v2 via GOEXPERIMENT=jsonv2

**Date:** 2026-07-12
**Status:** Accepted

## Context

Go 1.26 ships `encoding/json/v2` behind the `GOEXPERIMENT=jsonv2` build flag.
The v2 package offers significant improvements over `encoding/json` (v1):

- **Type-safe `MarshalTo`/`MarshalEncode`** — write directly to an `io.Writer`
  without intermediate allocation
- **Consistent error handling** — `SemanticError` and `SyntaxError` are typed,
  enabling precise `errors.Is` / `errors.As` chains
- **No silent default behavior** — v2 rejects unknown fields by default instead
  of silently dropping them, catching schema drift at decode time
- **Faster** — 2-3x throughput on common workloads due to streaming-first
  architecture

This project uses `encoding/json/v2` across 28+ `.go` files spanning every
module: root, usermgmt, and all auth strategy submodules (totp, webauthn,
oauth2). It is used for event payload marshaling, HTTP request/response bodies,
SQL view serialization, and WebAuthn/OAuth2 ceremony data.

## Decision

**Adopt `encoding/json/v2` as the project-wide JSON library.**

All modules require `GOEXPERIMENT=jsonv2` to build. This is:

1. **Documented** in AGENTS.md (Quick Reference table + Gotcha #9)
2. **Enforced** in `flake.nix` devShell (`GOEXPERIMENT = "jsonv2"`)
3. **Set** in every nix app (`test`, `build`, `lint`, `coverage`, etc.)
4. **Required** in manual commands (`GOEXPERIMENT=jsonv2 go test ./...`)

### What NOT to do

- **Never ban `encoding/json/v2`** — depguard, golangci-lint, or custom checks
  that flag `encoding/json/v2` as banned are false positives. The previous
  `scripts/release-checklist.sh` had this bug (step 3) and was fixed.
- **Never use `encoding/json` (v1)** — the v1 API is avoided for consistency.
  New code should import `encoding/json/v2`.
- **Never remove `GOEXPERIMENT=jsonv2`** from build commands — the code will
  not compile without it.

### Migration

There is no migration path from v1 to v2 for existing code. The project was
built on json/v2 from the start (v4.0.0). Consumers who import cqrs-htmx must
also set `GOEXPERIMENT=jsonv2` in their build environment.

## Consequences

| Aspect                | Impact                                                           |
| --------------------- | ---------------------------------------------------------------- |
| **Build requirement** | `GOEXPERIMENT=jsonv2` must be set in ALL environments            |
| **Consumer burden**   | Importers must also set the flag (documented in Quick Reference) |
| **CI**                | `.github/workflows/ci.yml` sets `GOEXPERIMENT: jsonv2`           |
| **Nix**               | `flake.nix` devShell + all apps set it automatically             |
| **LSP**               | gopls auto-detects from go.mod (no manual config needed)         |
| **Future**            | When Go promotes json/v2 from experiment to stable, the flag can |
|                       | be dropped with zero code changes                                |

## Related

- [Go JSON v2 proposal](https://go.googlesource.com/proposal/+/master/design/71453-json-v2.md)
- AGENTS.md Gotcha #9: "encoding/json/v2 is INTENTIONAL"
- AGENTS.md Quick Reference: all commands include `GOEXPERIMENT=jsonv2`
