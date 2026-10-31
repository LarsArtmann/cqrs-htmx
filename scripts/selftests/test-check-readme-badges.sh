#!/usr/bin/env bash
# test-check-readme-badges.sh — offline fixture self-test for
# check-readme-badges.sh (BADGE_FETCHER stub; no network). Covers: happy
# resolution, the lying-verdict classes (no tags / inaccessible), fetch-error
# exit 4 locally vs CI warning downgrade, the no-badges no-op, and .json
# query normalization.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHECK="$SCRIPT_DIR/../checks/check-readme-badges.sh"
TMP=$(mktemp -d /tmp/badge-test-XXXXXX)
trap 'rm -rf "$TMP"' EXIT
failures=0

check() {
  local name="$1" want_rc="$2" got_rc="$3" pattern="$4" out="$5"
  if [ "$want_rc" != "$got_rc" ]; then
    echo "FAIL $name: rc=$got_rc want=$want_rc"
    printf '%s\n' "$out" | head -5
    failures=$((failures + 1))
    return
  fi
  if [ -n "$pattern" ] && ! printf '%s\n' "$out" | grep -q "$pattern"; then
    echo "FAIL $name: output missing '$pattern'"
    printf '%s\n' "$out" | head -5
    failures=$((failures + 1))
    return
  fi
  echo "ok   $name"
}

# Fixture README with two badges (query + plain forms).
README="$TMP/README.md"
cat >"$README" <<'EOF'
[![Release](https://img.shields.io/github/v/tag/LarsArtmann/cqrs-htmx?filter=v4.*&sort=semver&label=release)](x)
[![Go](https://img.shields.io/badge/Go-1.27+-00ADD8)](y)
EOF

# Stub fetcher: records the requested URL, answers by requested path.
CAPTURE="$TMP/captured-url.txt"
cat >"$TMP/fetch-ok" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$1" >>"$CAPTURE"
cat "$TMP/ok.json"
exit 0
STUB
chmod +x "$TMP/fetch-ok"
printf '{"name":"release","value":"v4.14.1"}\n' >"$TMP/ok.json"

out=$(BADGE_FETCHER="$TMP/fetch-ok" bash "$CHECK" "$README" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] &&
  grep -q 'tag/LarsArtmann/cqrs-htmx.json?filter' "$CAPTURE" &&
  grep -q 'Go-1.27+-00ADD8.json' "$CAPTURE" &&
  printf '%s\n' "$out" | grep -q '2 badge(s) resolved'; then
  echo "ok   happy: both URLs normalized to .json and resolved"
else
  echo "FAIL happy: rc=$rc"
  printf '%s\n' "$out" | head -5
  failures=$((failures + 1))
fi

# Lying verdict: value "no tags" fails the gate.
printf '{"name":"release","value":"no tags"}\n' >"$TMP/ok.json"
out=$(BADGE_FETCHER="$TMP/fetch-ok" bash "$CHECK" --gate "$README" 2>&1)
check "no-tags verdict fails" 1 $? 'lying badge' "$out"

printf '{"name":"release","value":"inaccessible"}\n' >"$TMP/ok.json"
out=$(BADGE_FETCHER="$TMP/fetch-ok" bash "$CHECK" "$README" 2>&1)
check "inaccessible verdict fails" 1 $? 'lying badge' "$out"

# Fetch error: exit 4 locally, warning under CI=true.
cat >"$TMP/fetch-dead" <<'STUB'
#!/usr/bin/env bash
exit 1
STUB
chmod +x "$TMP/fetch-dead"
# env -u CI: the local-exit-4 branch must hold even on a runner (the
# selftest-env-leaks gate requires CI-stripping for CI-branching checkers).
out=$(env -u CI BADGE_FETCHER="$TMP/fetch-dead" bash "$CHECK" "$README" 2>&1)
check "fetch error exits 4 locally" 4 $? 'exit 4' "$out"

out=$(CI=true BADGE_FETCHER="$TMP/fetch-dead" bash "$CHECK" "$README" 2>&1)
check "fetch error downgrades to CI warning" 0 $? 'WARN (ci)' "$out"

# No badges: clean no-op.
printf '# just a readme\n' >"$README"
out=$(BADGE_FETCHER="$TMP/fetch-ok" bash "$CHECK" "$README" 2>&1)
check "no-badges readme is a clean no-op" 0 $? 'nothing to resolve' "$out"

if [ "$failures" -gt 0 ]; then
  echo "badge-gate self-test: $failures FAILURE(S)"
  exit 1
fi
echo "badge-gate self-test: all cases green"
