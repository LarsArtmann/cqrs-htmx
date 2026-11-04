# STATUS REPORT — Root-package decomposition (`go-modularize`, episode 5)

> **ANNOTATED 2026-11-02** — ~~this report~~ **SUPERSEDED** by [`2026-11-02_16-23_root-split-recovery-reexport-refactor-and-gate-battery_status.md`](2026-11-02_16-23_root-split-recovery-reexport-refactor-and-gate-battery_status.md), which the 16-23 report is the same episode carried to its end state: adds the recovery, the 0-issue lint verdict with the fix table, the var-alias facade conversion, the API pin, the layering gate, the 79/84 battery triage, and the master tag-cut that closes the coverage blocker. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-02 (root-package-decomposition close-out (2026-11-02)).**

**Date:** 2026-11-02 16:20 · **Branch:** `modularize/root-package-split` (worktree:
`/home/lars/projects/cqrs-htmx-root-split`) · **Base:** `master` @ `7daabb8a` plus the
non-overlapping content of `master` @ `4c631647` · **Skill:** `go-modularize`

**Trigger:** the owner, after the measurement pass: *"I think our root project is like WAY
to full with source files!"* The module landscape itself was already settled — the
[2026-10-18 proposal](../../modularization/2026-10-18_PROPOSAL.html) closed with "zero
merges, zero splits, zero reorganizes" and exactly one watch item (W1: the root package,
57 production files, 267 symbols in one directory). This episode works that item at the
PACKAGE level: no new `go.mod`, no `go.work` change, 29 modules before and after.

---

## (a) FULLY DONE

### a.1 The decision that shaped the work

A module split was rejected (the 2026-10-24 memo already chose one repo, and 29 modules
already publish as one release train). What the complaint describes is directory shape, so
the fix is packages inside the existing module, with the root package kept as the
published surface via re-export facades. Consumers see zero change; the repository gains
four independently importable packages and a machine-checked direction.

### a.2 Four packages extracted

| Package         | Owns                                                                                                                                                                                                                                          | Files      |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| `internal/hx`   | Shared plumbing that breaks the import cycle: JSON keys, `WriteAll`/`WriteAllString`, `MarshalJSONForResponse`/`OrFallback`, `HashTag`, `ServeImmutableJSON`, `SerializeToImmutableHandler`, the request-ID context key, `PanicCode`/`IsPanicError` | 7          |
| `htmx/`         | HTMX protocol vocabulary: `Request`, `ParseRequest`, `FromContext`/`WithRequest`, swap strategies, all 18 `Header*` constants, embedded `htmx.min.js` + 2 extension assets, `ScriptHandler`/`ScriptTag`/`CDNScriptTag`, `Extension*` handlers     | 6 + assets |
| `eventcatalog/` | The published event language: `Catalog`, `EventMetadata`, `PayloadField`, `Handler` (immutable JSON + ETag)                                                                                                                                    | 2          |
| `httperr/`      | Error family → HTTP status, `SafeDetail`, `ErrorCode`, the RFC 9457 `StructuredError`, `HandleErrorCore`/`PlainBodyWriter`, every ready-made handler                                                                                            | 3          |

`internal/hx` exists for one reason: a subpackage cannot import the root package once the
root re-exports it, so every primitive needed by two or more clusters lives at the floor.

### a.3 Names cleaned at the seam, published names untouched

Moving `HTMXTarget` into package `htmx` makes `htmx.HTMXTarget` stutter — `revive`'s
`exported` rule rejects it, and that is the repo's own lint bar, so it is not negotiable.
The subpackages therefore drop the redundant prefix (`htmx.Target`, `htmx.ScriptHandler`,
`htmx.Request`, `eventcatalog.Catalog`, `eventcatalog.Handler`), while the root package
keeps the historical spelling. Same value, two spellings, each correct in its own scope.

### a.4 The facades follow the repo's own precedent

`csrf_reexport.go` already established the pattern, including the `.golangci.yml`
exemption for `*_reexport.go` (`gochecknoglobals` + `revive`). The three new facades are
`htmx_reexport.go`, `eventcatalog_reexport.go`, `httperr_reexport.go`, and they use
`var` aliases rather than wrapper functions:

```go
HTMXTarget       = htmx.Target      // identical function value, no wrapper frame
HTMXVersion      = htmx.Version
type HTMXRequest = htmx.Request     // identical type
```

A `func HTMXTarget(r *http.Request) string { return htmx.Target(r) }` wrapper can silently
drop a parameter; a `var` alias cannot. 36 wrappers became 36 aliases.

### a.5 The published API is pinned at compile time

`api_surface_compat_test.go` (`package cqrshtmx_test`) asserts the exact signature of all
40 moved functions, the identity of 3 types, all 13 constants, and all 8 sentinel errors.
It exists because every in-repo consumer compiles against the root package either way — a
forgotten alias would pass this repository's entire test suite and only break an external
consumer on upgrade.

### a.6 Layering is enforced, not documented

`scripts/checks/check-module-layers.sh` (`nix run .#check-module-layers`) reads
`go list -f '{{.ImportPath}}|{{join .Imports " "}} {{join .TestImports " "}}'` and fails on
any edge outside the declared table:

```
internal/hx   → (nothing in-module)
htmx          → internal/hx
eventcatalog  → internal/hx
httperr       → htmx, internal/hx
.             → all of the above
```

Test imports are included, because a back-edge in a `_test.go` is the same defect.
`scripts/selftests/test-check-module-layers.sh` plants one violation per rule (9 cases,
offline fixtures). Both are wired into `check-modules`, CI, and the flake app set.

### a.7 Measured result

| Metric                      | Before | After          | Note                                         |
| --------------------------- | ------ | -------------- | -------------------------------------------- |
| Root production `.go` files | 57     | 51             | +3 are the re-export facades                 |
| Root test files             | 105    | 98             | +1 is the API conformance test               |
| Root-package LOC            | 8,121  | 7,325          | −796                                         |
| Packages in root module     | 5      | 9              | root, authztest, cmd, openapi, transport + 4 |
| Go modules                  | 29     | 29             | unchanged, by design                         |
| Public API                  | —      | —              | identical; compile-pinned                    |

### a.8 Verification actually run (all on the final tree)

```
go build ./...                                        rc=0, all 29 workspace modules (incl. 13 examples + e2e/server)
go test ./... -count=1                                rc=0, 9/9 packages
go mod tidy -diff   (every module)                    rc=0, no module drifts
golangci-lint run   (root module, v2.15.0)            0 issues
bash scripts/checks/check-module-layers.sh            OK, 4 packages
bash scripts/selftests/test-check-module-layers.sh    9/9 cases
bash scripts/selftests/test-check-modules-parity.sh   green (new stages stay subsets)
nix flake check --no-build                            all checks passed
bash scripts/checks/check-root-files.sh --strict      green (baseline re-pinned: 30 files)
```

**One config change came with the extraction:** `wrapcheck.ignore-package-globs` gained
`github.com/larsartmann/cqrs-htmx/v4/internal/*` and `github.com/larsartmann/go-cqrs-lite/id/*`.
Both are the false-positive class the repo already globs for `go-error-family` and
`go-cqrs-lite/event`: those packages build family-tagged errors at the source, so an
unwrapped return loses nothing, and re-wrapping would duplicate a message that already
names its code. The three new findings this silenced were exactly that shape.

### a.9 Archive pass and docs budgets

The tail budget was red at 6 live reports against a ratchet baseline of 4; the documented
ritual (`archive-status.sh --apply --update-baseline`) archived two SUPERSEDED reports —
this episode's earlier draft (`2026-11-02_12-38_…`, kept for its AST cluster analysis and
dependency-edge matrix, which is the evidence behind the four packages) and the
`2026-10-31_09-26` round-45 self-critique, whose series is closed and whose successor is
the 2026-11-02 consumer-usage audit. Tail is now 4/4. `docs/CORPUS_BASELINE` was bumped
38574 → 39685 in the same change (the gate's sanctioned path for deliberate growth: two
reports plus gotchas 47/48); one broken link the new report carried was fixed
(`docs/modularization/…` → `../modularization/…`), and `check-docs-links.sh` is green at
989 links.

### a.10 Docs updated

`AGENTS.md` (root-module packages bullet, Quick-Reference row for the new gate, gotcha
index), `README.md` (subpackage table), `.agents/skills/cqrs-htmx/SKILL.md` (subpackage
section and the two rules that matter), `docs/gotchas.md` (47, 48), `CURRENT_FOCUS.md`
(the round-46 thread kept as active; this work recorded as landed on its own branch).

---

## (b) PARTIALLY DONE

### b.1 `coverage-gate` is red — and it is red on master, not here

`nix run .#coverage-gate` fails on `usermgmt` and `setup` with
`undefined: cqrshtmx.PermissionChecker`. Those two modules are tested **hermetically**
(`goEnv` exports `GOWORK=off`), so they resolve the PUBLISHED root module (`v4.14.1`),
which predates the 2026-11-02 authorization symbols. Proven pre-existing by running the
same hermetic build in a detached worktree at `master` @ `4c631647`: identical failure,
same single symbol. This is the root-tag gap the round-46 focus already names (gotcha 30);
fixing it is a release decision (cut a root `v4` tag), not a refactor step. Root-module
coverage itself is **90.8 %** against a 90 % threshold.

### b.2 `sync` and `openapi` are NOT extracted — deliberately

Both need `App` methods (`App.SyncPushHandler`) or the unexported `handlerConfig`; Go
forbids declaring methods outside the type's own package. Extraction requires either the
root becoming a pure facade over a `core/` package or a redesign of those seams — v5
material behind an ADR, not mechanics. Recorded as the first "next thread" in
`CURRENT_FOCUS.md`.

### b.3 `htmx` package coverage is 77.5 %

The moved tests cover the behaviour, but the package measures lower than root because
several accessors are now exercised only through root-level specs. Not a correctness gap;
a follow-up worth one focused test file. `internal/hx` 87.7 %, `httperr` 89.2 %,
`eventcatalog` 93.8 %.

### b.4 Branch base vs. a moving `master`

`master` advanced twice during the work (`7daabb8a` → `4c631647`, including a history
rewrite: `67c66fa2` and `69f029b2` are twin commits of the same content). This branch was
rebased onto `7daabb8a`, then took the non-overlapping content of `4c631647` (`authztest`
grant proof kit, `docs/PERMISSIONS.md`, the consumer `go.mod` alignment on
identity-model v4.15.0, `check-permission-surface.sh`, the now-untracked CLI binary). Only
`CURRENT_FOCUS.md` overlapped, and it was reconciled by hand. A final rebase onto the
branch tip is the owner's call, because it is a history operation on a pushed branch.

---

## (c) WHAT I GOT WRONG

### c.1 Editing the main tree while a second session was writing there

~10 minutes of work went into `/home/lars/projects/cqrs-htmx` while another agent wrote
`authz_contract.go` (created 11:55:04; my check at 11:55:40). No data was lost, but that
was luck. Everything afterwards ran in a dedicated worktree. → gotcha 47.

### c.2 Losing 59 files to a Git Town stash, and the recovery that saved them

The other session's Git Town run stashed the whole uncommitted refactor and started
`Merge branch 'master' into …`, which conflicted (`AA authztest/authztest.go`,
`UU docs/CORPUS_BASELINE`, `AA transport/deadline_test.go`). The tree looked empty. The
recovery order — patch plus `git update-ref refs/heads/rescue/wip stash@{0}` FIRST, then
`merge --abort`, then `stash pop` — lost nothing. The lesson is not "git is scary": it is
that a rescue ref must be created before any recovery step, because `git stash drop` plus
gc would have been unrecoverable.

### c.3 A split brain I created myself: two request-ID context keys

Root wrote the ID under `contextKey[RequestID]` while `internal/hx` read it under its own
key. Two tests caught it (`recovery_test.go:157`, `app_test.go:81`). Fixed by making root
delegate to `hx.WithRequestID`/`hx.RequestIDFromContext` — one key, one truth. → gotcha 48.

### c.4 `sed` mass-renames destroyed definitions (three times)

`s/\bpanicCode\b/hx.PanicCode/g` also rewrote the DEFINITION into `const hx.PanicCode = …`
(a syntax error); same for `IsPanicError`, `SerializeToImmutableHandler`, `HashTag`,
`ServeJS`. A second pass renamed function PARAMETERS (`includeInternal` → `IncludeInternal`).
Rule now written down: move or delete the definition first, then rename call sites only —
and never mass-rename parameters.

### c.5 A test helper copied with the wrong header name

Moved specs carried `r.Header.Set(htmx.HeaderTrue, htmx.HeaderTrue)` (header name "true")
instead of `htmx.HeaderRequest`. Caught by the moved tests themselves — which is exactly why
tests move WITH the package.

### c.6 The first lint verdict was inconclusive, and I nearly reported it as a defect

`nix run .#lint` timed out on most modules and the root module had no verdict. Re-running
per-module inside `nix develop` (the pinned golangci-lint **2.15.0**, whereas the ambient
binary is 2.14.0) produced a real verdict: 34 findings, all in the new code, all fixed at
the source rather than by nolint. Ambient-binary trap worth remembering: always lint with
the flake's toolchain, or the config's linters may not even exist.

### c.7 The first `coverage-gate` run was blamed on load — it was a tag gap

Two modules failed; both passed standalone, so the first hypothesis was the known
parallel-load flake. A second identical run disproved it. The real cause needs a hermetic
`GOWORK=off` build, which is a different test path than `go test ./...` in the workspace.
Lesson: when a gate fails and the standalone command passes, reproduce the GATE's
environment (here `goEnv`: `GOWORK=off`, `GOTOOLCHAIN=local`), not just its command.

---

## (d) WHAT WE SHOULD IMPROVE

1. **One session per worktree, always.** `git worktree add ../<repo>-<thread> -b <branch>`
   before the first edit. Written into `CURRENT_FOCUS.md` ground rule 6 and gotcha 47.
2. **Cut the root tag before the next gate run.** The coverage gate cannot be green while
   `usermgmt` consumes unreleased root symbols; the fix is the release train, not code.
3. **Move `sync`/`openapi` behind an ADR** if the root package should shrink further — the
   blocker is `App` methods, and no amount of package shuffling solves it.
4. **Add a coverage file for the `htmx` package's root-only paths** (77.5 % → 90 %).
5. **Pin the toolchain in every gate that shells out to `golangci-lint`** (the flake apps
   already do; direct invocations are the trap).

---

## (e) OPEN OWNER QUESTIONS

1. **Rebase or merge** onto the current `master` tip (`4c631647`) before landing? The branch
   is content-complete and verified; moving its base is a history operation on a pushed
   branch.
2. **Should the root eventually become a pure facade over a `core/` package?** That is the
   only shape that unblocks `sync`/`openapi`.
3. **Are the facades permanent?** They are not marked `// Deprecated:` (unlike
   `csrf_reexport.go`), because the subpackages are a new spelling, not a replacement.
4. **Commit granularity when this lands:** four independently revertable commits
   (`internal/hx`, `htmx`, `eventcatalog`, `httperr`) plus one for the gate and one for the
   docs, or a single commit? Nothing is committed yet — repo rule.
