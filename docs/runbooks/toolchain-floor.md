# Runbook: Bumping the Go Toolchain Floor

The fleet-wide Go floor (currently **1.27.2**, raised from 1.27.1 on
2026-10-11 for stdlib net/http vulns GO-2026-6603/6611/6613/6617) lives in
more places than `go.work`. Every bump is a train: miss a site and some leg
CI goes red with a signature that looks like a code bug, not a version skew.

## The touch-list (in order)

1. **`goPkg` override in `flake.nix`** — nixpkgs' `go_1_27` often lags the
   patch you need (it carried 1.27.1 while 1.27.2 was required). Pin the
   patch forward via `overrideAttrs` (src tarball + hash) instead of a whole
   nixpkgs bump. Every flake app that needs Go must take `goPkg`, never the
   ambient `pkgs.go`.
2. **`go.work` `go` directive** — the workspace floor. `scripts/checks/
   check-go-toolchain.sh` fails when this directive is NEWER than the
   devShell toolchain (the 1.26.5-era trap: CI and devShells run
   `GOTOOLCHAIN=local`, so a too-new directive breaks every local build).
3. **Per-module `go.mod` directives** — the go.work floor is the ceiling for
   module directives; align ONLY the modules the bump actually requires
   (16 modules sat at major.minor `go 1.27` long before the 1.27.2 patch
   floor — aligning published-module directives is patch-train material,
   consumer-facing; see gotcha 35's adjudication note).
4. **`scripts/lib/go-cache-env.sh`** — bare shells get
   `GOTOOLCHAIN=go<floor>` automatically (floor alignment: never
   downgrades). If the floor moves, verify this still derives from go.work,
   not a stale literal.
5. **CI `go-version-file:` steps** — the three setup-go steps read root
   `go.mod`/`go.work`; they follow the directives, but a CI-only tool (e.g.
   golangci-lint) may need its pinned `version:` bumped when a new Go's
   export data format lands (the "export data version 5" class — keep the
   CI action pin == the flake's nixpkgs golangci-lint).
6. **`vendorHash.nix` / `hash.nix`** — a floor bump changes the dependency
   graph hashes; `nix run .#build` (or BuildFlow's nix-hash-fix step)
   re-derives them. Never hand-edit.
7. **Tools that read nixpkgs' DEFAULT Go** (gotcha 22): cqrs-lint needs the
   pinned toolchain (`nix develop` or `GOTOOLCHAIN=go1.27.2`); the
   license-check wrapper needs `goPkg`'s GOROOT exported. After a bump,
   smoke-test BOTH via their flake apps.

## Verification battery (same push as the bump)

```bash
nix run .#build            # hermetic per-module builds on the new toolchain
nix run .#test             # full battery
nix run .#lint             # golangci-lint parses the new export data
nix run .#check-modules    # toolchain gate + the rest
bash scripts/tools/tidy-sweep.sh   # module-local tidy under the new floor
```

## History

- 2026-10-11: 1.27.1 → 1.27.2 (net/http HPACK race vulns; nixpkgs lagged,
  overrideAttrs pin landed). Root-cause narrative: `docs/agents-notes.md`.
