# Go vendor hash for the templGen derivation in flake.nix. Unrelated to
# this repo's own go.sum files; update from the nix build .#templGen error
# message when templ's dependencies change.
#
# 2026-11-14: real hash recorded from the build's own mismatch report (the
# v0.3.1100 vendor tree); the placeholder broke the devShell (templGen is a
# devShell package) and with it every nix-backed gate.
# 2026-11-19: hash UPDATED to the true v0.3.1118 vendor tree via the
# fakeHash probe. The 1100-era value MMAoHofeZzGNFsjxY4nrPRyaiKOk4fLYY+
# kIVmdAL3g= did NOT mismatch against the changed templ go.mod because the
# go-modules fixed-output was cache-satisfied with the old vendor content
# (the same substitution poison as templ-hash.nix — the compile then failed
# on "inconsistent vendoring", which is how the layer surfaced).
"sha256-SI3Y7eqw+sLvBmRi7r4Oc30ovIKvFRGxdXF8fyKog9g="
