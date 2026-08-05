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
# Usage: nix run .#check-templates  OR  bash scripts/check-templates.sh
set -euo pipefail

export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

TEMPLATE_FILES=(
	"usermgmt/sqlite_setup.go"
	"usermgmt/postgres_setup.go"
	"usermgmt/mysql_setup.go"
	"usermgmt/sql_setup_shared.go"
)
CONFIG_FILES=(
	"go.work"
	"usermgmt/go.mod"
)
ALL_FILES=("${CONFIG_FILES[@]}" "${TEMPLATE_FILES[@]}" "usermgmt/go.sum")

BACKUP_DIR=$(mktemp -d)
trap 'restore; rm -rf "$BACKUP_DIR"' EXIT

restore() {
	for f in "${ALL_FILES[@]}"; do
		local key
		key="$(echo "$f" | tr '/' '_')"
		if [ -f "$BACKUP_DIR/$key" ]; then
			cp "$BACKUP_DIR/$key" "$f"
		fi
	done
}

for f in "${ALL_FILES[@]}"; do
	[ -f "$f" ] && cp "$f" "$BACKUP_DIR/$(echo "$f" | tr '/' '_')"
done

# 1. Add stack/mysql replace to go.work (not currently in workspace replaces)
if ! grep -q 'stack/mysql/v4' go.work; then
	printf '\nreplace github.com/larsartmann/go-cqrs-lite/stack/mysql/v4 => /home/lars/projects/go-cqrs-lite/stack/mysql\n' >>go.work
fi

# 2. Add stack backend requires to usermgmt/go.mod
(
	cd usermgmt
	go mod edit \
		-require github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4@v4.2.0 \
		-require github.com/larsartmann/go-cqrs-lite/stack/postgres/v4@v4.2.0 \
		-require github.com/larsartmann/go-cqrs-lite/stack/mysql/v4@v4.2.0
)

# 3. Strip //go:build ignore + following blank line from template files
for f in "${TEMPLATE_FILES[@]}"; do
	sed -i '1,2{/^\/\/go:build ignore$/d; /^$/d}' "$f"
done

# 4. Build the usermgmt package with all template files included
echo "==> Building usermgmt with template files (build tags stripped)..."
if go build ./usermgmt/...; then
	echo "✓ All SQL setup template files compile successfully"
else
	echo "✗ SQL setup template files have compilation errors" >&2
	exit 1
fi
