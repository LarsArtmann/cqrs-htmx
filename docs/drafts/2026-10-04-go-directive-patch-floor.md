# Proposed title

submodules + the v4.13.0 tag pin `go 1.27.1` — every consumer inherits a patch floor no code needs

> **CLOSED WITHOUT FILING — 2026-11-08 (U1 per-module verification, TODO_LIST timesheets #59).**
> The premise is resolved upstream: **v4.16.0's 24 real module directives all
> carry `go 1.27.2`** (verified against the tag: root, adminui, dashboardui,
> datastar, e2e/server, health, identity-model, integration_test, loginpage,
> setup, auditlog@`go 1.27`, and 12 examples — the `go 1.26.7`/`go 1.27`
> stragglers are `scripts/testdata/` fixtures, not published modules). The
> v4.13.0-era mismatch (floor exceeds code needs, master unreleased) no longer
> exists: the 1.27.2 floor is the deliberate 2026-10-12 family-floor alignment
> (timesheets `f91c896` records the same move), and v4.16.0 shipped it. The
> "file upstream, then tag the normalization release" plan is moot — the
> normalization release (v4.16.0) already exists. The remaining behavior —
> consumers below 1.27.2 are forced to ratchet — is fleet versioning policy,
> not a defect. Evidence: `git grep -h "^go 1" v4.16.0 -- "**/go.mod"`.

## Problem

`go 1.27.1` is declared in 20 module directives across the repo, while the
code itself runs green on Go 1.27.0. Consumers pay for this on every
`go get`: the Go toolchain refuses any directive below a dependency's floor,
so importing `github.com/larsartmann/cqrs-htmx/v4@v4.13.0` (or any submodule)
forces `go >= 1.27.1` onto the consumer's go.mod. Under `GOTOOLCHAIN=local`
or Nix-managed toolchains that don't have 1.27.1, the build fails outright;
everywhere else it silently downloads a toolchain for a patch release that
adds no API.

Concrete breakage, this morning, in `timesheets`:

```
$ go mod edit -go=1.27 && go build ./...
go: github.com/larsartmann/cqrs-htmx/v4@v4.13.0 requires go >= 1.27.1
(running go 1.27.0)
```

## Evidence

- `v4.13.0:go.mod:4` declares `go 1.27.1` (the latest release; this is what
  consumers resolve).
- Master root was already normalized to `go 1.27` (go.mod on `492e473e`),
  but 20 module directives still say `go 1.27.1`: adminui, dashboardui,
  e2e/server, examples/* (12), health, integration_test, loginpage, setup,
  systemadapter, usermgmt — plus the root `go.work`.
- Load-bearing check (worktree at `492e473e`): `dashboardui` with its
  directive forced to `go 1.27`, root `replace` set to local master, built and
  tested green on `GOTOOLCHAIN=go1.27.0`:

```
$ GOWORK=off GOTOOLCHAIN=go1.27.0 go build ./... && go test ./...
BUILD_OK_1_27_0
ok  github.com/larsartmann/cqrs-htmx/dashboardui/v4       0.496s
ok  github.com/larsartmann/cqrs-htmx/dashboardui/v4/core  0.007s
```

Patch releases add no language or stdlib API, so a `1.27.1` floor is never
load-bearing — it only propagates. This one most likely came from
`go mod tidy`/`go work use` running under a local 1.27.1 toolchain.

## Fix

Normalize the remaining 20 directives to `go 1.27` to match the root, and
tag the next release from master (the root fix is already on master but
unreleased — consumers are stuck on v4.13.0's floor until then).

## Scope

Every consumer of any cqrs-htmx module benefits (timesheets and
file-and-image-renamer hit this exact wall during today's cmdguard v4 fleet
migration); cqrs-htmx itself is unaffected — 1.27.0 is within its own
supported range.

---

💘 Generated with Crush
