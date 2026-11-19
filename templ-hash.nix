# Pinned source hash for the templ fetchgit input in flake.nix (templGen,
# the CI-canonical v0.3.1118 generator — note: `templ version` still prints
# v0.3.1100 because upstream's embedded `.version` file is not bumped at
# release time; the stamp form is likewise unchanged, verified updates=0).
# Update together with the flake rev when bumping; nix build .#templGen
# prints the new value on mismatch.
#
# 2026-11-14: real hash recorded from the build's own mismatch report (the
# v0.3.1100 tree); the placeholder broke the devShell and therefore every
# nix-backed gate.
# 2026-11-19: hash UPDATED to the true v0.3.1118 tree (fresh
# nix-prefetch-git). The 1100-era value QViQAMtCqaXS4GodlSCW/nE4Sox+
# EF49sbb7h9RBIS0= validated against rev v0.3.1118 WITHOUT a mismatch only
# because the fetchgit output was satisfied from cache/substitution instead
# of a fresh fetch — the poisoned path shipped the 1100 tree under the 1118
# rev and silently voided that day's `nix run .#gen` verification (gotcha
# 10 carries the mechanism).
"sha256-4ARBfwLD37FSCC3pgYWomsXRThl8NkHLte8awD0/Ddo="
