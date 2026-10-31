#!/usr/bin/env bash
# check-dependabot-alerts.sh — dependabot-alert watch (round-45 M18, ADVISORY).
#
# Reports open dependabot alerts for the repo via `gh api`. Advisory by
# default: findings never fail the run (the CVE triage is the owner's cadence
# — the go-jose patch wave of 2026-10-2x is the worked example); the gate
# exists so the count is VISIBLE in check-modules output instead of living in
# someone's inbox.
#
# Traps encoded (both hit for real in this repo):
#   - GH auth absent (no gh, no token) → WARNING + exit 0 under CI, exit 3
#     locally (a local "green" that never asked is not a verdict).
#   - Manifest-path trap: alerts can name go.mods that carry no `require` for
#     the flagged module (indirect/SBOM mapping) — the output prints the
#     manifest path so the reader doesn't patch a file with nothing to bump.
#
# Usage: bash scripts/checks/check-dependabot-alerts.sh [--strict]
#   --strict: findings DO fail (for the future blocking promotion).
# Exit: 0 = no alerts (or advisory findings); 1 = --strict and alerts exist;
#       3 = auth unavailable (locally only).
set -uo pipefail

STRICT=0
[ "${1:-}" = "--strict" ] && STRICT=1

REPO="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "check-dependabot-alerts: not a git repo" >&2
  exit 2
}
cd "$REPO" || exit 2

command -v gh >/dev/null 2>&1 || {
  if [ "${CI:-}" = "true" ]; then
    echo "check-dependabot-alerts: WARN (ci) — gh not available; alert watch skipped"
    exit 0
  fi
  echo "check-dependabot-alerts: FAILED — gh not found (exit 3; a local run must actually ask)" >&2
  exit 3
}

api_out="$(gh api "repos/{owner}/{repo}/dependabot/alerts?state=open" 2>&1)"
rc=$?
if [ "$rc" -ne 0 ]; then
  if printf '%s' "$api_out" | grep -q '403\|404\|Bad credentials\|gh auth'; then
    if [ "${CI:-}" = "true" ]; then
      echo "check-dependabot-alerts: WARN (ci) — GitHub API unavailable/unauthorized; alert watch skipped"
      exit 0
    fi
    echo "check-dependabot-alerts: FAILED — gh api rejected (exit 3): ${api_out}" >&2
    exit 3
  fi
  echo "check-dependabot-alerts: FAILED — gh api error (rc=$rc): ${api_out}" >&2
  exit "$rc"
fi

count="$(printf '%s' "$api_out" | jq 'length' 2>/dev/null || echo "?")"
if [ "$count" = "0" ]; then
  echo "check-dependabot-alerts: 0 open dependabot alerts"
  exit 0
fi

echo "check-dependabot-alerts: $count open alert(s):"
printf '%s' "$api_out" | jq -r '.[] | "  [\(.security_advisory.severity)] \(.security_vulnerability.package.name) — manifest: \(.dependency.manifest_path) (fixed in \(.security_vulnerability.vulnerable_version_range // "?") → \(.security_advisory.summary))"' 2>/dev/null
echo "  note: an alert may name a go.mod WITHOUT a direct require (indirect/SBOM mapping) — patch the OWNING module, not a file with nothing to bump"

if [ "$STRICT" -eq 1 ]; then
  echo "check-dependabot-alerts: FAILED — $count open alert(s) (strict mode)" >&2
  exit 1
fi
echo "check-dependabot-alerts: ADVISORY — findings do not block (strict promotion is an owner decision)"
exit 0
