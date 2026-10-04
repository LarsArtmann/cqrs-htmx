#!/usr/bin/env bash
# test-check-release-train.sh — fixture self-test for the release-train gate's
# lag verdicts and the FIX RECIPE output (added 2026-09-22 after four
# same-evening upstream publish waves made the recipe the gate's main UX).
#
# Builds a throwaway tree (script copy + fixture go.mod + seeded tag cache —
# offline; ls-remote is short-circuited by a fresh cache hit):
#   F1  root-module lag          -> exit 3, recipe names exact-anchor sweep
#   F2  submodule lag            -> exit 3, recipe anchors the submodule path
#   F3  aligned at latest        -> exit 0, no recipe lines
#
# Usage: ./scripts/test-check-release-train.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"

pass=0
fail=0

report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

# build_tree <require-line> — returns tree dir in $tree
build_tree() {
  tree=$(mktemp -d)
  mkdir -p "$tree/scripts/lib" "$tree/fixture" "$tree/cache"
  cp "$SCRIPT_DIR/check-release-train.sh" "$tree/scripts/"
  cp "$SCRIPT_DIR/lib/replace-exemption.sh" "$tree/scripts/lib/"
  cat >"$tree/fixture/go.mod" <<'GOMOD'
module github.com/larsartmann/cqrs-htmx/fixture/v4

go 1.27.1

require (
GOMOD
  printf '\t%s\n)\n' "$1" >>"$tree/fixture/go.mod"
}

seed_tags() { # <repo> <tag lines...>
  local repo="$1"
  shift
  printf '%s\n' "$@" >"$tree/cache/$repo.tags"
}

run_gate() { # extra args...
  (
    cd "$tree" &&
      TRAIN_TAG_CACHE_DIR="$tree/cache" bash scripts/check-release-train.sh "$@"
  ) >"$tree/out.txt" 2>&1
  gate_rc=$?
}

echo "== test-check-release-train.sh"

# F1 root-module lag -> exact-anchor recipe
build_tree 'github.com/larsartmann/go-fixture-fake/v9 v0.1.0'
seed_tags go-fixture-fake v0.1.0 v0.2.0
run_gate --strict-lag 0
ok=0
[ "$gate_rc" -eq 3 ] || ok=1
grep -q "TRAIN LAG: github.com/larsartmann/go-fixture-fake/v9@v0.1.0 but v0.2.0" "$tree/out.txt" || ok=1
grep -qF "scripts/bump-dep.sh 'larsartmann/go-fixture-fake/v9\$' v0.2.0" "$tree/out.txt" || ok=1
report $ok "F1 root lag exits 3 with exact-anchor recipe (got $gate_rc)"
rm -rf "$tree"

# F2 submodule lag -> submodule-anchored recipe
build_tree 'github.com/larsartmann/go-fixture-fake/sub/v3 v3.1.0'
seed_tags go-fixture-fake v0.1.0 sub/v3.0.0 sub/v3.1.1
run_gate --strict-lag 0
ok=0
[ "$gate_rc" -eq 3 ] || ok=1
grep -qF "scripts/bump-dep.sh 'larsartmann/go-fixture-fake/sub/v3\$' v3.1.1" "$tree/out.txt" || ok=1
report $ok "F2 submodule lag anchors submodule path (got $gate_rc)"
rm -rf "$tree"

# F3 aligned at latest -> exit 0, no recipe
build_tree 'github.com/larsartmann/go-fixture-fake/v9 v0.2.0'
seed_tags go-fixture-fake v0.1.0 v0.2.0
run_gate --strict-lag 0
ok=0
[ "$gate_rc" -eq 0 ] || ok=1
grep -q "FIX RECIPE" "$tree/out.txt" && ok=1
report $ok "F3 aligned exits 0 with no recipe (got $gate_rc)"
rm -rf "$tree"

echo "== $pass passed, $fail failed"
[ "$fail" -eq 0 ]
