#!/usr/bin/env bash
# replace-exemption.sh — single source of truth for the "local replace
# satisfies this require" exemption rule shared by the version gates.
#
# Rule: a require of <module_path> recorded in <module_dir>/go.mod is EXEMPT
# from tag-existence and drift checks when that go.mod carries a local
# `replace <module_path> => ...` — the replaced source wins at build time, so
# the recorded version string is cosmetic and cannot drift against siblings.
#
# Consumers: scripts/check-version-drift.sh (drift leg),
# scripts/check-release-train.sh (tag-existence leg). Keep the regex
# byte-identical across the rule's consumers by importing it here.

# replace_exemption_applies <module_dir> <module_path>
# Returns 0 (true) when a local replace directive covers module_path.
replace_exemption_applies() {
  local moddir="$1"
  local mod_path="$2"
  grep -qE "^[[:space:]]*replace[[:space:]]+${mod_path}([[:space:]]|v[0-9])" \
    "$moddir/go.mod" 2>/dev/null
}
