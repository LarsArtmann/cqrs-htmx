# Draft: Go-installable cqrs-lint distribution

**Status:** PREPARED draft, awaiting approval to publish a new artifact.
**Prepared:** 2026-08-30 · Blocks: `check-cqrs-lint` in CI (needs a
CI-installable binary; the Nix flake output is NixOS-only).

> **ANNOTATED 2026-10-01 (docs-health round 14):** the layout premise moved under this draft — the linter source now lives as real Go code in go-cqrs-lite's `cmd/cqrs-lint` (the 2026-09-30 C040 collector fix landed there with regression tests), so "move the analyzer from the nix store build" is obsolete; what remains gated is exactly the publish step (module + `go install` path + tag parity) and the CI flip. Version references (v4.6.0) are stale — the gate now runs a v4.8.x-era binary. The draft's acceptance criteria and CI wiring shape still stand.

## Why

cqrs-lint (currently v4.6.0) is enforced locally via the nix devShell and
`.cqrs-lint.json` per module, but CI cannot install it — so architectural
rules (command-flow, server pins, false-positive presets) gate on my machine
only. A `go install github.com/larsartmann/cqrs-lint/cmd/cqrs-lint@latest`
path makes CI enforcement mechanical.

## Draft layout (new repo `larsartmann/cqrs-lint`, or a `cmd/` inside an

existing tooling repo)

```
cqrs-lint/
  cmd/cqrs-lint/main.go      # flag parsing only; no business logic
  internal/...               # the analyzer, moved as-is from the nix store build
  go.mod                     # go 1.26; deps: go/ast, go/packages, yaml.v3
  .goreleaser.yml            # optional: multi-platform binaries + checksums
```

- Module path `github.com/larsartmann/cqrs-lint` (tool convention: no /vN
  needed for binaries).
- `go install ...@v4.6.0` parity: tag the repo `v4.6.0` so the version the
  nix flake pins and the Go-installable are the same artifact.
- Config discovery unchanged: `.cqrs-lint.json` in the module dir; exit
  codes already CI-shaped (non-zero on findings, `--adoption` mode,
  stale-suppression detector included).

## CI wiring (after publish)

```yaml
- name: Install cqrs-lint
  run: go install github.com/larsartmann/cqrs-lint/cmd/cqrs-lint@v4.6.0
- name: cqrs-lint
  run: cqrs-lint ./...   # per-module loop mirrors scripts/check-cqrs-lint logic
```

## Acceptance

1. `go install` works on a clean machine (no Nix).
2. Output byte-compatible with the nix-built 4.6.0 on this repo (fixture:
   the intentional findings in AGENTS/docs — D005 proximity, V006 lockstep).
3. CI job green; then flip `check-cqrs-lint` from blocked → enforced.
