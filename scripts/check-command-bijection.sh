#!/usr/bin/env bash
# check-command-bijection.sh — prove the 1:1 bijection between identity-model
# command type constants and usermgmt RegisterTyped dispatch registrations.
#
# Evidence task M11 of the command-utilization audit
# (docs/research/2026-09-17_go-cqrs-lite-command-deep-dive.html).
# It asserts BOTH directions:
#   1. every Cmd* constant in identity-model/constants.go is registered
#      exactly once via command.RegisterTyped in usermgmt/es_*_dispatch.go
#   2. every RegisterTyped call site references a defined Cmd* constant
#
# usermgmt re-exports the constants as lowercase vars (cmdRegisterUser etc.),
# so registration identifiers are normalized by stripping the prefix.

set -uo pipefail

cd "$(dirname "$0")/.."

constants_file="identity-model/constants.go"
dispatch_files=(usermgmt/es_dispatch.go usermgmt/es_tenant_dispatch.go usermgmt/es_membership_dispatch.go usermgmt/es_bot_dispatch.go)

# 1. Collect defined constants: CmdFoo -> "RegisterUser"-ish suffix
mapfile -t defined < <(grep -oE '^\s+Cmd[A-Za-z]+\s' "$constants_file" | tr -d ' \t' | sort)
if [ "${#defined[@]}" -eq 0 ]; then
	echo "FAIL: no Cmd* constants found in $constants_file"
	exit 1
fi

# 2. Collect registered identifiers at RegisterTyped call sites: cmdFoo,
registered=()
for f in "${dispatch_files[@]}"; do
	if [ ! -f "$f" ]; then
		echo "FAIL: dispatch file missing: $f"
		exit 1
	fi
	# The identifier is on the line after "command.RegisterTyped(" — the
	# first argument line "dispatcher, cmdFoo,".
	while IFS= read -r id; do
		[ -n "$id" ] && registered+=("$id")
	done < <(grep -A1 'command\.RegisterTyped(' "$f" | grep -oE 'cmd[A-Za-z]+' | sort -u)
done

norm() { printf '%s' "$1" | sed -e 's/^Cmd//' -e 's/^cmd//'; }

fail=0

# Direction 1: every defined constant is registered.
for c in "${defined[@]}"; do
	found=0
	for r in "${registered[@]}"; do
		if [ "$(norm "$c")" = "$(norm "$r")" ]; then found=1; break; fi
	done
	if [ "$found" -eq 0 ]; then
		echo "FAIL: constant $c (identity-model) has NO RegisterTyped registration"
		fail=1
	fi
done

# Direction 2: every registration references a defined constant.
for r in "${registered[@]}"; do
	found=0
	for c in "${defined[@]}"; do
		if [ "$(norm "$c")" = "$(norm "$r")" ]; then found=1; break; fi
	done
	if [ "$found" -eq 0 ]; then
		echo "FAIL: registration $r (usermgmt) references NO identity-model constant"
		fail=1
	fi
done

dups=$(printf '%s\n' "${registered[@]}" | sort | uniq -d)
if [ -n "$dups" ]; then
	echo "FAIL: duplicate registrations:"
	echo "$dups"
	fail=1
fi

echo "constants: ${#defined[@]}  registrations: ${#registered[@]}"
if [ "$fail" -ne 0 ]; then
	exit 1
fi
echo "OK: bijection proven (${#defined[@]}/${#registered[@]}, both directions)"
