#!/usr/bin/env bash
# check-service-methods.sh — Guard the *usermgmt.Service method count
#
# *Service is the leading v5 god-object indicator (ADR-0038 proposed
# decomposition). Once the method count crosses the threshold, decomposition
# can no longer be deferred. This check fails CI before the type grows past
# the agreed ceiling, surfacing the trend early.
#
# The count includes BOTH exported and unexported methods (unexported methods
# are part of the type's complexity too).
#
# Usage: ./scripts/check-service-methods.sh
# Env:   SERVICE_METHOD_LIMIT  hard ceiling (default 80, per ADR-0038 trigger)
# Exit:  0 = under limit, 1 = at/over limit

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LIMIT="${SERVICE_METHOD_LIMIT:-80}"

# Count all methods (exported + unexported) with a *Service receiver in usermgmt.
# Excludes test files and generated _templ.go files.
count=$(
	cd "$REPO_ROOT/usermgmt"
	grep -rhoP 'func \(\w+ \*Service\) \w+' \
		--include='*.go' --exclude='*_test.go' . |
		sort -u |
		wc -l
)

echo "=== *Service method count: ${count} (limit ${LIMIT}, ADR-0038 trigger at ${LIMIT}) ==="

if [ "$count" -ge "$LIMIT" ]; then
	echo "FAIL: *usermgmt.Service has ${count} methods — the v5 decomposition" >&2
	echo "      ceiling (${LIMIT}) has been reached. Decompose before adding more." >&2
	echo "      See docs/adr/0038-service-decomposition-proposed.md." >&2
	exit 1
fi

remaining=$((LIMIT - count))
echo "OK: ${count} methods, ${remaining} until the decomposition trigger."
