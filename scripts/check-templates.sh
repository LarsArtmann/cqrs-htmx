#!/usr/bin/env bash
# Verifies that //go:build ignore SQL setup template files in usermgmt/ compile.
#
# These template files (sqlite_setup.go, postgres_setup.go, mysql_setup.go,
# sql_setup_shared.go) are excluded from normal builds via //go:build ignore
# because they import backend-specific stack packages (stack/sqlite, stack/postgres,
# stack/mysql) that are not dependencies of the published usermgmt module.
#
# This script temporarily strips the build tags, adds the stack backend deps,
# compiles the full usermgmt package, then restores all originals.
#
# Runs hermetically (GOWORK=off) so every dependency resolves from published
# go-cqrs-lite tags — identical behavior locally and in CI. This also isolates
# the check from local go.work replaces pointing at in-flight sibling work.
#
# Version notes (checked against published upstream tags):
#   - stack/sqlite/v4 v4.3.0, stack/postgres/v4 v4.3.0: latest published.
#   - stack/postgres v4.2.0 is broken in isolation (references unreleased
#     storage/v4 API — NotificationListener, NewPostgresBus); v4.3.0 fixed it.
#   - stack/mysql/v4 has no v4.2.0+ tag; v4.1.0 is the latest published and
#     compiles against the current templates.
#
# Usage: nix run .#check-templates  OR  bash scripts/check-templates.sh
set -euo pipefail

export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"
export GOWORK=off

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT/usermgmt"

TEMPLATE_FILES=(
  "sqlite_setup.go"
  "postgres_setup.go"
  "mysql_setup.go"
  "sql_setup_shared.go"
)
ALL_FILES=("${TEMPLATE_FILES[@]}" "go.mod" "go.sum")

BACKUP_DIR=$(mktemp -d)
trap 'restore; rm -rf "$BACKUP_DIR"' EXIT

restore() {
  for f in "${ALL_FILES[@]}"; do
    if [ -f "$BACKUP_DIR/$f" ]; then
      cp "$BACKUP_DIR/$f" "$f"
    fi
  done
}

for f in "${ALL_FILES[@]}"; do
  [ -f "$f" ] && cp "$f" "$BACKUP_DIR/$f"
done

# 1. Add stack backend requires to usermgmt/go.mod
go mod edit \
  -require github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4@v4.3.0 \
  -require github.com/larsartmann/go-cqrs-lite/stack/postgres/v4@v4.3.0 \
  -require github.com/larsartmann/go-cqrs-lite/stack/mysql/v4@v4.1.0

# 2. Strip //go:build ignore + following blank line from template files
for f in "${TEMPLATE_FILES[@]}"; do
  sed -i '1,2{/^\/\/go:build ignore$/d; /^$/d}' "$f"
done

# 3. Build the usermgmt package with all template files included
echo "==> Building usermgmt with template files (build tags stripped)..."
if go mod tidy && go build ./...; then
  echo "✓ All SQL setup template files compile successfully"
else
  echo "✗ SQL setup template files have compilation errors" >&2
  exit 1
fi
