# Pinned source hash for the templ fetchgit input in flake.nix (templGen,
# the CI-canonical v0.3.1100 generator). Update together with the flake rev
# when bumping; nix build .#templGen prints the new value on mismatch.
#
# 2026-11-14: real hash recorded from the build's own mismatch report; the
# placeholder broke the devShell and therefore every nix-backed gate.
"sha256-QViQAMtCqaXS4GodlSCW/nE4Sox+EF49sbb7h9RBIS0="
