#!/usr/bin/env bash
# test-check-pr-secrets.sh — offline fixture self-test for check-pr-secrets.py
# (round-45 M16). Covers: the gated shape passes, the unconditional-secrets
# pull_request job fails, push-only and schedule-only workflows are ignored,
# and the architecture-gate-preflight probe's OUTPUT semantics (secret
# present → has_key=true; absent → has_key=false) are executed for real with
# GITHUB_OUTPUT pointed at a temp file.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHECK="$SCRIPT_DIR/../checks/check-pr-secrets.py"
TMP=$(mktemp -d /tmp/pr-secrets-test-XXXXXX)
trap 'rm -rf "$TMP"' EXIT
failures=0

check() {
  local name="$1" want_rc="$2" got_rc="$3" pattern="$4" out="$5"
  if [ "$want_rc" != "$got_rc" ]; then
    echo "FAIL $name: rc=$got_rc want=$want_rc"
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

run_in() { # run_in <fixture-dir> — run the checker with CWD=fixture
  (cd "$1" && python3 "$CHECK" 2>&1)
}

# GOOD: probe + skip-gated consumer (the ci.yml shape).
mkdir -p "$TMP/good/.github/workflows"
cat >"$TMP/good/.github/workflows/ci.yml" <<'EOF'
name: ci
on:
  pull_request:
jobs:
  probe:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    outputs:
      has_key: ${{ steps.probe.outputs.has_key }}
    steps:
      - id: probe
        env:
          KEY: ${{ secrets.MY_KEY }}
        run: echo "has_key=$([ -n "$KEY" ] && echo true || echo false)" >> "$GITHUB_OUTPUT"
  gated:
    if: github.event_name == 'pull_request' && needs.probe.outputs.has_key == 'true'
    needs: probe
    runs-on: ubuntu-latest
    steps:
      - uses: webfactory/ssh-agent@v0.10.0
        with:
          ssh-private-key: ${{ secrets.MY_KEY }}
EOF
out=$(run_in "$TMP/good")
check "gated shape passes" 0 $? 'skip-gated' "$out"

# BAD: unconditional secrets consumer on pull_request.
mkdir -p "$TMP/bad/.github/workflows"
cat >"$TMP/bad/.github/workflows/ci.yml" <<'EOF'
name: ci
on:
  pull_request:
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - run: ./deploy.sh
        env:
          TOKEN: ${{ secrets.DEPLOY_TOKEN }}
EOF
out=$(run_in "$TMP/bad")
check "unconditional secrets consumer fails" 1 $? "job 'deploy' consumes secrets unconditionally" "$out"

# PUSH-ONLY: secrets fine, ignored by the gate.
mkdir -p "$TMP/pushonly/.github/workflows"
cat >"$TMP/pushonly/.github/workflows/deploy.yml" <<'EOF'
name: deploy
on:
  push:
    branches: [master]
jobs:
  ship:
    runs-on: ubuntu-latest
    steps:
      - run: ./ship.sh
        env:
          TOKEN: ${{ secrets.DEPLOY_TOKEN }}
EOF
out=$(run_in "$TMP/pushonly")
check "push-only workflow ignored" 0 $? 'skip-gated' "$out"

# SCHEDULE-ONLY: ignored too.
mkdir -p "$TMP/cron/.github/workflows"
cat >"$TMP/cron/.github/workflows/nightly.yml" <<'EOF'
name: nightly
on:
  schedule:
    - cron: "0 3 * * *"
jobs:
  run:
    runs-on: ubuntu-latest
    steps:
      - run: ./n.sh
        env:
          TOKEN: ${{ secrets.DEPLOY_TOKEN }}
EOF
out=$(run_in "$TMP/cron")
check "schedule-only workflow ignored" 0 $? 'skip-gated' "$out"

# Architecture-gate probe OUTPUT semantics, executed for real (m16.3):
# extract the probe run body from ci.yml and run it with a fake GITHUB_OUTPUT.
REPO="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROBE_RUN="$(awk '/id: probe/{flag=1} flag && /run: \|/{grab=1; next} grab && /^          /{print; next} grab{exit}' "$REPO/.github/workflows/ci.yml")"
if [ -z "$PROBE_RUN" ]; then
  echo "FAIL probe-extract: could not extract the probe run body from ci.yml"
  failures=$((failures + 1))
else
  for want in true false; do
    OUTFILE="$TMP/gh-output-$want.txt"
    : >"$OUTFILE"
    if [ "$want" = true ]; then KEYVAL="some-key-material"; else KEYVAL=""; fi
    out=$(SSH_PRIVATE_KEY="$KEYVAL" GITHUB_OUTPUT="$OUTFILE" bash -c "$PROBE_RUN" 2>&1)
    got="$(grep -o 'has_key=.*' "$OUTFILE" | head -1)"
    if [ "$got" = "has_key=$want" ]; then
      echo "ok   probe semantics (secret $([ "$want" = true ] && echo present || echo absent) → $got)"
    else
      echo "FAIL probe semantics: want has_key=$want, got '$got'"
      printf '%s\n' "$out" | head -4
      failures=$((failures + 1))
    fi
  done
fi

if [ "$failures" -gt 0 ]; then
  echo "pr-secrets self-test: $failures FAILURE(S)"
  exit 1
fi
echo "pr-secrets self-test: all cases green"
