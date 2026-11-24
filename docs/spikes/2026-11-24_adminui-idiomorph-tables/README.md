# idiomorph-on-tables spike (T24 F176)

Scratch experiment for the adminui live-updates task: does swapping the
users table via idiomorph (DOM-morphing) instead of `innerHTML` buy enough
(churn reduction + focus retention) to justify a new runtime dependency?

## Harness

- `index.html` / `spike.js` — 200-row users-shaped table, 50 swap rounds,
  one mutated cell per round, a per-row button focused before each swap.
  Node churn via MutationObserver (flushed before reading), focus retention
  = "same button (or id-identical replacement) still focused".
- `spike-runner.mjs` — Playwright drives it headless via
  `E2E_BROWSER_PATH=<chromium> node spike-runner.mjs` with a static server
  (`python3 -m http.server <port>`, port edited into the runner).
- `idiomorph.esm.js` — vendored copy of idiomorph 0.7.3 for THIS spike only;
  deliberately NOT an adminui dependency.

## Result (chromium 158, shared machine — advisory)

See `RESULTS.json`: innerHTML 20,000 nodes added + 20,000 removed over 50
rounds, focus kept 0/50; idiomorph 0 childList mutations (in-place text
patching), focus kept 50/50; swap cost ~0.9ms vs ~6.9ms per swap.

Verdict recorded in `adminui/README.md` § ADR-lite: closed — NOT adopted
(focus win is real but the panel never swaps a focused row; dependency +
7x swap cost not justified today).
