# Go vendor hash for the templGen derivation in flake.nix. Unrelated to
# this repo's own go.sum files; update from the nix build .#templGen error
# message when templ's dependencies change.
#
# 2026-11-14: real hash recorded from the build's own mismatch report; the
# placeholder broke the devShell (templGen is a devShell package) and with it
# every nix-backed gate.
"sha256-MMAoHofeZzGNFsjxY4nrPRyaiKOk4fLYY+kIVmdAL3g="
