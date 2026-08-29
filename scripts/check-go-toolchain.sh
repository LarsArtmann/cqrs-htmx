#!/usr/bin/env bash
# check-go-toolchain.sh — fail when go.work's `go` directive is NEWER than
# the Go toolchain the flake devShell provides (nixpkgs go).
#
# Catches: bumping go directives past what nixpkgs ships (the 1.26.5-era
# trap: CI and devShells run GOTOOLCHAIN=local, so a too-new directive breaks
# every local build the moment go.work is ahead of nixpkgs).
#
# Usage: ./scripts/check-go-toolchain.sh
# Exit:  0 = go.work directive <= toolchain version, 1 otherwise.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

work_go="$(awk '/^go /{print $2; exit}' go.work)"
toolchain_go="$(go env GOVERSION)"
toolchain_go="${toolchain_go#go}"

if [[ -z $work_go ]]; then
  echo "check-go-toolchain: no 'go' directive found in go.work"
  exit 1
fi

if [[ "$(printf '%s\n%s\n' "$work_go" "$toolchain_go" | sort -V | tail -1)" == "$work_go" && $work_go != "$toolchain_go" ]]; then
  echo "✗ go.work requires go >= $work_go but the flake toolchain is go $toolchain_go."
  echo "  Fix: pin nixpkgs to a Go >= $work_go, or lower the go directives."
  exit 1
fi

echo "✓ go.work go directive ($work_go) <= toolchain ($toolchain_go)"
