# Decision: `/mnt/buildcache` (sda1) — replace or retire

**Status:** PREPARED, awaiting user decision (hardware spend).
**Prepared:** 2026-08-30.

## Facts (verified)

- sda1 began failing 2026-08-16 (I/O errors under build load), FULLY dead
  since 2026-08-22: the mount is gone, `df` reports "No such device".
- All Go caches lived there: `GOCACHE`, `GOMODCACHE`, `GOLANGCI_LINT_CACHE`.
- Current workaround: `/tmp` layout
  (`export GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache GOLANGCI_LINT_CACHE=/tmp/golangci-cache`),
  institutionalized in `scripts/lib/go-cache-env.sh` (auto-fallback + free-space
  guard) and sourced by the pre-commit hook since 2026-08-30.
- Costs of the workaround: /tmp is tmpfs (RAM-backed) on this machine — a
  full /tmp surfaces as phantom `no space left` / corrupt-module errors
  (the 2026-08-29 incident); a reboot wipes the caches, so the first build
  after every boot re-downloads the module universe (~minutes) and cold
  compiles the stdlib GOEXPERIMENT=jsonv2 packages.

## Options

| Option                                                                                             | Upside                                                                                  | Downside                                                                               |
| -------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| **A. New SSD for caches** (mount at /mnt/buildcache)                                               | restores persistent warm caches; env vars unchanged; fastest builds                     | hardware cost + install effort; the old disk died young — same failure class may recur |
| **B. Repoint caches to a persistent NVMe dir** (e.g. `~/projects/.gocache` on the healthy root fs) | zero hardware cost; survives reboots; one-line env change + go-cache-env default update | consumes root-SSD space (~10-30 GB warm); slight wear                                  |
| **C. Keep /tmp status quo**                                                                        | nothing to do                                                                           | reboot = cold everything; tmpfs pressure incidents recur                               |

## Recommendation

**B.** The root NVMe is healthy and large; caches are reproducible data, so
RAID/redundancy concerns do not apply; and B needs no purchase or mount
changes — just move the `GOCACHE`/`GOMODCACHE`/`GOLANGCI_LINT_CACHE` exports
(one flake/devShell edit + AGENTS Quick Reference row + go-cache-env fallback
path) and run one warm-up build. Revisit A only if root-SSD space becomes
tight.

## Execution sketch (B)

1. `mkdir -p ~/projects/.gocache/{build,mod,golangci}`.
2. Update `scripts/lib/go-cache-env.sh` fallback paths + AGENTS Quick
   Reference + flake devShell env (all three name the /tmp layout today).
3. `GOEXPERIMENT=jsonv2 go build ./...` once per module group to warm.
4. Keep /tmp as the last-resort fallback in go-cache-env (order: env var →
   persistent dir → /tmp).
