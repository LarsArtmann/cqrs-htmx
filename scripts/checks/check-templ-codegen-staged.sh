#!/usr/bin/env bash
# check-templ-codegen-staged.sh — pre-commit codegen-drift gate for staged
# .templ / _templ.go edits (master plan S48 / 08-05 plan D03).
#
# Closes the formatter→generate chain at commit time: buildflow's templ steps
# are deliberately excluded (its bundled templ binary regenerates with
# directory-prefixed FileName: fields — see .buildflow.yml), so nothing in the
# default hook flow guarantees that a staged .templ edit ships with its
# regenerated _templ.go. This gate regenerates (module-dir, the canonical bare
# FileName: form — same command as `nix run .#check-codegen`) and fails when
# the working-tree _templ.go differs from the staged one.
#
# Fast-path: commits with no staged .templ/_templ.go file cost one git call.
# Direct hand-edits to a committed _templ.go are caught too (the regen
# overwrites the hand edit; the diff vs the staged blob goes red).
#
# Fixture self-test: scripts/selftests/test-check-templ-codegen-staged.sh
# (stubbed templ/gofmt; 4 cases).
set -uo pipefail

ROOT="$(git rev-parse --show-toplevel)" || exit 1
cd "$ROOT" || exit 1

staged="$(git diff --cached --name-only 2>/dev/null || true)"
[ -z "$staged" ] && {
  echo "templ-codegen-staged: nothing staged — skip"
  exit 0
}

templ_files="$(printf '%s\n' "$staged" | grep -E '\.templ$|_templ\.go$' || true)"
if [ -z "$templ_files" ]; then
  echo "templ-codegen-staged: no staged .templ/_templ.go changes — skip"
  exit 0
fi

mods="$(printf '%s\n' "$templ_files" | cut -d/ -f1 | sort -u | grep -E '^(adminui|loginpage|dashboardui)$' || true)"
if [ -z "$mods" ]; then
  # .templ files outside the three committed-codegen modules (none today, but
  # the guard keeps the gate honest if one appears).
  echo "templ-codegen-staged: staged templ files are outside adminui/loginpage/dashboardui — skip"
  exit 0
fi

if ! command -v templ >/dev/null 2>&1; then
  echo "templ-codegen-staged: FAIL — templ not on PATH (run inside 'nix develop'; pkgs.templ is the pinned version)." >&2
  exit 1
fi

fail=0
for mod in $mods; do
  if ! (
    cd "$mod" || exit 1
    GOWORK=off templ generate || exit 1
    gofmt -w ./*_templ.go || exit 1
  ); then
    echo "templ-codegen-staged: FAIL — templ generate errored in $mod (see above)." >&2
    fail=1
    continue
  fi
  drift="$(git diff --name-only -- "$mod" | grep '_templ\.go$' || true)"
  if [ -n "$drift" ]; then
    echo "templ-codegen-staged: FAIL — codegen drift in $mod (staged _templ.go does not match the .templ sources):" >&2
    printf '      %s\n' "$drift" >&2
    echo "      fix: run 'nix run .#gen' (module-dir generation is canonical) and re-stage the _templ.go files." >&2
    fail=1
  else
    echo "templ-codegen-staged: $mod — no drift"
  fi
done

exit "$fail"
