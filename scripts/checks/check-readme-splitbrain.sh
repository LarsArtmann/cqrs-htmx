#!/usr/bin/env bash
# check-readme-splitbrain.sh — module-README split-brain scan (w15.3).
#
# A module README that names an API the module no longer declares is a
# split brain: consumers read the README, the compiler disagrees. This
# reuses the check-skill-claims core (same drift class) over every module
# README; symbols qualified with the OWNING module's alias must resolve
# there. Root-README claims against sub-module aliases resolve against the
# sub-module, so cross-references stay honest too.
#
# Usage: bash scripts/checks/check-readme-splitbrain.sh
# Exit: 0 (delegates), see check-skill-claims.sh.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

readmes=(
  "$REPO_ROOT/README.md"
  "$REPO_ROOT/usermgmt/README.md"
  "$REPO_ROOT/adminui/README.md"
  "$REPO_ROOT/dashboardui/README.md"
  "$REPO_ROOT/loginpage/README.md"
  "$REPO_ROOT/setup/README.md"
  "$REPO_ROOT/datastar/README.md"
  "$REPO_ROOT/health/README.md"
  "$REPO_ROOT/auditlog/README.md"
  "$REPO_ROOT/systemadapter/README.md"
)

missing=0
docs=()
for readme in "${readmes[@]}"; do
  if [ -f "$readme" ]; then
    docs+=("$readme")
  else
    echo "✗ readme-splitbrain: expected README missing: $readme" >&2
    missing=1
  fi
done
[ "$missing" -eq 0 ] || exit 1

exec bash "$SCRIPT_DIR/check-skill-claims.sh" "${docs[@]}"
