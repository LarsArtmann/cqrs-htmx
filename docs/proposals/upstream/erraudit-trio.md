# Upstream draft: erraudit — three DX findings

Target: the erraudit CLI (local tool; distribution TBD — file when a public
repo exists). Posture context: cqrs-htmx gotcha 44 documents the canonical
strict audit as `--max --disable generic_return`; these three findings would
remove the need for the disable flag and the per-invocation repetition.

## 1. Config-file support for flags

Every invocation repeats `--max --disable generic_return --format csv`
(gotcha 44 makes that exact shape the recorded posture). An `.erraudit.yml`
(or flags file) would let a repo declare its posture once. Ask: read a
config file for the recurring flags; `--no-config` as the escape hatch.

## 2. `--modules` / workspace hint

Multi-module Go workspaces need one invocation PER module (the in-repo
`erraudit-inventory.sh` exists only because the tool walks a single module
root). A `--modules` flag (or native go.work decomposition with the
zero-candidates guard the inventory script implements) would remove the
shell wrapper. Ask: workspace-aware sweep, printing `candidates=N` and
failing loudly on zero modules.

## 3. `generic_return` fires on callback-bound identity functions

The documented false-positive class (cqrs-htmx gotcha 44): a decoder option
whose callback RETURNS its input unchanged (`func(c C) (C, error) { return
c, nil }`) is flagged as a generic return, although the identity IS the
contract (the pipeline supports both identity and enriching callbacks).
Suppression today is per-site `//nolint:erraudit` with an invariant-backed
test annotation. Ask: exempt single-identity-return closures, or a
structured suppression (marker + reason) that survives formatting.
