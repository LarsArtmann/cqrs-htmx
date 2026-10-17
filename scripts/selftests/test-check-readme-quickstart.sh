#!/usr/bin/env bash
# test-check-readme-quickstart.sh — offline fixture self-test for
# check-readme-quickstart.sh.
#
# Builds a throwaway fake "cqrs-htmx root" module (stdlib only) and four
# fixture READMEs:
#   clean          — a Quick start fence that uses the fake module's one
#                    exported symbol                    → gate must PASS
#   broken-go      — syntactically invalid Go           → gate must FAIL
#   missing-fence  — Quick start section, no go fence   → gate must FAIL
#                    (candidate-count guard)
#   poisoned       — imports the module but calls a symbol that does not
#                    exist                              → gate must FAIL
#
# Fully offline: GOPROXY=off, GOWORK=off, GOTOOLCHAIN=local; the fake root
# has zero requires, so the fixture graph never touches the network. This is
# the coverage CI gets for the gate (the real-README run needs the proxy).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-readme-quickstart.sh"
TMP=$(mktemp -d /tmp/test-readme-quickstart-XXXXXX)
trap 'rm -rf "$TMP"' EXIT
failures=0

export GOWORK=off
export GOEXPERIMENT=jsonv2
export GOTOOLCHAIN=local
export GOPROXY=off

# --- fixture: fake root module (the replace target) ---
mkdir -p "$TMP/fakeroot"
cat >"$TMP/fakeroot/go.mod" <<'EOF'
module github.com/larsartmann/cqrs-htmx/v4

go 1.21
EOF
cat >"$TMP/fakeroot/realthing.go" <<'EOF'
package cqrshtmx

func RealSymbol() string { return "ok" }
EOF

mkreadme() { # mkreadme <dir> <body...>
  mkdir -p "$1"
  { printf '# Fixture\n\n## Quick start\n\n'; cat; printf '\n## After\n'; } >"$1/README.md"
}

run_gate() { # run_gate <readme> -> sets out/rc
  out="$(bash "$GATE" "$1" 2>&1)"
  rc=$?
}

check() { # check <name> <expected-rc> <pattern>
  local name="$1" want="$2" pattern="$3"
  if [ "$rc" != "$want" ]; then
    echo "FAIL $name: rc=$rc want=$want"
    printf '%s\n' "$out" | head -6
    failures=$((failures + 1))
    return
  fi
  if [ -n "$pattern" ] && ! printf '%s\n' "$out" | grep -q "$pattern"; then
    echo "FAIL $name: output missing '$pattern'"
    printf '%s\n' "$out" | head -6
    failures=$((failures + 1))
    return
  fi
  echo "ok   $name"
}

export README_QUICKSTART_ROOT="$TMP/fakeroot"
export README_FENCE_TIMEOUT=60

# --- case 1: clean fence passes ---
mkreadme "$TMP/clean" <<'EOF'
Copy this:

```go
package main

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func main() {
	_ = cqrshtmx.RealSymbol()
}
```
EOF
run_gate "$TMP/clean/README.md"
check "clean passes" 0 "compiles"

# --- case 2: broken Go fails ---
mkreadme "$TMP/broken" <<'EOF'
```go
package main

func main() { this is not go }
```
EOF
run_gate "$TMP/broken/README.md"
check "broken-go fails" 1 "FAILED to compile"

# --- case 3: missing fence fails (candidate-count guard) ---
mkreadme "$TMP/nofence" <<'EOF'
Just words, no fence.
EOF
run_gate "$TMP/nofence/README.md"
check "missing-fence fails" 1 "candidate count: 0"

# --- case 4: poisoned code (nonexistent symbol) fails ---
mkreadme "$TMP/poisoned" <<'EOF'
```go
package main

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func main() {
	_ = cqrshtmx.DoesNotExist()
}
```
EOF
run_gate "$TMP/poisoned/README.md"
check "poisoned fails" 1 "FAILED to compile"

# --- summary (the gotcha-2 loud candidate count) ---
if [ "$failures" -ne 0 ]; then
  echo "FAIL: $failures case(s) failed (of 4)"
  exit 1
fi
echo "PASS: 4/4 cases (candidate count: 4)"
