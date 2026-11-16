// position-area computed-style probe runner (the 2026-11-15 popover
// investigation's reusable form; see probe.html for the mechanism).
//
// Usage (from the repo root, any Chromium via the e2e battery's browser):
//   E2E_BROWSER_PATH=/path/to/chromium node e2e/probes/position-area-probe/probe.mjs \
//     'bottom span-inline-start' 'bottom span-x-start' 'bottom x-start'
//
// Prints one `spelling -> computed position-area` line per argument:
// the accepted spellings echo verbatim, rejected ones come back `none`.
import { chromium } from "../../node_modules/playwright/index.mjs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const spellings = process.argv.slice(2);
if (spellings.length === 0) {
  console.error(
    "usage: node probe.mjs '<position-area spelling>' [<more spellings>...]",
  );
  process.exit(2);
}

const browserPath = process.env.E2E_BROWSER_PATH;
if (!browserPath) {
  console.error("E2E_BROWSER_PATH must point at a Chromium binary");
  process.exit(2);
}

const page = join(
  dirname(fileURLToPath(import.meta.url)),
  "probe.html",
);

const browser = await chromium.launch({
  executablePath: browserPath,
  headless: true,
});
try {
  const context = await browser.newContext();
  const p = await context.newPage();
  await p.goto(`file://${page}`);

  for (const spelling of spellings) {
    const computed = await p.evaluate((v) => window.setArea(v), spelling);
    const verdict = computed === "none" ? "REJECTED" : "accepted";
    console.log(`${spelling} -> ${computed} [${verdict}]`);
  }
} finally {
  await browser.close();
}
