# Go vendor hash for the benchstat derivation in flake.nix. Unrelated to this
# repo's own go.sum files; update from the `nix build .#benchstat` error
# message when benchstat's dependencies change.
#
# 2026-11-14: real hash recorded from the build's own mismatch report; the
# placeholder broke the devShell (benchstat is a devShell package) and with it
# every nix-backed gate and the pre-commit hook.
"sha256-AZx9tPzsPvjc5kpmiBa6eYKtrw0hczYi0sbcd/lkiiA="
