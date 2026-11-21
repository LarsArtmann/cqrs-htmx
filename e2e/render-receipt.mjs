// render-receipt.mjs — render an HTML report to a fullPage receipt PNG.
//
// The `.#render-receipt` flake app wraps this script with the
// Playwright-managed chromium + the nixpkgs lib forest (E2E_BROWSER_PATH /
// E2E_BROWSER_LIBS). Run standalone only from e2e/ (bare "playwright-core"
// specifier resolves against e2e/node_modules) with both env vars set —
// the runbook in docs/runbooks/ carries the hand-rolled recipe.
//
// Exit 0 on success, 2 on usage error, 3 when the PNG exceeds the 1 MiB
// binary-guard limit (the quantize recipe prints on stderr).
import { chromium } from "playwright-core";
import { statSync } from "node:fs";
import { resolve } from "node:path";

const [input, output, widthArg] = process.argv.slice(2);
if (!input || !output) {
  console.error("usage: render-receipt.mjs <input.html> <output.png> [width]");
  process.exit(2);
}

const width = Number(widthArg ?? 1440);
if (!Number.isFinite(width) || width < 320) {
  console.error(`invalid width: ${widthArg ?? "(missing)"}`);
  process.exit(2);
}

const exe = process.env.E2E_BROWSER_PATH;
const libs = process.env.E2E_BROWSER_LIBS;
const browser = await chromium.launch({
  executablePath: exe || undefined,
  ...(libs
    ? {
        env: {
          ...process.env,
          LD_LIBRARY_PATH: [libs, process.env.LD_LIBRARY_PATH].filter(Boolean).join(":"),
        },
      }
    : {}),
});

try {
  const page = await browser.newPage({ viewport: { width, height: 900 }, deviceScaleFactor: 1 });
  const target = resolve(input);
  await page.goto(`file://${target}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  await page.screenshot({ path: output, fullPage: true });

  const bytes = statSync(output).size;
  console.log(`rendered ${output} (${width}w, fullPage, ${bytes} bytes)`);

  if (bytes > 1048576) {
    console.warn(`WARN: ${bytes} bytes exceeds the 1 MiB binary-guard limit — quantize before committing:`);
    console.warn(`  python3 -c "from PIL import Image; im = Image.open(${JSON.stringify(output)}).convert('RGB').quantize(256, dither=Image.Dither.NONE); im.save(${JSON.stringify(output)})"`);
    process.exit(3);
  }
} finally {
  await browser.close();
}
