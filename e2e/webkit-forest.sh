#!/usr/bin/env bash
# webkit-forest.sh — build the NixOS LD_LIBRARY_PATH forest for the webkit
# smoke leg (T28/F203, 2026-11-24). Reproducible: the nixpkgs pin is read
# from flake.lock at runtime, so a re-run rebuilds the same forest until
# the flake bump — after any bump, re-verify with ONE smoke spec before
# anything wider (gotcha 66: a poisoned launch env fails in seconds, not
# in a 68-test battery).
#
# WHY this forest exists (each bullet is a live finding from 2026-11-24):
#   1. The shared playwright-browser-libs forest (chromium + firefox)
#      lacks ~40 libs the WPE MiniBrowser links (gstreamer stack, icu74,
#      soup3, webp, enchant, ...). The WPE bundle carries WPEWebKit +
#      libjxl + libsoup itself; everything else comes from nixpkgs here.
#   2. Soname traps in the pinned nixpkgs: `libjpeg` is the 62-ABI build
#      (webkit-2388 needs libjpeg.so.8 → use the `libjpeg8` attr) and
#      `opus` builds static-only (the shared lib lives under `libopus`).
#   3. harfbuzz dropped the hb-icu API upstream (7.0) and nixpkgs never
#      shipped libharfbuzz-icu — but webkit-2388 hard-needs it: the
#      WPEWebProcess died with `undefined symbol: hb_icu_script_to_script`.
#      Ubuntu re-added the lib (libharfbuzz-icu0 10.2.0), so this script
#      extracts that .deb into the forest (plus pkgs.icu76 for its
#      libicuuc.so.76 dependency).
#   4. The pin's libxml2 is soname 16; webkit-2388 needs soname 2. The
#      shim symlink works today (launch+tests green, only "no version
#      information available" warnings) — treat it as the FIRST suspect
#      on any future XML-related webkit red.
#
# Usage:
#   bash e2e/webkit-forest.sh [target-dir]     # default /tmp/webkit-forest
#   E2E_WEBKIT_PATH=/mnt/buildcache/playwright/webkit-2388/pw_run.sh \
#     E2E_WEBKIT_LIBS=<target>/lib \
#     ./node_modules/.bin/playwright test --project=webkit
#
# The webkit project in playwright.config.ts auto-detects the cache layout
# and bypasses BOTH launch wrappers (pw_run.sh and the zip bundle's own
# MiniBrowser script) because each rebuilds LD_LIBRARY_PATH for the browser
# child and would clobber this injection — see the config comment.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TARGET="${1:-/tmp/webkit-forest}"
PIN_REV="$(python3 -c "import json;print(json.load(open('$REPO_ROOT/flake.lock'))['nodes']['nixpkgs']['locked']['rev'])")"
# The shared chromium+firefox forest (built by the flake's playwrightBrowserLibs):
# webkit reuses its GTK/ATK/cairo/... entries — symlink it LAST so its
# chromium-proven sonames win any name collision.
EXISTING_FOREST="/nix/store/zw1prm0dbqwxiaz2v242lav89g4m7a2i-playwright-browser-libs/lib"

mkdir -p "$TARGET/lib"

DIRS="$(nix eval --raw --impure --expr "
  let
    pkgs = import (builtins.getFlake \"github:NixOS/nixpkgs/$PIN_REV\") { };
    gst = pkgs.gst_all_1;
    libs = [
      pkgs.glib gst.gstreamer gst.gst-plugins-base gst.gst-plugins-good gst.gst-plugins-bad
      pkgs.icu74 pkgs.icu76 pkgs.libwebp pkgs.libjpeg8 pkgs.libpng pkgs.zlib pkgs.harfbuzz
      pkgs.hyphen pkgs.libmanette pkgs.nghttp2 pkgs.libpsl pkgs.wayland pkgs.libavif
      pkgs.openssl pkgs.enchant pkgs.libepoxy pkgs.libevent pkgs.sqlite pkgs.lcms2
      pkgs.libopus pkgs.libxslt pkgs.libxml2
    ];
  in
  builtins.concatStringsSep \"\\n\" (map (p: builtins.toString (pkgs.lib.getLib p)) libs)
")"

for p in $DIRS; do
  if [ -d "$p/lib" ]; then
    ln -sf "$p"/lib/*.so* "$TARGET/lib/" || true
  fi
done
# The shared chromium+firefox forest last (name-collision winner, see above).
ln -sf "$EXISTING_FOREST"/*.so* "$TARGET/lib/" || true

# Soname shims + the Ubuntu hb-icu lib (see WHY 3 and 4 above).
ln -sf libxml2.so.16 "$TARGET/lib/libxml2.so.2"
if [ ! -f "$TARGET/lib/libharfbuzz-icu.so.0" ]; then
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  curl -fsSL -o "$TMP/hbicu.deb" \
    http://archive.ubuntu.com/ubuntu/pool/main/h/harfbuzz/libharfbuzz-icu0_10.2.0-1_amd64.deb
  (cd "$TMP" && ar x hbicu.deb && tar xf data.tar.zst)
  cp -P "$TMP"/usr/lib/x86_64-linux-gnu/libharfbuzz-icu.so* "$TARGET/lib/"
fi

echo "==> webkit forest ready: $TARGET/lib ($(ls "$TARGET/lib" | wc -l) entries)"
echo "    launch with: E2E_WEBKIT_LIBS=$TARGET/lib (plus E2E_WEBKIT_PATH — see e2e/README.md)"
