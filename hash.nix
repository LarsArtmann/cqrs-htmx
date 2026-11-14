# Pinned source hash for the golang.org/x/perf fetchgit input in flake.nix
# (benchstat). Update together with the flake.nix `rev` when bumping benchstat;
# nix build .#benchstat prints the new value on mismatch.
#
# 2026-11-14: real hash recorded from the build's own mismatch report. The
# placeholder here broke the devShell (and therefore the pre-commit hook's
# nix-fmt step) exactly like the goPkg placeholder did.
"sha256-CimaQbwjQ5SMl/VTzuMeSciOp7aSomGbT/iyEsguOCg="
