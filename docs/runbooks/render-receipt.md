# Runbook: HTML report render receipts (fullPage PNG)

> The `.#render-receipt` flake app is the canonical path (§f48 codified,
> 2026-11-21). This file records the recipe it mechanizes so the
> hand-rolled fallback stays possible when the app cannot run.

## Why this exists

Every HTML-series audit deliverable (`docs/research/*.html`) ships with a
full-page receipt PNG so a reader can eyeball the artifact without opening
it, and so the replay ritual can diff renders. Rendering headless chromium
on NixOS has two traps (gotcha 66 family):

1. **The browser must be the Playwright-managed build.** A system/nix
   chromium skews OLDER than the driver and fails every `newContext`
   (hit live 2026-11-18: nix chromium 157 vs the driver's expected 159).
2. **The managed build needs a nixpkgs lib forest.** The headless shell
   dies on `libglib-2.0.so.0` (then libnss3, libcups, ...) unless
   `LD_LIBRARY_PATH` carries the forest — one shared derivation,
   `playwrightBrowserLibs` in `flake.nix`, exported as
   `E2E_BROWSER_LIBS` for chromium and `E2E_FIREFOX_LIBS` for firefox.

## Canonical: the flake app

```sh
nix run .#render-receipt -- docs/research/<report>.html docs/research/<report>.render.png [width]
```

- Width defaults to 1440 (the fleet-audit deliverable's viewport).
- Prints the rendered byte size; **exits 3** and prints the quantize
  command when the PNG exceeds the 1 MiB binary-guard limit (the
  `checks` job's binary guard reddened on a 1.6 MB render PNG live,
  2026-11-21).

## Binary guard: quantize before committing

The committed PNG must stay under 1 MiB. The proven compressor
(256-color palette, NO dither — the dither variant compressed WORSE,
2.4 MB vs the source 1.6 MB, measured 2026-11-21):

```sh
nix shell nixpkgs#imagemagick -c magick <output.png> -colors 256 -dither None <output.png>
```

Verify the result stays under 1,048,576 bytes and re-eyeball it at 100%
zoom: palette banding on gradients is expected and acceptable; unreadable
text is not (then render at a higher width and re-quantize).

## Hand-rolled fallback (when the flake app cannot run)

The app wraps exactly this recipe:

```sh
CHROME_BIN="$(find /mnt/buildcache/playwright ~/.cache/ms-playwright \
  -maxdepth 4 -path '*/chromium-*/chrome-linux*/chrome' -type f 2>/dev/null | sort -V | tail -1)"
LIBS="$(nix build --print-out-paths --no-link \
  nixpkgs#gtk3 nixpkgs#cups.lib nixpkgs#expat nixpkgs#libgbm nixpkgs#udev \
  nixpkgs#alsa-lib nixpkgs#dbus.lib nixpkgs#dbus-glib nixpkgs#atk \
  nixpkgs#at-spi2-atk nixpkgs#cairo nixpkgs#pango.out nixpkgs#gdk-pixbuf \
  nixpkgs#glib.out nixpkgs#freetype nixpkgs#fontconfig.lib \
  nixpkgs#libxkbcommon nixpkgs#libdrm nixpkgs#mesa nixpkgs#libX11 \
  nixpkgs#libXcomposite nixpkgs#libXdamage nixpkgs#libXext \
  nixpkgs#libXfixes nixpkgs#libXrandr nixpkgs#libXt nixpkgs#libxcb \
  nixpkgs#libXi nixpkgs#nss nixpkgs#nspr nixpkgs#libXcursor \
  nixpkgs#libXrender nixpkgs#stdenv.cc.cc.lib \
  | sed 's|$|/lib|' | paste -sd: -)"
cd e2e   # the script resolves "playwright-core" against e2e/node_modules
E2E_BROWSER_PATH="$CHROME_BIN" E2E_BROWSER_LIBS="$LIBS" \
  node render-receipt.mjs <input.html> <output.png> [width]
```

Pass ABSOLUTE input/output paths when driving the script directly after a
`cd` — arguments resolve against the process CWD (the app realpath-ifies
them before entering `e2e/`, hit live 2026-11-21 as `ERR_FILE_NOT_FOUND`).

## Replay discipline

The fleet-audit deliverable's verify block freezes as the audit-time
record; a replay re-renders the PNG, re-runs the block verbatim, and moves
the replay receipt's date (docs/review/libs/FORMAT.md carries the policy).
Re-render, quantize, commit — never hand-edit the receipt PNG's dimensions.
